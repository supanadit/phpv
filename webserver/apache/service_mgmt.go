package apache

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/supanadit/phpv/domain"
)

// httpdBin returns the httpd binary path for the installed version.
func (s *ApacheService) httpdBin() (string, error) {
	installed, version := s.IsInstalled()
	if !installed {
		return "", fmt.Errorf("apache is not installed. Run `phpv apache install <version>` first")
	}
	bin := filepath.Join(s.httpdPrefix(version), "bin", "httpd")
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("httpd binary not found at %s", bin)
	}
	return bin, nil
}

// httpdPidPath returns the PID file path for the httpd process.
func httpdPidPath() string {
	return filepath.Join(runDir(), "httpd.pid")
}

// fpmPidPath returns the PID file path for php-fpm.
func fpmPidPath() string {
	return filepath.Join(runDir(), "php-fpm.pid")
}

// Start launches Apache (and php-fpm for FPM mode). With foreground=true it
// blocks and runs httpd -D FOREGROUND.
func (s *ApacheService) Start(foreground bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("apache is not configured. Run `phpv apache configure` first")
	}
	if err := ensureDirs(); err != nil {
		return err
	}
	bin, err := s.httpdBin()
	if err != nil {
		return err
	}
	conf := httpdConf(cfg.Prefix)

	// Validate the config before starting.
	if err := runCmd(bin, "-t", "-f", conf); err != nil {
		return fmt.Errorf("apache config test failed: %w\nRun `phpv apache configure` to fix configuration", err)
	}

	// Start php-fpm first for FPM mode.
	if domain.ConnectorMode(cfg.Connector) == domain.ConnectorFPM {
		if err := s.startFPM(cfg); err != nil {
			return err
		}
	}

	if foreground {
		fmt.Println("Starting Apache in foreground (Ctrl+C to stop)...")
		cmd := exec.Command(bin, "-D", "FOREGROUND", "-f", conf)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("apache foreground exited: %w", err)
		}
		return nil
	}

	if err := s.stopIfRunning(); err != nil {
		return err
	}
	cmd := exec.Command(bin, "-f", conf)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start apache: %w", err)
	}
	// httpd daemonizes itself; the wrapper exits quickly. Write our PID file.
	if err := os.WriteFile(httpdPidPath(), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		return err
	}
	fmt.Println("✓ Apache started")
	return nil
}

// Stop terminates httpd and php-fpm.
func (s *ApacheService) Stop() error {
	if err := s.stopFPM(); err != nil {
		// not fatal
	}
	if err := s.stopIfRunning(); err != nil {
		return err
	}
	fmt.Println("✓ Apache stopped")
	return nil
}

// Restart reloads Apache and php-fpm.
func (s *ApacheService) Restart() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("apache is not configured. Run `phpv apache configure` first")
	}
	if err := s.stopFPM(); err != nil {
		// ignore
	}
	if domain.ConnectorMode(cfg.Connector) == domain.ConnectorFPM {
		if err := s.startFPM(cfg); err != nil {
			return err
		}
	}
	if err := s.stopIfRunning(); err != nil {
		return err
	}
	bin, err := s.httpdBin()
	if err != nil {
		return err
	}
	conf := filepath.Join(cfg.Prefix, "conf", "httpd.conf")
	cmd := exec.Command(bin, "-d", cfg.Prefix)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart apache: %w", err)
	}
	_ = conf
	if err := os.WriteFile(httpdPidPath(), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		return err
	}
	fmt.Println("✓ Apache restarted")
	return nil
}

// Status reports whether Apache (and FPM) are running.
func (s *ApacheService) Status() (string, error) {
	installed, version := s.IsInstalled()
	if !installed {
		return "not installed", nil
	}
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	httpdRunning := processAlive(httpdPidPath())
	status := fmt.Sprintf("Apache httpd %s: %s", version, runState(httpdRunning))
	if cfg != nil && domain.ConnectorMode(cfg.Connector) == domain.ConnectorFPM {
		fpmRunning := processAlive(fpmPidPath())
		status += fmt.Sprintf("\nPHP-FPM %s: %s", cfg.PHPVersion, runState(fpmRunning))
	}
	return status, nil
}

// restartIfRunning reloads Apache only if it is currently running.
func (s *ApacheService) restartIfRunning() error {
	if processAlive(httpdPidPath()) {
		return s.Restart()
	}
	return nil
}

func runState(alive bool) string {
	if alive {
		return "running"
	}
	return "stopped"
}

// startFPM launches php-fpm for the configured PHP version using the generated
// pool config.
func (s *ApacheService) startFPM(cfg *domain.WebserverConfig) error {
	fpmBin := filepath.Join(s.siloSvc.PackagePrefix("php", cfg.PHPVersion), "bin", "php-fpm")
	if _, err := os.Stat(fpmBin); err != nil {
		return fmt.Errorf("php-fpm binary not found for PHP %s at %s", cfg.PHPVersion, fpmBin)
	}
	if processAlive(fpmPidPath()) {
		return nil
	}
	if err := s.stopFPM(); err != nil {
		// ignore
	}
	cmd := exec.Command(fpmBin, "--fpm-config", fpmConf(), "--pid", fpmPidPath())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start php-fpm: %w", err)
	}
	return nil
}

// stopFPM stops php-fpm via its PID file.
func (s *ApacheService) stopFPM() error {
	if err := killPID(fpmPidPath()); err != nil {
		return err
	}
	return nil
}

// stopIfRunning stops httpd if a live PID file exists.
func (s *ApacheService) stopIfRunning() error {
	return killPID(httpdPidPath())
}

func killPID(pidPath string) error {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil
	}
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(os.Interrupt)
	}
	_ = os.Remove(pidPath)
	return nil
}

func processAlive(pidPath string) bool {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, out)
	}
	return nil
}
