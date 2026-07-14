package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewService(t *testing.T) {
	s := NewService()
	if s == nil {
		t.Fatal("NewService returned nil")
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()

	if err := s.Set("mirror", "https://cn2.php.net"); err != nil {
		t.Fatalf("Set mirror: %v", err)
	}

	got, err := s.Get("mirror")
	if err != nil {
		t.Fatalf("Get mirror: %v", err)
	}
	if got != "https://cn2.php.net" {
		t.Fatalf("Get mirror = %q, want %q", got, "https://cn2.php.net")
	}
}

func TestSetAndGetConcurrency(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()

	if err := s.Set("concurrency", "8"); err != nil {
		t.Fatalf("Set concurrency: %v", err)
	}

	got, err := s.Get("concurrency")
	if err != nil {
		t.Fatalf("Get concurrency: %v", err)
	}
	if got != "8" {
		t.Fatalf("Get concurrency = %q, want %q", got, "8")
	}
}

func TestSetAndGetBool(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()

	if err := s.Set("static_libgcc", "true"); err != nil {
		t.Fatalf("Set static_libgcc: %v", err)
	}

	got, err := s.Get("static_libgcc")
	if err != nil {
		t.Fatalf("Get static_libgcc: %v", err)
	}
	if got != "true" {
		t.Fatalf("Get static_libgcc = %q, want %q", got, "true")
	}
}

func TestGetUnknownKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("Get nonexistent expected error")
	}
}

func TestSetUnknownKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()
	err := s.Set("nonexistent", "value")
	if err == nil {
		t.Fatal("Set nonexistent expected error")
	}
}

func TestSetInvalidInt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()
	err := s.Set("concurrency", "notanumber")
	if err == nil {
		t.Fatal("Set concurrency with non-int expected error")
	}
}

func TestSetInvalidBool(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()
	err := s.Set("static_libgcc", "maybe")
	if err == nil {
		t.Fatal("Set static_libgcc with invalid bool expected error")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()
	if err := s.Set("mirror", "https://cn2.php.net"); err != nil {
		t.Fatal(err)
	}

	lines, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(lines) != len(knownKeys) {
		t.Fatalf("List returned %d lines, want %d", len(lines), len(knownKeys))
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s1 := NewService()
	if err := s1.Set("mirror", "https://cn2.php.net"); err != nil {
		t.Fatal(err)
	}

	s2 := NewService()
	got, err := s2.Get("mirror")
	if err != nil {
		t.Fatalf("Get mirror from new service: %v", err)
	}
	if got != "https://cn2.php.net" {
		t.Fatalf("Get mirror = %q, want %q", got, "https://cn2.php.net")
	}
}

func TestConfigFileLocation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()
	if err := s.Set("mirror", "https://cn2.php.net"); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(dir, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", configPath)
	}
}

func TestUnsetReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	s := NewService()
	got, err := s.Get("mirror")
	if err != nil {
		t.Fatalf("Get unset mirror: %v", err)
	}
	if got != "" {
		t.Fatalf("Get unset mirror = %q, want empty", got)
	}
}
