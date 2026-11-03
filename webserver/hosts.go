package webserver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HostsEntry is one managed line in /etc/hosts.
type HostsEntry struct {
	IP   string
	Host string
}

const hostsMarker = "# managed by phpv server"

// hostsPath is a variable so tests can redirect it.
var hostsPath = "/etc/hosts"

// HostsHasEntry reports whether /etc/hosts already maps host.
func HostsHasEntry(host string) bool {
	f, err := os.Open(hostsPath)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for _, f := range fields[1:] {
			if f == host {
				return true
			}
		}
	}
	return false
}

// HostsLine returns the exact line phpv would add for host.
func HostsLine(host string) string {
	return fmt.Sprintf("127.0.0.1 %s %s", host, hostsMarker)
}

// EnsureHostEntry adds `127.0.0.1 <host>` to /etc/hosts when missing.
//
// Writing /etc/hosts needs root, so this shells out to sudo. When sudo is
// unavailable or non-interactive the caller gets a *HostsPermissionError
// carrying the manual instruction; serving continues regardless.
func EnsureHostEntry(host string) error {
	if host == "" || host == "localhost" {
		return nil
	}
	if HostsHasEntry(host) {
		return nil
	}
	line := HostsLine(host)
	if err := appendHostsAsUser(line); err == nil {
		return nil
	}
	if err := appendHostsViaSudo(line); err != nil {
		return &HostsPermissionError{Host: host, Line: line, Err: err}
	}
	return nil
}

// HostsPermissionError signals that phpv could not edit /etc/hosts itself.
type HostsPermissionError struct {
	Host string
	Line string
	Err  error
}

func (e *HostsPermissionError) Error() string {
	return fmt.Sprintf("could not add %s to %s: %v", e.Host, hostsPath, e.Err)
}

// ManualInstruction returns the copy-pasteable fix for the user.
func (e *HostsPermissionError) ManualInstruction() string {
	return fmt.Sprintf("add this line to %s manually:\n    %s", hostsPath, e.Line)
}

func appendHostsAsUser(line string) error {
	f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n", line)
	return err
}

func appendHostsViaSudo(line string) error {
	if _, err := lookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not available")
	}
	// Use tee so sudo elevates only the write, never a shell.
	cmd := command("sudo", "-n", "tee", "-a", hostsPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdin, "\n%s\n", line); err != nil {
		stdin.Close()
		_ = cmd.Wait()
		return err
	}
	stdin.Close()
	return cmd.Wait()
}

// RemoveHostEntry drops every phpv-managed line for host from /etc/hosts.
// Missing lines and missing permissions are not treated as errors.
func RemoveHostEntry(host string) error {
	if host == "" || host == "localhost" {
		return nil
	}
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return nil
	}
	var kept []string
	removed := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, hostsMarker) {
			fields := strings.Fields(line)
			match := false
			for _, f := range fields[1:] {
				if f == host {
					match = true
					break
				}
			}
			if match {
				removed = true
				continue
			}
		}
		kept = append(kept, line)
	}
	if !removed {
		return nil
	}
	tmp := filepath.Join(os.TempDir(), "phpv-hosts")
	if err := os.WriteFile(tmp, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		return err
	}
	if err := copyIntoHosts(tmp); err != nil {
		return err
	}
	return os.Remove(tmp)
}

// copyFileOver copies src to dst with the given mode (replacing content).
func copyFileOver(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

// copyIntoHosts replaces /etc/hosts content with src (sudo when needed).
func copyIntoHosts(src string) error {
	if err := copyFileOver(src, hostsPath, 0o644); err == nil {
		return nil
	}
	if _, err := lookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not available")
	}
	return command("sudo", "-n", "cp", src, hostsPath).Run()
}
