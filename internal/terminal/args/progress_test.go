package args

import (
	"strings"
	"testing"
	"time"
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

// Regression: CI logs (stdout is a pipe) used to receive one redraw line per
// 80ms spinner tick, so a single download produced dozens of near-identical
// lines full of \r and ANSI erase codes. Non-interactive stdout must get one
// plain line per progress message and nothing else.
func TestProgressReporter_NonInteractiveHasNoSpinnerRedraws(t *testing.T) {
	progress, stop := newProgressReporter()

	var stdout string
	stdout, _ = captureAll(t, func() {
		progress("download", "Downloading nginx 1.28.0...")
		// Long enough for several spinner ticks had the ticker been running.
		time.Sleep(250 * time.Millisecond)
		progress("done", "nginx 1.28.0 installed")
		stop()
	})
	lines := len(strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"))

	if strings.ContainsAny(stdout, "\r\033") {
		t.Fatalf("stdout contains carriage returns or escape codes: %q", stdout)
	}
	for _, frame := range spinnerFrames {
		if strings.Contains(stdout, frame) {
			t.Fatalf("stdout contains spinner frame %q: %q", frame, stdout)
		}
	}
	if lines != 2 {
		t.Fatalf("stdout has %d lines, want exactly 2 (one per message):\n%s", lines, stdout)
	}
	if !strings.HasPrefix(stdout, "↓ Downloading nginx 1.28.0...\n✓ nginx 1.28.0 installed\n") {
		t.Fatalf("stdout = %q, want one plain line per message with stage glyphs", stdout)
	}
}
