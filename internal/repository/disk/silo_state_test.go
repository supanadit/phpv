package disk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSiloRepository_GetState_None(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	state, err := repo.GetState("8.4.0")
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state != "" {
		t.Fatalf("GetState = %q, want empty", state)
	}
}

func TestSiloRepository_MarkInProgress(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	if err := repo.MarkInProgress("8.4.0"); err != nil {
		t.Fatalf("MarkInProgress returned error: %v", err)
	}

	state, err := repo.GetState("8.4.0")
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state != "in_progress" {
		t.Fatalf("GetState = %q, want in_progress", state)
	}
}

func TestSiloRepository_MarkComplete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	if err := repo.MarkInProgress("8.4.0"); err != nil {
		t.Fatalf("MarkInProgress returned error: %v", err)
	}
	if err := repo.MarkComplete("8.4.0"); err != nil {
		t.Fatalf("MarkComplete returned error: %v", err)
	}

	state, err := repo.GetState("8.4.0")
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state != "installed" {
		t.Fatalf("GetState = %q, want installed", state)
	}
}

func TestSiloRepository_MarkFailed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	if err := repo.MarkInProgress("8.4.0"); err != nil {
		t.Fatalf("MarkInProgress returned error: %v", err)
	}
	if err := repo.MarkFailed("8.4.0"); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}

	state, err := repo.GetState("8.4.0")
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state != "failed" {
		t.Fatalf("GetState = %q, want failed", state)
	}
}

func TestSiloRepository_GetDefault_None(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	ver, err := repo.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault returned error: %v", err)
	}
	if ver != "" {
		t.Fatalf("GetDefault = %q, want empty", ver)
	}
}

func TestSiloRepository_SetDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	if err := repo.SetDefault("8.4.0"); err != nil {
		t.Fatalf("SetDefault returned error: %v", err)
	}

	ver, err := repo.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault returned error: %v", err)
	}
	if ver != "8.4.0" {
		t.Fatalf("GetDefault = %q, want 8.4.0", ver)
	}
}

func TestSiloRepository_GetSilo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	silo := repo.GetSilo()
	if silo.Root != dir {
		t.Fatalf("GetSilo().Root = %q, want %q", silo.Root, dir)
	}
}

func TestSiloRepository_PathHelpers(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()

	got := repo.PHPOutputPath("8.4.0")
	want := filepath.Join(dir, "versions", "8.4.0", "output")
	if got != want {
		t.Fatalf("PHPOutputPath = %q, want %q", got, want)
	}

	got = repo.SourcePath("php", "8.4.0")
	want = filepath.Join(dir, "sources", "php", "8.4.0")
	if got != want {
		t.Fatalf("SourcePath = %q, want %q", got, want)
	}

	got = repo.PackagePrefix("openssl", "1.1.1w")
	want = filepath.Join(dir, "packages", "openssl", "1.1.1w")
	if got != want {
		t.Fatalf("PackagePrefix = %q, want %q", got, want)
	}
}

func TestSiloRepository_StateFilePersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()

	// Write state via one instance.
	if err := repo.MarkInProgress("8.4.0"); err != nil {
		t.Fatalf("MarkInProgress: %v", err)
	}
	if err := repo.MarkComplete("8.4.0"); err != nil {
		t.Fatalf("MarkComplete: %v", err)
	}

	// Read state via a new instance (proves file persistence).
	repo2 := NewSiloRepository()
	state, err := repo2.GetState("8.4.0")
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state != "installed" {
		t.Fatalf("GetState = %q, want installed", state)
	}
}

func TestSiloRepository_DefaultFilePersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	if err := repo.SetDefault("8.4.0"); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}

	repo2 := NewSiloRepository()
	ver, err := repo2.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault: %v", err)
	}
	if ver != "8.4.0" {
		t.Fatalf("GetDefault = %q, want 8.4.0", ver)
	}
}

func TestSiloPaths_WithPHPVRoot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	if got := RootPath(); got != dir {
		t.Fatalf("RootPath = %q, want %q", got, dir)
	}
	if got := CachePath(); got != filepath.Join(dir, "caches") {
		t.Fatalf("CachePath = %q, want %q", got, filepath.Join(dir, "caches"))
	}
	if got := SourcesPath(); got != filepath.Join(dir, "sources") {
		t.Fatalf("SourcesPath = %q, want %q", got, filepath.Join(dir, "sources"))
	}
	if got := SourcePath("php", "8.4.0"); got != filepath.Join(dir, "sources", "php", "8.4.0") {
		t.Fatalf("SourcePath = %q, want %q", got, filepath.Join(dir, "sources", "php", "8.4.0"))
	}
	if got := VersionPath("8.4.0"); got != filepath.Join(dir, "versions", "8.4.0") {
		t.Fatalf("VersionPath = %q, want %q", got, filepath.Join(dir, "versions", "8.4.0"))
	}
	if got := PHPOutputPath("8.4.0"); got != filepath.Join(dir, "versions", "8.4.0", "output") {
		t.Fatalf("PHPOutputPath = %q, want %q", got, filepath.Join(dir, "versions", "8.4.0", "output"))
	}
	if got := PackagePrefix("openssl", "1.1.1w"); got != filepath.Join(dir, "packages", "openssl", "1.1.1w") {
		t.Fatalf("PackagePrefix = %q, want %q", got, filepath.Join(dir, "packages", "openssl", "1.1.1w"))
	}
	if got := BinPath(); got != filepath.Join(dir, "bin") {
		t.Fatalf("BinPath = %q, want %q", got, filepath.Join(dir, "bin"))
	}
	if got := StatePath("8.4.0"); got != filepath.Join(dir, "versions", "8.4.0", ".state") {
		t.Fatalf("StatePath = %q, want %q", got, filepath.Join(dir, "versions", "8.4.0", ".state"))
	}
	if got := DefaultPath(); got != filepath.Join(dir, "default") {
		t.Fatalf("DefaultPath = %q, want %q", got, filepath.Join(dir, "default"))
	}
}

func TestSiloPaths_WithoutPHPVRoot(t *testing.T) {
	t.Setenv("PHPV_ROOT", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	expected := filepath.Join(home, ".phpv")
	if got := RootPath(); got != expected {
		t.Fatalf("RootPath = %q, want %q", got, expected)
	}
}

func TestSiloPaths_StatePathCreatesDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	if err := repo.MarkInProgress("8.4.0"); err != nil {
		t.Fatalf("MarkInProgress: %v", err)
	}

	stateFile := StatePath("8.4.0")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatalf("state file %s should exist", stateFile)
	}
}
