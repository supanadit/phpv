package assembler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/system"
)

// mockSiloRepo is a minimal SiloRepository implementation for tests that only
// exercise path helpers.
type mockSiloRepo struct {
	root string
}

func (m *mockSiloRepo) Download(url, checksumType, checksumValue string) (bool, error) {
	return false, nil
}
func (m *mockSiloRepo) Extract(archivePath, destDir string) (bool, error) { return false, nil }
func (m *mockSiloRepo) GetSilo() domain.Silo                              { return domain.Silo{} }
func (m *mockSiloRepo) GetState(name, version string) (domain.InstallState, error) {
	return domain.StateNone, nil
}
func (m *mockSiloRepo) MarkInProgress(name, version string) error  { return nil }
func (m *mockSiloRepo) MarkComplete(name, version string) error    { return nil }
func (m *mockSiloRepo) MarkFailed(name, version string) error      { return nil }
func (m *mockSiloRepo) MarkInterrupted(name, version string) error { return nil }
func (m *mockSiloRepo) GetDefault() (string, error)                { return "", nil }
func (m *mockSiloRepo) SetDefault(version string) error            { return nil }
func (m *mockSiloRepo) PHPOutputPath(phpVersion string) string     { return "" }
func (m *mockSiloRepo) SourcePath(pkg, version string) string      { return "" }
func (m *mockSiloRepo) PackagePrefix(name, version string) string {
	return filepath.Join(m.root, "packages", name, version)
}
func (m *mockSiloRepo) PECLArchivePath(name, version string) string      { return "" }
func (m *mockSiloRepo) BuildLogPath(pkg, version, logName string) string { return "" }
func (m *mockSiloRepo) GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error) {
	return nil, nil
}
func (m *mockSiloRepo) SaveExtensionManifest(phpVersion string, manifest *domain.ExtensionManifest) error {
	return nil
}
func (m *mockSiloRepo) IsSystemMode() bool       { return false }
func (m *mockSiloRepo) SetSystemMode(bool) error { return nil }

func TestResolveVersionConstraint_Exact(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.4.2", "8.3.0", "7.4.0"}
	got, err := resolveVersionConstraint(versions, "8.4.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "8.4.1" {
		t.Fatalf("got %q, want 8.4.1", got)
	}
}

func TestResolveVersionConstraint_MajorMinor(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.4.2", "8.3.0", "7.4.0"}
	got, err := resolveVersionConstraint(versions, "8.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "8.4.2" {
		t.Fatalf("got %q, want 8.4.2 (latest patch)", got)
	}
}

func TestResolveVersionConstraint_MajorOnly(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.3.0", "8.2.0", "7.4.0"}
	got, err := resolveVersionConstraint(versions, "8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "8.4.1" {
		t.Fatalf("got %q, want 8.4.1 (latest 8.x)", got)
	}
}

func TestResolveVersionConstraint_NoMatch(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}
	_, err := resolveVersionConstraint(versions, "7")
	if err == nil {
		t.Fatal("expected error for non-matching constraint")
	}
}

func TestResolveVersionConstraint_EmptyVersions(t *testing.T) {
	_, err := resolveVersionConstraint(nil, "8")
	if err == nil {
		t.Fatal("expected error for empty versions")
	}
}

func TestLatestMatching(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.4.2", "8.3.0", "7.4.0"}
	got := latestMatching(versions, "8.4.")
	if got != "8.4.2" {
		t.Fatalf("got %q, want 8.4.2", got)
	}
}

func TestLatestMatching_NoMatch(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}
	got := latestMatching(versions, "7.")
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"8.4.0", "8.3.0", 1},
		{"8.3.0", "8.4.0", -1},
		{"8.4.0", "8.4.0", 0},
		{"8.4.2", "8.4.1", 1},
		{"8.4.0", "7.4.0", 1},
		{"7.4.0", "8.4.0", -1},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestFindDepPrefix_PicksHighestVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PHPV_ROOT", tmpDir)

	// Create two openssl versions: 1.0.1u and 1.1.1w
	oldDir := filepath.Join(tmpDir, "packages", "openssl", "1.0.1u", "include", "openssl")
	newDir := filepath.Join(tmpDir, "packages", "openssl", "1.1.1w", "include", "openssl")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatal(err)
	}

	svc := &Service{}
	got := svc.findDepPrefix(nil, "openssl", "include/openssl")
	want := filepath.Join(tmpDir, "packages", "openssl", "1.1.1w")
	if got != want {
		t.Errorf("findDepPrefix = %q, want %q (highest version)", got, want)
	}
}

// TestFindDepPrefix_FallbackRespectsConstraint verifies that the fallback to
// an installed version only picks versions that satisfy the build plan's
// version constraint. This prevents curl/8.10.1 (built against OpenSSL 3.x)
// from being used when PHP 7.4 needs curl/7.88.1 (>=7.80.0,<8.0.0).
func TestFindDepPrefix_FallbackRespectsConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PHPV_ROOT", tmpDir)

	// Installed curl versions.
	for _, ver := range []string{"7.88.1", "8.10.1"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, "packages", "curl", ver, "lib"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	mock := &mockSiloRepo{root: tmpDir}
	siloSvc := silo.NewService(mock, nil)
	svc := &Service{silo: siloSvc}
	deps := []domain.Dependency{
		{Name: "curl", Version: "7.88.1|>=7.80.0,<8.0.0"},
	}

	got := svc.findDepPrefix(deps, "curl", "lib")
	want := filepath.Join(tmpDir, "packages", "curl", "7.88.1")
	if got != want {
		t.Errorf("findDepPrefix = %q, want %q", got, want)
	}
}

// TestFindDepPrefix_FallbackWithoutExactMatch verifies that when the exact
// pinned version is not installed, the fallback picks the highest installed
// version that still satisfies the constraint.
func TestFindDepPrefix_FallbackWithoutExactMatch(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PHPV_ROOT", tmpDir)

	// Only curl 7.85.0 and 7.88.1 installed; plan wants 7.80.0 (not installed).
	for _, ver := range []string{"7.85.0", "7.88.1"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, "packages", "curl", ver, "lib"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	mock := &mockSiloRepo{root: tmpDir}
	siloSvc := silo.NewService(mock, nil)
	svc := &Service{silo: siloSvc}
	deps := []domain.Dependency{
		{Name: "curl", Version: "7.80.0|>=7.80.0,<8.0.0"},
	}

	got := svc.findDepPrefix(deps, "curl", "lib")
	want := filepath.Join(tmpDir, "packages", "curl", "7.88.1")
	if got != want {
		t.Errorf("findDepPrefix = %q, want %q", got, want)
	}
}

// TestResolveDependencyFlags_RespectsConstraintFallback verifies that
// --with-curl is resolved to a version that satisfies the plan's constraint,
// not merely the highest installed version.
func TestResolveDependencyFlags_RespectsConstraintFallback(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PHPV_ROOT", tmpDir)

	for _, ver := range []string{"7.88.1", "8.10.1"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, "packages", "curl", ver, "lib"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "packages", "curl", ver, "lib", "libcurl.so"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	mock := &mockSiloRepo{root: tmpDir}
	siloSvc := silo.NewService(mock, nil)
	svc := &Service{silo: siloSvc}
	deps := []domain.Dependency{
		{Name: "curl", Version: "7.88.1|>=7.80.0,<8.0.0"},
	}

	flags := svc.resolveDependencyFlags("php", "7.4.33", []string{"--with-curl"}, deps)
	want := "--with-curl=" + filepath.Join(tmpDir, "packages", "curl", "7.88.1")
	if len(flags) != 1 || flags[0] != want {
		t.Errorf("resolveDependencyFlags = %q, want %q", flags, want)
	}
}

// TestResolveDepPlaceholders_ResolvesWithInstalledPrefix verifies that
// {{dep:NAME}} placeholders are resolved for dependencies in the build plan
// when their install prefix directory exists (i.e., the dependency was built
// locally). The assembler builds dependencies in dependency order, so the
// directory will exist before the dependent is configured.
func TestResolveDepPlaceholders_ResolvesWithInstalledPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	mock := &mockSiloRepo{root: tmpDir}
	siloSvc := silo.NewService(mock, nil)
	svc := &Service{silo: siloSvc}

	deps := []domain.Dependency{
		{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
	}
	// Simulate the dependency being built locally by creating the directory.
	prefix := filepath.Join(tmpDir, "packages", "openssl", "1.1.1w")
	if err := os.MkdirAll(filepath.Join(prefix, "include", "openssl"), 0o755); err != nil {
		t.Fatal(err)
	}
	flags := []string{"--with-openssl={{dep:openssl}}"}
	got := svc.resolveDepPlaceholders(flags, deps)

	want := prefix
	if len(got) != 1 || got[0] != "--with-openssl="+want {
		t.Errorf("resolveDepPlaceholders = %q, want %q", got, "--with-openssl="+want)
	}
}

// TestResolveDepPlaceholders_SkipsWhenPrefixMissing verifies that
// {{dep:NAME}} placeholders are skipped when the dependency's install
// directory doesn't exist (i.e., the dependency was satisfied by a system
// package). In this case, the flag is omitted entirely so the package's
// configure script can find the system version via pkg-config.
func TestResolveDepPlaceholders_SkipsWhenPrefixMissing(t *testing.T) {
	tmpDir := t.TempDir()
	mock := &mockSiloRepo{root: tmpDir}
	siloSvc := silo.NewService(mock, nil)
	svc := &Service{silo: siloSvc}

	deps := []domain.Dependency{
		{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
	}
	flags := []string{"--with-openssl={{dep:openssl}}"}
	got := svc.resolveDepPlaceholders(flags, deps)

	if len(got) != 0 {
		t.Errorf("resolveDepPlaceholders = %q, want empty (flag should be skipped)", got)
	}
}

func TestShouldDownloadDep(t *testing.T) {
	systemPkgs := map[string]system.Package{
		"openssl": {Name: "openssl", SystemName: "openssl", Version: "3.2.0", Installed: true},
		"zlib":    {Name: "zlib", SystemName: "zlib", Version: "1.3.1", Installed: true},
	}

	cases := []struct {
		name       string
		dep        domain.Dependency
		mustBuild  map[string]bool
		systemPkgs map[string]system.Package
		static     bool
		want       bool
	}{
		{
			name: "build tool never downloaded",
			dep:  domain.Dependency{Name: "autoconf", Version: "2.71"},
			want: false,
		},
		{
			name:       "system package compatible -> skip download",
			dep:        domain.Dependency{Name: "openssl", Version: "1.1.1w|>=1.1.0,<4.0.0"},
			systemPkgs: systemPkgs,
			want:       false,
		},
		{
			name:       "system package incompatible -> download",
			dep:        domain.Dependency{Name: "openssl", Version: "1.1.1w|>=1.1.1,<3.0.0"},
			systemPkgs: systemPkgs,
			want:       true,
		},
		{
			name:       "must build -> download",
			dep:        domain.Dependency{Name: "openssl", Version: "1.1.1w|>=1.1.0,<4.0.0"},
			mustBuild:  map[string]bool{"openssl": true},
			systemPkgs: systemPkgs,
			want:       true,
		},
		{
			name:       "no system package -> download",
			dep:        domain.Dependency{Name: "curl", Version: "8.5.0"},
			systemPkgs: systemPkgs,
			want:       true,
		},
		{
			name:       "static build downloads everything",
			dep:        domain.Dependency{Name: "openssl", Version: "1.1.1w|>=1.1.0,<4.0.0"},
			systemPkgs: systemPkgs,
			static:     true,
			want:       true,
		},
		{
			name:       "no-system (nil map) downloads everything",
			dep:        domain.Dependency{Name: "openssl", Version: "1.1.1w|>=1.1.0,<4.0.0"},
			systemPkgs: nil,
			want:       true,
		},
		{
			name:       "no constraint -> download",
			dep:        domain.Dependency{Name: "readline", Version: "8.2"},
			systemPkgs: systemPkgs,
			want:       true,
		},
	}

	for _, c := range cases {
		got := shouldDownloadDep(c.dep, c.mustBuild, c.systemPkgs, c.static)
		if got != c.want {
			t.Errorf("%s: shouldDownloadDep(%v) = %v, want %v", c.name, c.dep.Name, got, c.want)
		}
	}
}

func TestComputeMustBuild(t *testing.T) {
	systemPkgs := map[string]system.Package{
		"openssl": {Name: "openssl", Version: "3.2.0", Installed: true},
		"zlib":    {Name: "zlib", Version: "1.3.1", Installed: true},
	}

	deps := []domain.Dependency{
		{Name: "openssl", Version: "1.1.1w|>=1.1.1,<3.0.0"}, // system 3.2.0 incompatible
		{Name: "zlib", Version: "1.3.1|>=1.3.0"},            // system 1.3.1 compatible
		{Name: "autoconf", Version: "2.71"},                 // build tool, excluded
		{Name: "libxml2", Version: "2.12.7|>=2.12.0"},       // no system pkg
	}

	svc := &Service{}
	mustBuild := svc.computeMustBuild(deps, systemPkgs, false)

	if !mustBuild["openssl"] {
		t.Errorf("openssl should be marked must-build (system 3.2.0 doesn't satisfy <3.0.0)")
	}
	if mustBuild["zlib"] {
		t.Errorf("zlib should NOT be marked must-build (system satisfies)")
	}
	if mustBuild["autoconf"] {
		t.Errorf("autoconf is a build tool and must not be in mustBuild")
	}
	if mustBuild["libxml2"] {
		t.Errorf("libxml2 has no system package and no constraint match, should not be forced")
	}

	// openssl propagation happens in Assemble; verify static/nil behave.
	if got := svc.computeMustBuild(deps, systemPkgs, true); len(got) != 0 {
		t.Errorf("static build should not mark any must-build, got %v", got)
	}
}
