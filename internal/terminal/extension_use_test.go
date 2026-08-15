package terminal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newCmd(args ...string) *cobra.Command {
	cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().String("version", "", "")
	cmd.Flags().Bool("global", true, "")
	cmd.Flags().Bool("local", false, "")
	cmd.Flags().Bool("print", false, "")
	cmd.Flags().Bool("auto-deps", false, "")
	cmd.Flags().Bool("no-system", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Int("jobs", 0, "")
	_ = cmd.ParseFlags(args)
	return cmd
}

// captureStdout runs fn and returns everything written to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestExtensionAdd_DefaultsToActiveVersion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")
	if err := os.WriteFile(filepath.Join(dir, "default"), []byte("8.4.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h := newTestPHPHandler(dir)
	cmd := newCmd("--no-system", "--dry-run")
	out := captureStdout(t, func() {
		if err := h.extensionAdd(cmd, []string{"ctype"}); err != nil {
			t.Fatalf("extensionAdd with no --version should use active version, got error: %v", err)
		}
	})
	if !strings.Contains(out, "Dry run complete") {
		t.Fatalf("expected dry-run output, got: %q", out)
	}
}

func TestExtensionAdd_NoActiveVersionErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--no-system", "--dry-run")
	err := h.extensionAdd(cmd, []string{"ctype"})
	if err == nil {
		t.Fatal("extensionAdd with no active version and no --version should error")
	}
	if !strings.Contains(err.Error(), "no active PHP version") {
		t.Fatalf("expected active-version error, got: %v", err)
	}
}

func TestExtensionAdd_VersionFlagOverrides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--no-system", "--dry-run", "--version", "8.4")
	err := h.extensionAdd(cmd, []string{"ctype"})
	if err != nil {
		t.Fatalf("extensionAdd with --version 8.4 should resolve to 8.4.0, got error: %v", err)
	}
}

func TestExtensionAdd_VersionNotInstalledErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--no-system", "--dry-run", "--version", "7.4")
	err := h.extensionAdd(cmd, []string{"ctype"})
	if err == nil {
		t.Fatal("extensionAdd with --version 7.4 (not installed) should error")
	}
	if !strings.Contains(err.Error(), "PHP 7.4 is not installed") {
		t.Fatalf("expected not-installed error, got: %v", err)
	}
}

func TestUsePrint_EmitsExport(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--print")
	out := captureStdout(t, func() {
		if err := h.use(cmd, []string{"8.4"}); err != nil {
			t.Fatalf("use --print returned error: %v", err)
		}
	})
	want := `export PHPV_CURRENT="8.4.0"`
	if got := strings.TrimSpace(out); got != want {
		t.Fatalf("use --print 8.4 = %q, want %q", got, want)
	}
}

func TestUsePrint_SystemEmitsUnset(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	h := newTestPHPHandler(dir)
	cmd := newCmd("--print")
	out := captureStdout(t, func() {
		if err := h.use(cmd, []string{"system"}); err != nil {
			t.Fatalf("use --print system returned error: %v", err)
		}
	})
	if got := strings.TrimSpace(out); got != "unset PHPV_CURRENT" {
		t.Fatalf("use --print system = %q, want %q", got, "unset PHPV_CURRENT")
	}
}

func TestUsePrint_DoesNotWriteDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--print")
	err := h.use(cmd, []string{"8.4"})
	if err != nil {
		t.Fatalf("use --print returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "default")); !os.IsNotExist(err) {
		t.Fatalf("use --print must not write the global default file")
	}
}

func TestBareUse_DefaultsToGlobal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd()
	err := h.use(cmd, []string{"8.4"})
	if err != nil {
		t.Fatalf("bare use should default to global, got error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "default"))
	if err != nil {
		t.Fatalf("bare use should write the global default file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "8.4.0" {
		t.Fatalf("default file = %q, want 8.4.0", string(data))
	}
}

func TestUse_GlobalStillWorks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	cmd := newCmd("--global")
	err := h.use(cmd, []string{"8.4"})
	if err != nil {
		t.Fatalf("use --global returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "default"))
	if err != nil {
		t.Fatalf("use --global should write default file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "8.4.0" {
		t.Fatalf("default file = %q, want 8.4.0", string(data))
	}
}

func TestInitShell_EmitsShellFunction(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	h := newTestPHPHandler(dir)
	cmd := &cobra.Command{Use: "init"}
	out := captureStdout(t, func() {
		if err := h.initShell(cmd, []string{"bash"}); err != nil {
			t.Fatalf("initShell returned error: %v", err)
		}
	})
	if !strings.Contains(out, "export PATH=") {
		t.Fatalf("init output missing PATH export: %q", out)
	}
	if !strings.Contains(out, "phpv()") {
		t.Fatalf("init output missing phpv() function: %q", out)
	}
	if !strings.Contains(out, "use --print") {
		t.Fatalf("init output missing use --print: %q", out)
	}
}

func TestResolveOptionalVersion_Precedence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)
	createFakePHPInstall(t, dir, "8.4.0")
	createFakePHPInstall(t, dir, "7.4.0")
	if err := os.WriteFile(filepath.Join(dir, "default"), []byte("7.4.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h := newTestPHPHandler(dir)

	// --version flag wins.
	cmd := newCmd("--version", "8.4")
	got, err := h.resolveOptionalVersion(cmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "8.4.0" {
		t.Fatalf("resolveOptionalVersion with --version = %q, want 8.4.0", got)
	}

	// Legacy positional [version] wins over active.
	cmd = newCmd()
	got, err = h.resolveOptionalVersion(cmd, "8.4")
	if err != nil {
		t.Fatal(err)
	}
	if got != "8.4.0" {
		t.Fatalf("resolveOptionalVersion positional = %q, want 8.4.0", got)
	}

	// Neither set -> active version.
	cmd = newCmd()
	got, err = h.resolveOptionalVersion(cmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "7.4.0" {
		t.Fatalf("resolveOptionalVersion active = %q, want 7.4.0", got)
	}

	// Both --version and positional -> error.
	cmd = newCmd("--version", "8.4")
	_, err = h.resolveOptionalVersion(cmd, "7.4")
	if err == nil {
		t.Fatal("resolveOptionalVersion with both --version and positional should error")
	}
}
