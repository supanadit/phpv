package webserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FPMConfig is everything needed to render and run one php-fpm master for
// an instance.
type FPMConfig struct {
	Binary       string // php-fpm binary
	PHPVersion   string
	Socket       string // unix socket path
	PIDFile      string
	ErrorLog     string
	SlowLog      string
	StateDir     string
	// CLI install integration: FPM loads the CLI's shared extensions.
	PHPIniPath    string
	ExtensionDir  string
	Extensions    []string // extension=<name>.so lines from the CLI php.ini
	// PHP 5.3-era FPM lacks some pool directives; guard by version.
	Legacy bool // PHP < 7.0
}

// RenderFPMConf writes php-fpm.conf (global + pool) into stateDir and
// returns its path. The pool listens on the instance's unix socket and
// loads the CLI install's shared extensions via php_admin_value.
func RenderFPMConf(c FPMConfig) (string, error) {
	if err := os.MkdirAll(c.StateDir, 0o755); err != nil {
		return "", fmt.Errorf("create fpm state dir: %w", err)
	}
	var b strings.Builder
	b.WriteString("[global]\n")
	fmt.Fprintf(&b, "pid = %s\n", c.PIDFile)
	fmt.Fprintf(&b, "error_log = %s\n", c.ErrorLog)
	b.WriteString("daemonize = no\n\n")
	b.WriteString("[phpv]\n")
	fmt.Fprintf(&b, "listen = %s\n", c.Socket)
	b.WriteString("listen.mode = 0666\n")
	b.WriteString("pm = dynamic\n")
	b.WriteString("pm.max_children = 10\n")
	b.WriteString("pm.start_servers = 2\n")
	b.WriteString("pm.min_spare_servers = 1\n")
	b.WriteString("pm.max_spare_servers = 3\n")
	if !c.Legacy {
		fmt.Fprintf(&b, "slowlog = %s\n", c.SlowLog)
		b.WriteString("request_terminate_timeout = 120s\n")
	}
	fmt.Fprintf(&b, "php_admin_value[error_log] = %s\n", filepath.Join(c.StateDir, "php-error.log"))
	b.WriteString("php_admin_flag[log_errors] = on\n")
	b.WriteString("php_admin_value[error_reporting] = E_ALL\n")
	if c.ExtensionDir != "" {
		fmt.Fprintf(&b, "php_admin_value[extension_dir] = %s\n", c.ExtensionDir)
	}
	for _, ext := range c.Extensions {
		fmt.Fprintf(&b, "php_admin_value[extension] = %s\n", ext)
	}
	if c.PHPIniPath != "" {
		// Pull the CLI INI's plain settings (memory_limit, timezone, ...)
		// into the pool so FPM behaves like the CLI.
		for _, line := range iniValueLines(c.PHPIniPath) {
			b.WriteString(line + "\n")
		}
	}
	confPath := filepath.Join(c.StateDir, "php-fpm.conf")
	if err := os.WriteFile(confPath, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write php-fpm.conf: %w", err)
	}
	return confPath, nil
}

// iniValueLines extracts simple `key = value` ini settings (excluding
// extension lines and comments) from a php.ini, re-emitted as
// php_admin_value/php_admin_flag lines suitable for an FPM pool.
func iniValueLines(iniPath string) []string {
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return nil
	}
	var out []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || val == "" {
			continue
		}
		if key == "extension" || key == "zend_extension" || key == "extension_dir" {
			continue // handled explicitly by RenderFPMConf
		}
		// Only accept well-formed simple values; skip arrays/complex syntax.
		if strings.ContainsAny(val, "{}$") {
			continue
		}
		switch strings.ToLower(val) {
		case "on", "true", "yes":
			out = append(out, fmt.Sprintf("php_admin_flag[%s] = on", key))
		case "off", "false", "no":
			out = append(out, fmt.Sprintf("php_admin_flag[%s] = off", key))
		default:
			out = append(out, fmt.Sprintf("php_admin_value[%s] = %s", key, val))
		}
	}
	return out
}

// StopFPM gracefully stops a php-fpm master via its pid file, falling back
// to SIGTERM on the recorded pid.
func StopFPM(pidFile string, pid int) error {
	if data, err := os.ReadFile(pidFile); err == nil {
		if p := parseInt(string(data)); p > 0 {
			pid = p
		}
	}
	return SignalTerm(pid)
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
