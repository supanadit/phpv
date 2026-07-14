package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	issues := Check(root)
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

	issues := Check(root)
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

	issues := Check(root)
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

	issues := Check(root)
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

	issues := Check(root)
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
