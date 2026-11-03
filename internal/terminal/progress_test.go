package terminal

import (
	"strings"
	"testing"
)

// Regression: server installs used to run without any progress output. The
// shared reporter must render stage messages and terminate final stages with
// a newline so following output does not append to the spinner line.
func TestProgressReporter_RendersTerminalStages(t *testing.T) {
	progress, stop := newProgressReporter()

	stdout, stderr := captureAll(t, func() {
		progress("prepare", "php-fpm 7.4.33 not built yet — one-time setup")
		progress("download", "Downloading nginx 1.28.0...")
		progress("skip", "Already installed nginx@1.28.0")
		progress("done", "nginx 1.28.0 installed at /tmp/nginx")
		stop()
	})

	if stderr != "" {
		t.Fatalf("stderr = %q, want none", stderr)
	}
	if !strings.Contains(stdout, "⚙ php-fpm 7.4.33 not built yet — one-time setup") {
		t.Fatalf("stdout = %q, want the prepare stage", stdout)
	}
	if !strings.Contains(stdout, "Downloading nginx 1.28.0...") {
		t.Fatalf("stdout = %q, want the download stage", stdout)
	}
	if !strings.Contains(stdout, "↷ Already installed nginx@1.28.0\n") {
		t.Fatalf("stdout = %q, want a newline-terminated skip stage", stdout)
	}
	if !strings.Contains(stdout, "✓ nginx 1.28.0 installed at /tmp/nginx\n") {
		t.Fatalf("stdout = %q, want a newline-terminated done stage", stdout)
	}
}
