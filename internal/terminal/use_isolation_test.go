package terminal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureAll swaps both stdout and stderr while fn runs and returns what each
// received. Used to prove a command stays silent on a stream.
func captureAll(t *testing.T, fn func()) (string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()

	fn()
	outW.Close()
	errW.Close()

	var outBuf, errBuf bytes.Buffer
	if _, err := outBuf.ReadFrom(outR); err != nil {
		t.Fatal(err)
	}
	if _, err := errBuf.ReadFrom(errR); err != nil {
		t.Fatal(err)
	}
	return outBuf.String(), errBuf.String()
}

// Regression: `phpv use` switches the PHP CLI and must never depend on a
// webserver being installed or configured for the version.
func TestUsePrint_DoesNotRequireWebserver(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	t.Setenv("PHPV_CURRENT", "")
	createFakePHPInstall(t, root, "7.4.33")
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h := newTestPHPHandler(root)
	cmd := newCmd("--print")

	// No packages/<server> directory exists at all: use must still work.
	stdout, stderr := captureAll(t, func() {
		if err := h.use(cmd, []string{"7.4"}); err != nil {
			t.Fatalf("use --print: %v", err)
		}
	})

	if want := "export PHPV_CURRENT=\"7.4.33\"\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want no webserver hint", stderr)
	}
}

// Regression: the --local path is equally independent of webservers.
func TestUseLocal_DoesNotRequireWebserver(t *testing.T) {
	root := t.TempDir()
	work := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	createFakePHPInstall(t, root, "7.4.33")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	h := newTestPHPHandler(root)
	cmd := newCmd("--local")

	stdout, stderr := captureAll(t, func() {
		if err := h.use(cmd, []string{"7.4"}); err != nil {
			t.Fatalf("use --local: %v", err)
		}
	})

	if !strings.Contains(stdout, "Switched to PHP 7.4.33") {
		t.Fatalf("stdout = %q, want the switch confirmation", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want no webserver hint", stderr)
	}
}
