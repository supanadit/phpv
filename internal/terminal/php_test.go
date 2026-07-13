package terminal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/supanadit/phpv/bundle"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

func TestFindPhpvrc_NoFile(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	if got := findPhpvrc(); got != "" {
		t.Fatalf("findPhpvrc() = %q, want empty", got)
	}
}

func TestFindPhpvrc_Found(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".phpvrc"), []byte("8.4.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	if got := findPhpvrc(); got != "8.4.0" {
		t.Fatalf("findPhpvrc() = %q, want 8.4.0", got)
	}
}

func TestFindPhpvrc_WalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".phpvrc"), []byte("7.4.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	os.Chdir(subdir)
	defer os.Chdir(origWd)

	if got := findPhpvrc(); got != "7.4.0" {
		t.Fatalf("findPhpvrc() = %q, want 7.4.0", got)
	}
}

func TestResolveInstalledVersion_Exact(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	createFakePHPInstall(t, dir, "8.4.0")

	h := newTestPHPHandler(dir)
	got, err := h.resolveInstalledVersion("8.4.0")
	if err != nil {
		t.Fatalf("resolveInstalledVersion returned error: %v", err)
	}
	if got != "8.4.0" {
		t.Fatalf("resolveInstalledVersion = %q, want 8.4.0", got)
	}
}

func TestResolveInstalledVersion_MajorMinor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	for _, ver := range []string{"8.4.0", "8.4.1", "8.4.2", "8.3.0"} {
		createFakePHPInstall(t, dir, ver)
	}

	h := newTestPHPHandler(dir)
	got, err := h.resolveInstalledVersion("8.4")
	if err != nil {
		t.Fatalf("resolveInstalledVersion returned error: %v", err)
	}
	if got != "8.4.2" {
		t.Fatalf("resolveInstalledVersion = %q, want 8.4.2 (latest patch)", got)
	}
}

func TestResolveInstalledVersion_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	h := newTestPHPHandler(dir)
	_, err := h.resolveInstalledVersion("8.4.0")
	if err == nil {
		t.Fatal("resolveInstalledVersion expected error for uninstalled version")
	}
}

func TestResolveActivePHP_Default(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	createFakePHPInstall(t, dir, "8.4.0")
	createFakePHPInstall(t, dir, "8.3.0")

	// Set default to 8.3.0.
	diskRepo := disk.NewSiloRepository()
	diskRepo.SetDefault("8.3.0")

	h := newTestPHPHandler(dir)
	path, err := h.resolveActivePHP()
	if err != nil {
		t.Fatalf("resolveActivePHP returned error: %v", err)
	}
	want := filepath.Join(dir, "versions", "8.3.0", "output", "bin", "php")
	if path != want {
		t.Fatalf("resolveActivePHP = %q, want %q", path, want)
	}
}

func TestResolveActivePHP_Phpvrc(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	createFakePHPInstall(t, dir, "7.4.0")
	createFakePHPInstall(t, dir, "8.4.0")

	if err := os.WriteFile(filepath.Join(dir, ".phpvrc"), []byte("7.4.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	h := newTestPHPHandler(dir)
	path, err := h.resolveActivePHP()
	if err != nil {
		t.Fatalf("resolveActivePHP returned error: %v", err)
	}
	want := filepath.Join(dir, "versions", "7.4.0", "output", "bin", "php")
	if path != want {
		t.Fatalf("resolveActivePHP = %q, want %q", path, want)
	}
}

func TestResolveActivePHP_NoPHP(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	h := newTestPHPHandler(dir)
	_, err := h.resolveActivePHP()
	if err == nil {
		t.Fatal("resolveActivePHP expected error when no PHP installed")
	}
}

func createFakePHPInstall(t *testing.T, root, version string) {
	t.Helper()
	phpBin := filepath.Join(root, "versions", version, "output", "bin", "php")
	if err := os.MkdirAll(filepath.Dir(phpBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(phpBin, []byte("#!/bin/sh\necho php\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

func newTestPHPHandler(root string) *PHPHandler {
	diskRepo := disk.NewSiloRepository()
	regSvc := registry.NewService(&mockRegistryRepo{})
	siloSvc := silo.NewService(diskRepo, regSvc)
	bundleSvc := bundle.NewService(siloSvc)
	return &PHPHandler{
		siloSvc:     siloSvc,
		registrySvc: regSvc,
		bundleSvc:   bundleSvc,
	}
}

type mockRegistryRepo struct{}

func (m *mockRegistryRepo) List(name string, checksum bool, os string) ([]domain.Registry, error) {
	return nil, nil
}

func (m *mockRegistryRepo) Get(name, version string, checksum bool, os string) (domain.Registry, error) {
	return domain.Registry{}, nil
}
