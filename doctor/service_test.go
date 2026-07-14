package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memRepository struct {
	files map[string][]byte
	dirs  map[string][]string
	env   map[string]string
	path  string
}

func newMemRepository() *memRepository {
	return &memRepository{
		files: make(map[string][]byte),
		dirs:  make(map[string][]string),
		env:   make(map[string]string),
	}
}

func (m *memRepository) Stat(path string) (os.FileInfo, error) {
	if _, ok := m.files[path]; ok {
		return os.Stat(path) // fallback to real stat for temp dirs
	}
	return os.Stat(path)
}

func (m *memRepository) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return os.ReadFile(path)
}

func (m *memRepository) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (m *memRepository) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (m *memRepository) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.files[path] = data
	return nil
}

func (m *memRepository) Remove(path string) error {
	delete(m.files, path)
	return nil
}

func (m *memRepository) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (m *memRepository) Getenv(key string) string {
	return m.env[key]
}

func (m *memRepository) PathList() []string {
	if m.path != "" {
		return []string{m.path}
	}
	return nil
}

func (m *memRepository) Statfs(path string) (bavail, bsize uint64, err error) {
	return 1 << 30, 4096, nil // 4TB free
}

func TestCheck_NoIssues(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	t.Setenv("PATH", filepath.Join(root, "bin")+":"+os.Getenv("PATH"))

	phpDir := filepath.Join(root, "packages", "php", "8.4.0")
	os.MkdirAll(filepath.Join(phpDir, "bin"), 0755)
	os.WriteFile(filepath.Join(phpDir, "bin", "php"), []byte("#!/bin/sh\necho php\n"), 0755)
	os.WriteFile(filepath.Join(phpDir, ".state"), []byte("installed"), 0644)
	os.WriteFile(filepath.Join(root, "default"), []byte("8.4.0\n"), 0644)
	os.MkdirAll(filepath.Join(root, "bin"), 0755)
	os.WriteFile(filepath.Join(root, "bin", "php"), []byte("#!/bin/bash\necho shim\n"), 0755)

	svc := NewService(newOSRepository())
	issues := svc.Check(root)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %+v", len(issues), issues)
	}
}

func TestCheck_DefaultNotInstalled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	t.Setenv("PATH", filepath.Join(root, "bin")+":"+os.Getenv("PATH"))

	os.WriteFile(filepath.Join(root, "default"), []byte("9.9.9\n"), 0644)
	os.MkdirAll(filepath.Join(root, "bin"), 0755)
	os.WriteFile(filepath.Join(root, "bin", "php"), []byte("#!/bin/bash\necho shim\n"), 0755)

	svc := NewService(newOSRepository())
	issues := svc.Check(root)
	found := false
	for _, issue := range issues {
		if issue.Severity == SeverityCritical && strings.Contains(issue.Title, "not installed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected critical issue about default version not installed")
	}
}

func TestCheck_ShimMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	svc := NewService(newOSRepository())
	issues := svc.Check(root)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue.Title, "Shim not found") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected warning about missing shim")
	}
}

func TestCheck_CacheWritable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	os.MkdirAll(filepath.Join(root, "bin"), 0755)
	os.WriteFile(filepath.Join(root, "bin", "php"), []byte("#!/bin/bash\necho shim\n"), 0755)

	svc := NewService(newOSRepository())
	issues := svc.Check(root)
	for _, issue := range issues {
		if strings.Contains(issue.Title, "Cache") {
			t.Fatalf("unexpected cache issue: %s", issue.Title)
		}
	}
}

func TestCheck_SystemMode(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	os.MkdirAll(filepath.Join(root, "bin"), 0755)
	os.WriteFile(filepath.Join(root, "bin", "php"), []byte("#!/bin/bash\necho shim\n"), 0755)
	os.WriteFile(filepath.Join(root, ".phpv_system"), []byte{}, 0644)

	svc := NewService(newOSRepository())
	issues := svc.Check(root)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue.Title, "System mode") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected info about system mode")
	}
}
