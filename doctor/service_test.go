package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/graph"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/system"
)

func newTestGraphService() *graph.Service {
	return graph.NewService(memory.NewGraphRepository())
}

func newTestService(repo Repository) *Service {
	return NewService(repo, system.NewService(), newTestGraphService())
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

	svc := newTestService(newFakeRepository())
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

	svc := newTestService(newFakeRepository())
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

	svc := newTestService(newFakeRepository())
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

	svc := newTestService(newFakeRepository())
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

	svc := newTestService(newFakeRepository())
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
	fake := newFakeRepository()
	svc := NewService(fake, system.NewService(), newTestGraphService())

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
	fake := newFakeRepository()
	fake.lookPathResult["gcc"] = "/usr/bin/gcc"
	fake.lookPathResult["g++"] = "/usr/bin/g++"
	fake.lookPathResult["make"] = "/usr/bin/make"
	fake.lookPathResult["cmake"] = "/usr/bin/cmake"
	fake.lookPathResult["autoconf"] = "/usr/bin/autoconf"
	fake.lookPathResult["automake"] = "/usr/bin/automake"
	fake.lookPathResult["m4"] = "/usr/bin/m4"
	fake.lookPathResult["perl"] = "/usr/bin/perl"
	fake.lookPathResult["bison"] = "/usr/bin/bison"
	fake.lookPathResult["re2c"] = "/usr/bin/re2c"
	fake.lookPathResult["flex"] = "/usr/bin/flex"
	fake.lookPathResult["pkg-config"] = "/usr/bin/pkg-config"
	fake.lookPathResult["xz"] = "/usr/bin/xz"

	svc := NewService(fake, system.NewService(), newTestGraphService())
	issues := svc.checkBuildTools()
	if len(issues) != 0 {
		t.Fatalf("expected 0 build tool issues when all tools present, got %d: %+v", len(issues), issues)
	}
}

func TestCheck_DistroInfo(t *testing.T) {
	svc := newTestService(newFakeRepository())
	issues := svc.checkDistroInfo()
	if len(issues) != 1 {
		t.Fatalf("expected 1 distro info issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Title, "Detected OS") {
		t.Fatalf("expected distro info, got: %s", issues[0].Title)
	}
}

func TestCheck_StateFiles(t *testing.T) {
	fake := newFakeRepository()
	root := t.TempDir()

	phpDir := filepath.Join(root, "packages", "php", "8.4.0")
	os.MkdirAll(phpDir, 0755)
	os.WriteFile(filepath.Join(phpDir, ".state"), []byte("failed"), 0644)

	svc := NewService(fake, system.NewService(), newTestGraphService())
	issues := svc.checkStateFiles(root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 state file issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Title, "failed") {
		t.Fatalf("expected failed state issue, got: %s", issues[0].Title)
	}
}

func TestCheck_DiskSpace(t *testing.T) {
	fake := newFakeRepository()
	root := t.TempDir()
	fake.diskFree = 100 * 1024 * 1024

	svc := NewService(fake, system.NewService(), newTestGraphService())
	issues := svc.checkDiskSpace(root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 disk space issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Title, "Low disk space") {
		t.Fatalf("expected low disk space issue, got: %s", issues[0].Title)
	}
}

func TestCheck_ShimInPath(t *testing.T) {
	fake := newFakeRepository()
	root := t.TempDir()

	svc := NewService(fake, system.NewService(), newTestGraphService())
	issues := svc.checkShimInPath(root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 shim-in-path issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Title, "not in PATH") {
		t.Fatalf("expected shim not in PATH issue, got: %s", issues[0].Title)
	}
}

func TestCheck_PHPVEnvMismatch(t *testing.T) {
	fake := newFakeRepository()
	root := t.TempDir()
	fake.env["PHPV_ROOT"] = "/other/root"

	svc := NewService(fake, system.NewService(), newTestGraphService())
	issues := svc.checkPHPVEnv(root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 env mismatch issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Title, "PHPV_ROOT mismatch") {
		t.Fatalf("expected PHPV_ROOT mismatch issue, got: %s", issues[0].Title)
	}
}

func TestCheck_MissingStateFile(t *testing.T) {
	fake := newFakeRepository()
	root := t.TempDir()

	phpDir := filepath.Join(root, "packages", "php", "8.4.0")
	os.MkdirAll(phpDir, 0755)

	svc := NewService(fake, system.NewService(), newTestGraphService())
	issues := svc.checkStateFiles(root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 missing-state issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Title, "Missing state file") {
		t.Fatalf("expected missing state file issue, got: %s", issues[0].Title)
	}
}

func TestInstalledExtensions_AggregatesManifests(t *testing.T) {
	root := t.TempDir()
	fake := newFakeRepository()
	fake.env["PHPV_ROOT"] = root

	// Two PHP versions, each with an extension manifest.
	writeManifest := func(version string, names ...string) {
		dir := filepath.Join(root, "packages", "php", version)
		os.MkdirAll(dir, 0755)
		var exts []domain.ExtensionState
		for _, n := range names {
			exts = append(exts, domain.ExtensionState{Name: n, Type: domain.ExtensionTypeBuiltin})
		}
		data, err := json.Marshal(domain.ExtensionManifest{PHPVersion: version, Extensions: exts})
		if err != nil {
			t.Fatalf("marshal manifest: %v", err)
		}
		os.WriteFile(filepath.Join(dir, "extensions.json"), data, 0644)
	}
	writeManifest("8.4.0", "pdo_pgsql", "pdo")
	writeManifest("8.3.0", "pdo_pgsql", "zip")

	svc := NewService(fake, system.NewService(), newTestGraphService())
	got := svc.installedExtensions()
	want := []string{"pdo", "pdo_pgsql", "zip"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("installedExtensions() = %v, want %v", got, want)
	}
}

func TestInstalledExtensions_NoManifest(t *testing.T) {
	root := t.TempDir()
	fake := newFakeRepository()
	fake.env["PHPV_ROOT"] = root

	os.MkdirAll(filepath.Join(root, "packages", "php", "8.4.0"), 0755)

	svc := NewService(fake, system.NewService(), newTestGraphService())
	got := svc.installedExtensions()
	if len(got) != 0 {
		t.Fatalf("expected no extensions, got %v", got)
	}
}
