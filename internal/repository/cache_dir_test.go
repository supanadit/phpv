package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCacheDir_PHPVRoot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	got := ResolveCacheDir()
	want := filepath.Join(dir, "caches")
	if got != want {
		t.Fatalf("ResolveCacheDir() = %q, want %q", got, want)
	}
}

func TestResolveCacheDir_DefaultHome(t *testing.T) {
	t.Setenv("PHPV_ROOT", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir error: %v", err)
	}

	got := ResolveCacheDir()
	want := filepath.Join(home, ".phpv", "caches")
	if got != want {
		t.Fatalf("ResolveCacheDir() = %q, want %q", got, want)
	}
}