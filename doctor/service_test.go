package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/system"
)

type memRepository struct {
	files          map[string][]byte
	dirs           map[string][]string
	env            map[string]string
	path           string
	lookPathResult map[string]string
}

func newMemRepository() *memRepository {
	return &memRepository{
		files:          make(map[string][]byte),
		dirs:           make(map[string][]string),
		env:            make(map[string]string),
		lookPathResult: make(map[string]string),
	}
}

func (m *memRepository) Stat(path string) (os.FileInfo, error) {
	if _, ok := m.files[path]; ok {
		return os.Stat(path)
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
	return 1 << 30, 4096, nil
}

func (m *memRepository) LookPath(name string) (string, error) {
	if p, ok := m.lookPathResult[name]; ok {
		return p, nil
	}
	return "", os.ErrNotExist
}

func newTestService(repo Repository) *Service {
	return NewService(repo, system.NewService())
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

	svc := newTestService(newOSRepository())
	issues := svc.Check(root)
	// Filter out distro info (always present) and build tool issues (system-dependent)
	var nonInfoIssues []Issue
	for _, issue := range issues {
		if issue.Severity == SeverityInfo && strings.Contains(issue.Title, "Detected OS") {
			continue
		}
		if strings.Contains(issue.Title, "build tools") || strings.Contains(issue.Title, "system libraries") {
			continue
		}
		nonInfoIssues = append(nonInfoIssues, issue)
	}
	if len(nonInfoIssues) != 0 {
		t.Fatalf("expected 0 non-info issues, got %d: %+v", len(nonInfoIssues), nonInfoIssues)
	}
}

func TestCheck_DefaultNotInstalled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	t.Setenv("PATH", filepath.Join(root, "bin")+":"+os.Getenv("PATH"))

	os.WriteFile(filepath.Join(root, "default"), []byte("9.9.9\n"), 0644)
	os.MkdirAll(filepath.Join(root, "bin"), 0755)
	os.WriteFile(filepath.Join(root, "bin", "php"), []byte("#!/bin/bash\necho shim\n"), 0755)

	svc := newTestService(newOSRepository())
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

	svc := newTestService(newOSRepository())
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

	svc := newTestService(newOSRepository())
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

	svc := newTestService(newOSRepository())
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

func TestCheck_BuildToolsMissing(t *testing.T) {
	mem := newMemRepository()
	svc := NewService(mem, system.NewService())

	issues := svc.checkBuildTools()
	foundCritical := false
	foundOptional := false
	for _, issue := range issues {
		if issue.Severity == SeverityCritical && strings.Contains(issue.Title, "Missing build tools") {
			foundCritical = true
		}
		if issue.Severity == SeverityWarning && strings.Contains(issue.Title, "Optional build tools missing") {
			foundOptional = true
		}
	}
	if !foundCritical {
		t.Fatal("expected critical issue about missing build tools (gcc, g++, make)")
	}
	if !foundOptional {
		t.Fatal("expected warning about missing optional build tools")
	}
}

func TestCheck_BuildToolsPresent(t *testing.T) {
	mem := newMemRepository()
	mem.lookPathResult["gcc"] = "/usr/bin/gcc"
	mem.lookPathResult["g++"] = "/usr/bin/g++"
	mem.lookPathResult["make"] = "/usr/bin/make"
	mem.lookPathResult["cmake"] = "/usr/bin/cmake"
	mem.lookPathResult["autoconf"] = "/usr/bin/autoconf"
	mem.lookPathResult["automake"] = "/usr/bin/automake"
	mem.lookPathResult["m4"] = "/usr/bin/m4"
	mem.lookPathResult["perl"] = "/usr/bin/perl"
	mem.lookPathResult["bison"] = "/usr/bin/bison"
	mem.lookPathResult["re2c"] = "/usr/bin/re2c"
	mem.lookPathResult["flex"] = "/usr/bin/flex"
	mem.lookPathResult["pkg-config"] = "/usr/bin/pkg-config"
	mem.lookPathResult["xz"] = "/usr/bin/xz"

	svc := NewService(mem, system.NewService())
	issues := svc.checkBuildTools()
	if len(issues) != 0 {
		t.Fatalf("expected 0 build tool issues when all tools present, got %d: %+v", len(issues), issues)
	}
}

func TestCheck_DistroInfo(t *testing.T) {
	svc := newTestService(newOSRepository())
	issues := svc.checkDistroInfo()
	if len(issues) != 1 {
		t.Fatalf("expected 1 distro info issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Title, "Detected OS") {
		t.Fatalf("expected distro info, got: %s", issues[0].Title)
	}
}
