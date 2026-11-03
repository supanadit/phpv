package webserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FPMConfig is everything needed to render and run one php-fpm master for
// an instance. The CLI php.ini and extension_dir are supplied on the command
// line (see execFPM), so FPM runs with exactly the CLI's configuration.
type FPMConfig struct {
	Binary     string // php-fpm binary
	PHPVersion string
	Socket     string // unix socket path
	PIDFile    string
	ErrorLog   string
	SlowLog    string
	StateDir   string
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
	confPath := filepath.Join(c.StateDir, "php-fpm.conf")
	if err := os.WriteFile(confPath, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write php-fpm.conf: %w", err)
	}
	return confPath, nil
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
