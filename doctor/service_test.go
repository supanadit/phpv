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
)

func newTestGraphService() *graph.Service {
	return graph.NewService(memory.NewGraphRepository())
}

func newTestService(repo Repository) *Service {
	return NewService(repo, newTestSystemService(), newTestGraphService())
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
	svc := NewService(fake, newTestSystemService(), newTestGraphService())

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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
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

	svc := NewService(fake, newTestSystemService(), newTestGraphService())
	got := svc.installedExtensions()
	if len(got) != 0 {
		t.Fatalf("expected no extensions, got %v", got)
	}
}

// fakeRepository is a doctor.Repository test double. Filesystem probes
// delegate to the real temp directory; only env, LookPath, and free
// disk-space are controllable via fields.
type fakeRepository struct {
	env            map[string]string
	lookPathResult map[string]string
	diskFree       uint64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		env:            make(map[string]string),
		lookPathResult: make(map[string]string),
		diskFree:       1 << 30,
	}
}

func (m *fakeRepository) LookPath(name string) (string, error) {
	if p, ok := m.lookPathResult[name]; ok {
		return p, nil
	}
	return "", os.ErrNotExist
}

func (m *fakeRepository) ShimExists(root string) bool {
	_, err := os.Stat(filepath.Join(root, "bin", "php"))
	return err == nil
}

func (m *fakeRepository) GetDefaultVersion(root string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(root, "default"))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

func (m *fakeRepository) IsVersionInstalled(root, version string) bool {
	_, err := os.Stat(filepath.Join(root, "packages", "php", version, "bin", "php"))
	return err == nil
}

func (m *fakeRepository) IsCacheWritable(root string) error {
	cacheDir := filepath.Join(root, "caches")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	testFile := filepath.Join(cacheDir, ".phpv_write_test")
	if err := os.WriteFile(testFile, []byte{}, 0o644); err != nil {
		return err
	}
	return os.Remove(testFile)
}

func (m *fakeRepository) ListPHPVersions(root string) ([]string, error) {
	phpDir := filepath.Join(root, "packages", "php")
	entries, err := os.ReadDir(phpDir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	return versions, nil
}

func (m *fakeRepository) GetInstallState(root, version string) (domain.InstallState, bool) {
	data, err := os.ReadFile(filepath.Join(root, "packages", "php", version, ".state"))
	if err != nil {
		return domain.StateNone, false
	}
	return domain.InstallState(strings.TrimSpace(string(data))), true
}

func (m *fakeRepository) ExtensionManifestReadable(root, version string) error {
	path := filepath.Join(root, "packages", "php", version, "extensions.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	_, err := os.ReadFile(path)
	return err
}

func (m *fakeRepository) ListInstalledExtensions(root, version string) ([]string, error) {
	path := filepath.Join(root, "packages", "php", version, "extensions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var manifest domain.ExtensionManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(manifest.Extensions))
	for _, e := range manifest.Extensions {
		names = append(names, e.Name)
	}
	return names, nil
}

func (m *fakeRepository) GetPHPVRoot() string {
	return m.env["PHPV_ROOT"]
}

func (m *fakeRepository) IsDirInPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}

func (m *fakeRepository) IsSystemMode(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".phpv_system"))
	return err == nil
}

func (m *fakeRepository) FreeDiskBytes(path string) (uint64, error) {
	return m.diskFree, nil
}

// fakeSystemService supplies package-manager state without touching the host.
type fakeSystemService struct {
	missing []domain.SystemPackage
	checked []domain.SystemPackage
}

func (f *fakeSystemService) DistroInfo() domain.Distro {
	return domain.Distro{Name: "fedora", PM: "dnf"}
}

func (f *fakeSystemService) Check(deps []domain.SystemPackage) (*domain.SystemCheckResult, error) {
	f.checked = deps
	return &domain.SystemCheckResult{Distro: f.DistroInfo(), Missing: f.missing}, nil
}

func (f *fakeSystemService) DependencySet(exts []domain.ExtensionDef) []domain.SystemPackage {
	seen := make(map[string]bool)
	var pkgs []domain.SystemPackage
	add := func(name string, required bool) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		pkgs = append(pkgs, domain.SystemPackage{Name: name, SystemName: name, Required: required})
	}
	for _, def := range exts {
		add(def.RequiresPackage, true)
	}
	for _, def := range exts {
		for _, name := range def.RequiresPackages {
			add(name, false)
		}
	}
	for _, name := range []string{"sqlite3", "readline", "pcre2", "sodium"} {
		add(name, false)
	}
	return pkgs
}

func newTestSystemService() SystemService { return &fakeSystemService{} }

func depNames(pkgs []domain.SystemPackage) []string {
	names := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		names = append(names, p.Name)
	}
	return names
}

func (f *fakeSystemService) InstallCommand(packages []domain.SystemPackage) string {
	names := make([]string, 0, len(packages))
	for _, p := range packages {
		names = append(names, p.SystemName)
	}
	return "dnf install -y " + strings.Join(names, " ")
}

// installExtensions writes an extension manifest so installedExtensions sees
// these extensions present for version 8.4.0.
func installExtensions(t *testing.T, root string, exts ...string) {
	t.Helper()
	dir := filepath.Join(root, "packages", "php", "8.4.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	parts := make([]string, 0, len(exts))
	for _, e := range exts {
		parts = append(parts, `{"name":"`+e+`","type":"builtin","version":"8.4.0"}`)
	}
	manifest := `{"extensions":[` + strings.Join(parts, ",") + `]}`
	if err := os.WriteFile(filepath.Join(dir, "extensions.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestCheckSystemPackages_DoesNotCheckArgon2(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	installExtensions(t, root, "zip")

	sysSvc := &fakeSystemService{}
	svc := NewService(newFakeRepository(), nil, newTestGraphService())
	svc.sysSvc = sysSvc

	svc.checkSystemPackages()

	for _, p := range sysSvc.checked {
		if p.Name == "argon2" {
			t.Fatalf("argon2 checked; the build cannot consume it. checked=%v", depNames(sysSvc.checked))
		}
	}
}

func TestCheckSystemPackages_SplitsRequiredAndAdvisory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	installExtensions(t, root, "zip")

	sysSvc := &fakeSystemService{missing: []domain.SystemPackage{
		{Name: "libzip", SystemName: "libzip-devel", Required: true},
		{Name: "pcre2", SystemName: "pcre2-devel", Required: false},
	}}
	svc := NewService(newFakeRepository(), nil, newTestGraphService())
	svc.sysSvc = sysSvc

	issues := svc.checkSystemPackages()

	var requiredIssue, advisoryIssue *Issue
	for i := range issues {
		switch {
		case strings.Contains(issues[i].Title, "Missing build dependencies"):
			requiredIssue = &issues[i]
		case strings.Contains(issues[i].Title, "Optional system libraries"):
			advisoryIssue = &issues[i]
		}
	}
	if requiredIssue == nil {
		t.Fatalf("no build-dependency issue in %+v", issues)
	}
	if advisoryIssue == nil {
		t.Fatalf("no advisory issue in %+v", issues)
	}
	if requiredIssue.Severity != SeverityWarning {
		t.Errorf("required severity = %q, want warning", requiredIssue.Severity)
	}
	if advisoryIssue.Severity != SeverityInfo {
		t.Errorf("advisory severity = %q, want info", advisoryIssue.Severity)
	}
	if !strings.Contains(requiredIssue.Detail, "libzip") {
		t.Errorf("required detail %q does not name libzip", requiredIssue.Detail)
	}
	if strings.Contains(requiredIssue.Detail, "pcre2") {
		t.Errorf("required detail %q should not name advisory pcre2", requiredIssue.Detail)
	}
	if !strings.Contains(advisoryIssue.Detail, "pcre2") {
		t.Errorf("advisory detail %q does not name pcre2", advisoryIssue.Detail)
	}
}

func TestCheckSystemPackages_AgreesWithInstallDerivation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	installExtensions(t, root, "zip")

	sysSvc := &fakeSystemService{}
	svc := NewService(newFakeRepository(), nil, newTestGraphService())
	svc.sysSvc = sysSvc
	svc.checkSystemPackages()

	// Rebuild the set the way `phpv install` does, from the same definitions.
	var defs []domain.ExtensionDef
	for _, ext := range svc.installedExtensions() {
		if def, ok := svc.graphSvc.GetExtensionDef(ext); ok {
			defs = append(defs, def)
		}
	}
	want := depNames(sysSvc.DependencySet(defs))
	got := depNames(sysSvc.checked)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("doctor checked %v, install would check %v", got, want)
	}
}
