package webserver

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// PickFreePort returns an available TCP port. When preferred is 0 the scan
// starts at the default server port 8080 and walks upward; otherwise only
// the preferred port is probed.
func PickFreePort(preferred int) (int, error) {
	start := preferred
	if start == 0 {
		start = 8080
	}
	for p := start; p < start+100; p++ {
		ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(p))
		if err == nil {
			ln.Close()
			return p, nil
		}
		if preferred != 0 {
			return 0, fmt.Errorf("port %d is not available", preferred)
		}
	}
	return 0, fmt.Errorf("no free port found in range %d-%d", start, start+99)
}

// FindBinary locates a command inside an installed package prefix, probing
// the usual layouts (bin/, sbin/, root). Falls back to exec.LookPath when
// the prefix is empty.
func FindBinary(prefix, name string) string {
	if prefix != "" {
		for _, rel := range []string{
			filepath.Join("bin", name),
			filepath.Join("sbin", name),
			name,
		} {
			p := filepath.Join(prefix, rel)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
				return p
			}
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

// WaitPort polls until the TCP port accepts connections or the timeout
// expires. Used after server startup to confirm the instance is serving.
func WaitPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("port %d did not open within %s", port, timeout)
}
