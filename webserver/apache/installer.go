package apache

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/system"
)

// checkBuildDeps ensures the build tools required to compile Apache from
// source are present (falling back to a system package install when possible).
func (s *ApacheService) checkBuildDeps() error {
	names := []string{"gcc", "g++", "make", "pkg-config", "libtool", "autoconf", "perl"}
	result, err := s.systemSvc.CheckBuildTools(names)
	if err != nil {
		return fmt.Errorf("build tools check: %w", err)
	}
	if len(result.Missing) == 0 {
		return nil
	}
	fmt.Println("\nMissing build tools for Apache:")
	for _, p := range result.Missing {
		fmt.Printf("  ✗ %s\n", p.SystemName)
	}
	if result.Distro.PM == "unknown" {
		return fmt.Errorf("missing build tools: %s", strings.Join(namesOf(result.Missing), ", "))
	}
	fmt.Println("Install via package manager? [Y/n] ")
	answer := readLine()
	if answer == "n" || answer == "no" {
		return fmt.Errorf("Apache build requires: %s", strings.Join(namesOf(result.Missing), ", "))
	}
	if err := s.systemSvc.Install(result.Missing); err != nil {
		return fmt.Errorf("install build tools: %w", err)
	}
	return nil
}

func namesOf(pkgs []system.Package) []string {
	var out []string
	for _, p := range pkgs {
		out = append(out, p.SystemName)
	}
	return out
}

func readLine() string {
	var line string
	_, _ = fmt.Scanln(&line)
	return line
}

// postInstall prepares the phpv-managed Apache layout after httpd is built:
// it creates the managed directories and writes the base apache.json so the
// rest of the tooling knows where httpd lives.
func (s *ApacheService) postInstall(version, prefix string) error {
	if err := ensureDirs(); err != nil {
		return err
	}
	cfg := &domain.WebserverConfig{
		ServerType:  "apache",
		Version:     version,
		Prefix:      prefix,
		Connector:   string(domain.ConnectorNone),
		MPM:         "event",
		ListenPort:  8080,
		User:        os.Getenv("USER"),
		Group:       primaryGroup(),
		FPMBasePort: 9000,
		Vhosts:      []domain.Vhost{},
	}
	return saveConfig(cfg)
}

// primaryGroup returns the primary group of the current user, best-effort.
func primaryGroup() string {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/self/status"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "Gid:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						return fields[1]
					}
				}
			}
		}
	}
	// Fall back to a common default derived from the user name.
	return os.Getenv("USER")
}
