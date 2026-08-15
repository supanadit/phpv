package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPharList_PositionalVersionResolves(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd()
	out := captureStdout(t, func() {
		if err := h.pharList(cmd, []string{"8.4"}); err != nil {
			t.Fatalf("pharList returned error: %v", err)
		}
	})
	if !strings.Contains(out, "PHP 8.4.0") {
		t.Fatalf("phar list 8.4 should resolve to 8.4.0, got: %q", out)
	}
}

func TestPharList_VersionFlagResolves(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--version", "8.4")
	out := captureStdout(t, func() {
		if err := h.pharList(cmd, []string{}); err != nil {
			t.Fatalf("pharList returned error: %v", err)
		}
	})
	if !strings.Contains(out, "PHP 8.4.0") {
		t.Fatalf("phar list --version 8.4 should resolve to 8.4.0, got: %q", out)
	}
}

func TestPharList_BothVersionSourcesErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")
	createFakePHPInstall(t, dir, "7.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--version", "8.4")
	err := h.pharList(cmd, []string{"7.4"})
	if err == nil {
		t.Fatal("phar list with both --version and positional should error")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Fatalf("expected both-sources error, got: %v", err)
	}
}

func TestPharList_UnknownVersionErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--version", "9.9")
	err := h.pharList(cmd, []string{})
	if err == nil {
		t.Fatal("phar list --version 9.9 (not installed) should error")
	}
	if !strings.Contains(err.Error(), "PHP 9.9 is not installed") {
		t.Fatalf("expected not-installed error, got: %v", err)
	}
}

func TestPharList_DefaultsToActiveVersion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")
	os.WriteFile(filepath.Join(dir, "default"), []byte("8.4.0\n"), 0644)

	h := newTestPHPHandler(dir)
	cmd := newCmd()
	out := captureStdout(t, func() {
		if err := h.pharList(cmd, []string{}); err != nil {
			t.Fatalf("pharList returned error: %v", err)
		}
	})
	if !strings.Contains(out, "PHP 8.4.0") {
		t.Fatalf("phar list with no version should use active 8.4.0, got: %q", out)
	}
}

func TestPharInstall_VersionResolution(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--version", "8.4")
	got, err := h.resolveOptionalVersion(cmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "8.4.0" {
		t.Fatalf("resolveOptionalVersion(--version 8.4) = %q, want 8.4.0", got)
	}

	// Positional [version] also works and takes precedence over active.
	cmd = newCmd()
	got, err = h.resolveOptionalVersion(cmd, "8.4")
	if err != nil {
		t.Fatal(err)
	}
	if got != "8.4.0" {
		t.Fatalf("resolveOptionalVersion(positional 8.4) = %q, want 8.4.0", got)
	}

	// Both sources -> error.
	cmd = newCmd("--version", "8.4")
	_, err = h.resolveOptionalVersion(cmd, "7.4")
	if err == nil {
		t.Fatal("resolveOptionalVersion with both sources should error")
	}
}
