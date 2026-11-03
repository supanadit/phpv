package webserver

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// execFPM builds the php-fpm foreground command for a rendered config. It
// reads the CLI install's php.ini (`-c`) and pins extension_dir so shared
// extensions resolve exactly like they do for `php -S`.
func execFPM(fpmBin, confPath, iniDir, extDir string) *exec.Cmd {
	args := []string{"-F", "-y", confPath}
	if iniDir != "" {
		args = append(args, "-c", iniDir)
	}
	if extDir != "" {
		args = append(args, "-d", "extension_dir="+extDir)
	}
	return exec.Command(fpmBin, args...)
}

// waitSocket polls until the unix socket accepts connections.
func waitSocket(socket string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socket); err == nil {
			c, err := net.Dial("unix", socket)
			if err == nil {
				c.Close()
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("socket %s not ready within %s", filepath.Base(socket), timeout)
}

// asHostsPermission wraps errors.As for *HostsPermissionError.
func asHostsPermission(err error, target **HostsPermissionError) bool {
	return errors.As(err, target)
}
