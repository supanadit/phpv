package assembler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/compiler"
	"github.com/supanadit/phpv/pkg/source"
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
	got := vsn.LatestMatching(versions, "8.4.")
	if got != "8.4.2" {
		t.Fatalf("got %q, want 8.4.2", got)
	}
}

func TestLatestMatching_NoMatch(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}
	got := vsn.LatestMatching(versions, "7.")
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
		got := vsn.CompareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
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

	siloSvc := &testSiloService{root: tmpDir}
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

	siloSvc := &testSiloService{root: tmpDir}
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

	siloSvc := &testSiloService{root: tmpDir}
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

// Regression: curl gets a bare --with-ssl from graph defaults and
// --with-openssl=<prefix> from its patch flags. Both used to reach configure;
// curl 7.12 ignored the unknown --with-openssl and used the bare flag, so it
// linked against system OpenSSL 3 and failed checking for CRYPTO_lock.
func TestResolveDependencyFlags_CurlCollapsesSSLFlags(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PHPV_ROOT", tmpDir)
	prefix := filepath.Join(tmpDir, "packages", "openssl", "1.0.1u")
	if err := os.MkdirAll(filepath.Join(prefix, "include", "openssl"), 0o755); err != nil {
		t.Fatal(err)
	}
	svc := &Service{silo: &testSiloService{root: tmpDir}}
	deps := []domain.Dependency{{Name: "openssl", Version: "1.0.1u|>=1.1.0,<2.0.0"}}

	flags := []string{
		"--with-ssl", "--without-brotli",
		"--with-openssl=" + prefix, "--without-brotli",
	}
	got := svc.resolveDependencyFlags("curl", "7.12.0", flags, deps)
	want := []string{"--without-brotli", "--without-brotli", "--with-ssl=" + prefix}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveDependencyFlags = %q, want %q", got, want)
	}
}

// TestResolveDepPlaceholders_ResolvesWithInstalledPrefix verifies that
// {{dep:NAME}} placeholders are resolved for dependencies in the build plan
// when their install prefix directory exists (i.e., the dependency was built
// locally). The assembler builds dependencies in dependency order, so the
// directory will exist before the dependent is configured.
func TestResolveDepPlaceholders_ResolvesWithInstalledPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	siloSvc := &testSiloService{root: tmpDir}
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
	siloSvc := &testSiloService{root: tmpDir}
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
	systemPkgs := map[string]domain.SystemPackage{
		"openssl": {Name: "openssl", SystemName: "openssl", Version: "3.2.0", Installed: true},
		"zlib":    {Name: "zlib", SystemName: "zlib", Version: "1.3.1", Installed: true},
	}

	cases := []struct {
		name       string
		dep        domain.Dependency
		mustBuild  map[string]bool
		systemPkgs map[string]domain.SystemPackage
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
	systemPkgs := map[string]domain.SystemPackage{
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

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// TestEnsureCLIBinary_ReplacesNonCLI covers PHP 4, where `make install` leaves
// the CGI SAPI at bin/php while a working CLI sits in the source tree.
func TestEnsureCLIBinary_ReplacesNonCLI(t *testing.T) {
	sourceDir, prefix := t.TempDir(), t.TempDir()
	writeExecutable(t, filepath.Join(sourceDir, "sapi", "cli", "php"), "cli-binary")
	writeExecutable(t, filepath.Join(prefix, "bin", "php"), "cgi-binary")

	if err := ensureCLIBinary(sourceDir, prefix); err != nil {
		t.Fatalf("ensureCLIBinary() returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(prefix, "bin", "php"))
	if err != nil {
		t.Fatalf("ReadFile(bin/php): %v", err)
	}
	if string(got) != "cli-binary" {
		t.Errorf("bin/php = %q, want the CLI binary (cli-binary)", got)
	}
	if info, err := os.Stat(filepath.Join(prefix, "bin", "php")); err != nil || info.Mode().Perm() != 0o755 {
		t.Errorf("bin/php mode = %v (err %v), want 0755", info.Mode().Perm(), err)
	}
}

// TestEnsureCLIBinary_LeavesMatchingInstallAlone guards the PHP 5+ path, where
// `make install` already put the CLI in place: the file must not be rewritten.
func TestEnsureCLIBinary_LeavesMatchingInstallAlone(t *testing.T) {
	sourceDir, prefix := t.TempDir(), t.TempDir()
	writeExecutable(t, filepath.Join(sourceDir, "sapi", "cli", "php"), "cli-binary")
	installed := filepath.Join(prefix, "bin", "php")
	writeExecutable(t, installed, "cli-binary")

	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(installed, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if err := ensureCLIBinary(sourceDir, prefix); err != nil {
		t.Fatalf("ensureCLIBinary() returned error: %v", err)
	}

	info, err := os.Stat(installed)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.ModTime().Equal(past) {
		t.Errorf("bin/php was rewritten (mtime %s), want untouched", info.ModTime())
	}
}

// TestEnsureCLIBinary_NoCLIBuilt makes a --disable-cli build a no-op instead of
// an error.
func TestEnsureCLIBinary_NoCLIBuilt(t *testing.T) {
	sourceDir, prefix := t.TempDir(), t.TempDir()
	installed := filepath.Join(prefix, "bin", "php")
	writeExecutable(t, installed, "cgi-binary")

	if err := ensureCLIBinary(sourceDir, prefix); err != nil {
		t.Fatalf("ensureCLIBinary() returned error: %v", err)
	}
	got, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("ReadFile(bin/php): %v", err)
	}
	if string(got) != "cgi-binary" {
		t.Errorf("bin/php = %q, want untouched (cgi-binary)", got)
	}
}

func newApacheTestService(t *testing.T, root string) *ApacheService {
	t.Helper()
	regSvc := &testRegistryService{}
	return NewApacheService(&Service{reg: regSvc, silo: &testSiloService{root: root}}, root)
}

func TestEnsureApacheModule_ExistingSkipsProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")

	prefix := filepath.Join(root, "packages", "php-apache", "7.4.33")
	module := filepath.Join(prefix, "modules", "libphp.so")
	if err := os.MkdirAll(filepath.Dir(module), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(module, []byte("ELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "modphp",
		PHPVersion:  "7.4.33",
		ProfileHash: profile.Hash("modphp") + ":" + filepath.Base("/no-httpd") + ":",
	}); err != nil {
		t.Fatal(err)
	}

	var stages []string
	got, err := newApacheTestService(t, root).EnsureApacheModule(context.Background(), ApacheModuleSpec{PHPVersion: "7.4.33", HTTPDPrefix: "/no-httpd", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, Progress: func(stage, _ string) { stages = append(stages, stage) }})
	if err != nil {
		t.Fatalf("EnsureApacheModule: %v", err)
	}
	if got != module {
		t.Fatalf("module = %q, want %q", got, module)
	}
	if len(stages) != 0 {
		t.Fatalf("progress stages = %v, want none", stages)
	}
}

func TestEnsureApacheModule_RequiresApxs(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	writeFakeCLIProfile(t, root, "7.4.33")

	_, err := newApacheTestService(t, root).EnsureApacheModule(context.Background(), ApacheModuleSpec{PHPVersion: "7.4.33", HTTPDPrefix: "/no-httpd", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, Progress: nil})
	if err == nil || !strings.Contains(err.Error(), "apxs") {
		t.Fatalf("error = %v, want an apxs hint", err)
	}
}

func TestFindApacheModule(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "libs"), 0o755); err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(src, "libs", "libphp.so")
	if err := os.WriteFile(module, []byte("ELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findApacheModule(src)
	if err != nil {
		t.Fatalf("findApacheModule: %v", err)
	}
	if got != module {
		t.Fatalf("module = %q, want %q", got, module)
	}
}

func TestApacheNeedsRebuild(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")
	svc := newApacheTestService(t, root)
	httpdPrefix := "/no-httpd"

	if !svc.NeedsRebuild("7.4.33", httpdPrefix, compiler.Toolchain{}) {
		t.Fatal("missing module must need a rebuild")
	}

	prefix := filepath.Join(root, "packages", "php-apache", "7.4.33")
	module := filepath.Join(prefix, "modules", "libphp.so")
	if err := os.MkdirAll(filepath.Dir(module), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(module, []byte("ELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	hash := profile.Hash("modphp") + ":" + filepath.Base(httpdPrefix) + ":"
	if err := writeVariantMarker(prefix, VariantMarker{SAPI: "modphp", PHPVersion: "7.4.33", ProfileHash: hash}); err != nil {
		t.Fatal(err)
	}
	if svc.NeedsRebuild("7.4.33", httpdPrefix, compiler.Toolchain{}) {
		t.Fatal("matching marker must not need a rebuild")
	}
}

func newFPMTestService(t *testing.T, root string) *FPMService {
	t.Helper()
	regSvc := &testRegistryService{}
	return NewFPMService(&Service{reg: regSvc, silo: &testSiloService{root: root}}, root)
}

func writeFakeCLIProfile(t *testing.T, root, version string) BuildProfile {
	t.Helper()
	prefix := filepath.Join(root, "packages", "php", version)
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	profile := BuildProfile{ConfigureArgs: []string{
		"--prefix=" + prefix,
		"--disable-all",
		"--enable-cli",
		"--with-zlib",
	}}
	if err := saveBuildProfile(prefix, profile); err != nil {
		t.Fatal(err)
	}
	return profile
}

// Regression: the php-fpm build used to print plain lines with no spinner.
// Progress must be reported through the callback so `server run` can animate.
func TestEnsureFPM_ReportsPrepareProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	writeFakeCLIProfile(t, root, "0.0.0")

	var stages []string
	_, err := newFPMTestService(t, root).EnsureFPM(context.Background(), FPMSpec{PHPVersion: "0.0.0", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, StagedFrom: "", Progress: func(stage, _ string) { stages = append(stages, stage) }})
	if err == nil {
		t.Fatal("expected an error for an unknown PHP version")
	}
	if len(stages) == 0 || stages[0] != "prepare" {
		t.Fatalf("progress stages = %v, want prepare first", stages)
	}
}

// An existing FPM binary with a matching profile marker needs no work and must
// not animate or emit stages.
func TestEnsureFPM_ExistingBinarySkipsProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")

	prefix := filepath.Join(root, "packages", "php-fpm", "7.4.33")
	bin := filepath.Join(prefix, "sbin", "php-fpm")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "fpm",
		PHPVersion:  "7.4.33",
		ProfileHash: profile.Hash("fpm"),
	}); err != nil {
		t.Fatal(err)
	}

	var stages []string
	got, err := newFPMTestService(t, root).EnsureFPM(context.Background(), FPMSpec{PHPVersion: "7.4.33", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, StagedFrom: "", Progress: func(stage, _ string) { stages = append(stages, stage) }})
	if err != nil {
		t.Fatalf("EnsureFPM: %v", err)
	}
	if got != bin {
		t.Fatalf("binary = %q, want %q", got, bin)
	}
	if len(stages) != 0 {
		t.Fatalf("progress stages = %v, want none", stages)
	}
}

// A stale marker (CLI rebuilt with different extensions) must force a rebuild.
func TestEnsureFPM_StaleMarkerRebuilds(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	writeFakeCLIProfile(t, root, "0.0.0")

	prefix := filepath.Join(root, "packages", "php-fpm", "0.0.0")
	bin := filepath.Join(prefix, "sbin", "php-fpm")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "fpm",
		PHPVersion:  "0.0.0",
		ProfileHash: "stale",
	}); err != nil {
		t.Fatal(err)
	}

	var stages []string
	if _, err := newFPMTestService(t, root).EnsureFPM(context.Background(), FPMSpec{PHPVersion: "0.0.0", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, StagedFrom: "", Progress: func(stage, _ string) { stages = append(stages, stage) }}); err == nil {
		t.Fatal("expected the stale build to attempt a rebuild and fail without php sources")
	}
	if len(stages) == 0 || stages[0] != "prepare" {
		t.Fatalf("progress stages = %v, want a rebuild starting at prepare", stages)
	}
}

func TestFPMNeedsRebuild(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	profile := writeFakeCLIProfile(t, root, "7.4.33")
	svc := newFPMTestService(t, root)

	if !svc.NeedsRebuild("7.4.33", compiler.Toolchain{}) {
		t.Fatal("missing FPM binary must need a rebuild")
	}

	prefix := filepath.Join(root, "packages", "php-fpm", "7.4.33")
	bin := filepath.Join(prefix, "sbin", "php-fpm")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{SAPI: "fpm", PHPVersion: "7.4.33", ProfileHash: profile.Hash("fpm")}); err != nil {
		t.Fatal(err)
	}
	if svc.NeedsRebuild("7.4.33", compiler.Toolchain{}) {
		t.Fatal("matching marker must not need a rebuild")
	}
	if err := writeVariantMarker(prefix, VariantMarker{SAPI: "fpm", PHPVersion: "7.4.33", ProfileHash: "stale"}); err != nil {
		t.Fatal(err)
	}
	if !svc.NeedsRebuild("7.4.33", compiler.Toolchain{}) {
		t.Fatal("stale marker must need a rebuild")
	}
}

type fakePatchRemote struct {
	tree map[string]string
	rev  string
	fail error
}

func (f *fakePatchRemote) ListTree(ctx context.Context) ([]domain.RemoteFile, string, error) {
	if f.fail != nil {
		return nil, "", f.fail
	}
	out := make([]domain.RemoteFile, 0, len(f.tree))
	for p, c := range f.tree {
		out = append(out, domain.RemoteFile{Path: p, BlobSHA: "sha-" + c})
	}
	return out, f.rev, nil
}

func (f *fakePatchRemote) FetchBlob(ctx context.Context, path, dest string) error {
	if f.fail != nil {
		return f.fail
	}
	return nil
}

func TestSyncPatchesWarnAndContinueOnFailure(t *testing.T) {
	svc := &Service{patches: &testMirrorService{fail: errors.New("dial: connection refused")}}

	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 1 {
		t.Fatalf("events = %v, want exactly the warning", events)
	}
	if events[0][:4] != "warn" {
		t.Errorf("event = %q, want a warn event", events[0])
	}
}

func TestSyncPatchesSyncsDeltaBeforeBuild(t *testing.T) {
	svc := &Service{patches: &testMirrorService{files: 1}}

	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 1 || events[0][:10] != "patch-sync" {
		t.Fatalf("events = %v, want a patch-sync event", events)
	}
}

func TestSyncPatchesSkipsWithinLease(t *testing.T) {
	svc := &Service{patches: &testMirrorService{}}
	// Prime the lease with a successful full sync.
	if _, err := svc.patches.Update(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 1 || events[0][:10] != "patch-sync" {
		t.Fatalf("events = %v, want the lease-skip report", events)
	}
}

func TestSyncPatchesNilServiceIsNoop(t *testing.T) {
	svc := &Service{}
	var events []string
	svc.syncPatches(context.Background(), func(stage, msg string) {
		events = append(events, stage+": "+msg)
	})
	if len(events) != 0 {
		t.Fatalf("events = %v, want none", events)
	}
}

// extractSiloRepo simulates a successful extraction.
type extractSiloRepo struct{ mockSiloRepo }

func (r *extractSiloRepo) Extract(archivePath, destDir string) (bool, error) {
	src := filepath.Join(destDir, "php-8.5.12")
	if err := os.MkdirAll(src, 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(src, "configure"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		return false, err
	}
	return true, nil
}

// Regression: an interrupted extraction leaves a non-empty src dir; the
// shared Extract guard would skip re-extracting it forever.
func TestPreparePHPSourceCleansPartialExtraction(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "build", "php-apache", "8.5.12")

	partial := filepath.Join(work, "src", "php-8.5.12")
	if err := os.MkdirAll(partial, 0o755); err != nil {
		t.Fatal(err)
	}
	// No configure -> source.Find cannot use this tree.

	regSvc := &testRegistryService{}
	siloSvc := &testSiloService{root: root, extract: true}
	patcherSvc := &testPatcherService{}
	deps := &Service{reg: regSvc, silo: siloSvc, patcher: patcherSvc}

	if err := preparePHPSource(deps, "8.5.12", work); err != nil {
		t.Fatalf("preparePHPSource: %v", err)
	}
	if source.Find(filepath.Join(work, "src"), "php") == "" {
		t.Fatal("source tree was not re-extracted")
	}
}

func TestParseConfigureOptions(t *testing.T) {
	args, env := parseConfigureOptions("--prefix=/x --disable-all --with-curl PKG_CONFIG_PATH=/a:/b")
	if len(args) != 3 {
		t.Fatalf("args = %v, want 3", args)
	}
	if env["PKG_CONFIG_PATH"] != "/a:/b" {
		t.Fatalf("env = %v, want PKG_CONFIG_PATH", env)
	}
}

func TestVariantArgsFPMKeepsExtensions(t *testing.T) {
	profile := BuildProfile{ConfigureArgs: []string{
		"--prefix=/cli", "--disable-all", "--enable-cli", "--with-curl",
		"--with-pdo-mysql=mysqlnd", "--enable-mbstring",
	}}
	args, err := profile.VariantArgs("fpm", "/fpm", "")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--prefix=/fpm", "--enable-fpm", "--with-curl",
		"--with-pdo-mysql=mysqlnd", "--enable-mbstring",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("fpm args missing %q: %v", want, args)
		}
	}
	for _, unwanted := range []string{"--prefix=/cli", "--enable-cli"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("fpm args still contain %q: %v", unwanted, args)
		}
	}
}

func TestVariantArgsModPHPRequiresApxs(t *testing.T) {
	profile := BuildProfile{ConfigureArgs: []string{"--disable-all"}}
	if _, err := profile.VariantArgs("modphp", "/apache-php", "/no-httpd"); err == nil {
		t.Fatal("expected an error when apxs is missing")
	}

	httpd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(httpd, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	apxs := filepath.Join(httpd, "bin", "apxs")
	if err := os.WriteFile(apxs, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	args, err := profile.VariantArgs("modphp", "/apache-php", httpd)
	if err != nil {
		t.Fatalf("VariantArgs: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--with-apxs2="+apxs) || !strings.Contains(joined, "--disable-fpm") {
		t.Fatalf("modphp args = %v", args)
	}
}

func TestProfileHashChangesWithSAPI(t *testing.T) {
	profile := BuildProfile{ConfigureArgs: []string{"--with-curl"}}
	if profile.Hash("fpm") == profile.Hash("modphp") {
		t.Fatal("profile hash must differ by SAPI")
	}
	if profile.Hash("fpm") == (BuildProfile{ConfigureArgs: []string{"--with-curl", "--enable-gd"}}).Hash("fpm") {
		t.Fatal("profile hash must change when extensions change")
	}
}

func TestCLIExtensionDir(t *testing.T) {
	prefix := t.TempDir()
	if err := os.MkdirAll(filepath.Join(prefix, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	phpConfig := filepath.Join(prefix, "bin", "php-config")
	if err := os.WriteFile(phpConfig, []byte("#!/bin/sh\necho /opt/php/ext\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := cliExtensionDir(prefix); got != "/opt/php/ext" {
		t.Fatalf("cliExtensionDir = %q, want /opt/php/ext", got)
	}
}

func TestBuildProfileRoundTrip(t *testing.T) {
	prefix := t.TempDir()
	want := BuildProfile{
		ConfigureArgs: []string{"--disable-all", "--enable-pdo"},
		Env:           map[string]string{"PKG_CONFIG_PATH": "/a:/b", "CFLAGS": "-O2"},
	}
	if err := saveBuildProfile(prefix, want); err != nil {
		t.Fatalf("saveBuildProfile: %v", err)
	}
	got, err := readBuildProfile(prefix)
	if err != nil {
		t.Fatalf("readBuildProfile: %v", err)
	}
	if strings.Join(got.ConfigureArgs, " ") != strings.Join(want.ConfigureArgs, " ") {
		t.Fatalf("args = %v, want %v", got.ConfigureArgs, want.ConfigureArgs)
	}
	if got.Env["PKG_CONFIG_PATH"] != "/a:/b" || got.Env["CFLAGS"] != "-O2" {
		t.Fatalf("env = %v", got.Env)
	}
}

func newServerTestService(t *testing.T) *Service {
	t.Helper()
	regSvc := &testRegistryService{}
	root := os.Getenv("PHPV_ROOT")
	siloSvc := &testSiloService{root: root}
	return &Service{silo: siloSvc, reg: regSvc}
}

func writeFakeServerBinary(t *testing.T, prefix, sub, name string) {
	t.Helper()
	dir := filepath.Join(prefix, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// Regression: servers have no include/ tree, so the old isDepInstalled check
// always reported them missing and re-downloaded/rebuilt on every install.
func TestAssembleServer_SkipsInstalledStateWithoutIncludeDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	prefix := filepath.Join(root, "packages", "nginx", "1.28.0")
	writeFakeServerBinary(t, prefix, "sbin", "nginx")
	if err := os.WriteFile(filepath.Join(prefix, ".state"), []byte("installed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(prefix, "include")); err == nil {
		t.Fatal("test setup must not create include/, servers do not ship one")
	}

	var stages []string
	got, err := newServerTestService(t).AssembleServer(context.Background(), ServerAssembleSpec{Name: "nginx", Version: "1.28.0", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, Progress: func(stage, _ string) { stages = append(stages, stage) }})
	if err != nil {
		t.Fatalf("AssembleServer: %v", err)
	}
	if got != prefix {
		t.Fatalf("prefix = %q, want %q", got, prefix)
	}
	want := []string{"resolve", "skip"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("progress stages = %v, want %v", stages, want)
	}
}

// Legacy installs predate state markers; the installed binary alone must be
// enough to skip a rebuild.
func TestAssembleServer_SkipsLegacyBinaryWithoutState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	prefix := filepath.Join(root, "packages", "nginx", "1.28.0")
	writeFakeServerBinary(t, prefix, "sbin", "nginx")

	var stages []string
	if _, err := newServerTestService(t).AssembleServer(context.Background(), ServerAssembleSpec{Name: "nginx", Version: "1.28.0", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, Progress: func(stage, _ string) { stages = append(stages, stage) }}); err != nil {
		t.Fatalf("AssembleServer: %v", err)
	}
	want := []string{"resolve", "skip"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("progress stages = %v, want %v", stages, want)
	}
}

func TestServerInstalled_NotInstalled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	svc := newServerTestService(t)
	prefix := filepath.Join(root, "packages", "nginx", "1.28.0")
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	if svc.serverInstalled("nginx", "nginx", "1.28.0", prefix) {
		t.Fatal("serverInstalled = true for an empty prefix")
	}
}

// Progress must be reported even when assembly fails early, so callers can
// show what the command is doing.
func TestAssembleServer_EmitsResolveProgressOnError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)

	var stages []string
	_, err := newServerTestService(t).AssembleServer(context.Background(), ServerAssembleSpec{Name: "nginx", Version: "0.0.0-nope", Verbose: false, Jobs: 1, TC: domain.Toolchain{}, Progress: func(stage, _ string) { stages = append(stages, stage) }})
	if err == nil {
		t.Fatal("expected an error for an unknown version")
	}
	if len(stages) == 0 || stages[0] != "resolve" {
		t.Fatalf("progress stages = %v, want resolve first", stages)
	}
}

// testSiloService implements AssemblerSilo without importing the silo module.
type testSiloService struct {
	root    string
	extract bool
}

func (s *testSiloService) Download(name, version string) (bool, error) { return false, nil }
func (s *testSiloService) DownloadURL(url, checksumType, checksumValue string) (bool, error) {
	return false, nil
}
func (s *testSiloService) Extract(archivePath, destDir string) (bool, error) {
	if !s.extract {
		return false, nil
	}
	src := filepath.Join(destDir, "php")
	if err := os.MkdirAll(src, 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(src, "configure"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		return false, err
	}
	return true, nil
}
func (s *testSiloService) GetSilo() domain.Silo { return domain.Silo{Root: s.root} }
func (s *testSiloService) GetState(name, version string) (domain.InstallState, error) {
	return domain.StateNone, nil
}
func (s *testSiloService) MarkInProgress(name, version string) error  { return nil }
func (s *testSiloService) MarkComplete(name, version string) error    { return nil }
func (s *testSiloService) MarkFailed(name, version string) error      { return nil }
func (s *testSiloService) MarkInterrupted(name, version string) error { return nil }
func (s *testSiloService) GetDefault() (string, error)                { return "", nil }
func (s *testSiloService) SetDefault(version string) error            { return nil }
func (s *testSiloService) SourcePath(pkg, version string) string {
	return filepath.Join(s.root, "src", pkg+"-"+version)
}
func (s *testSiloService) PackagePrefix(name, version string) string {
	return filepath.Join(s.root, "packages", name, version)
}
func (s *testSiloService) PHPOutputPath(phpVersion string) string {
	return filepath.Join(s.root, "packages", "php", phpVersion)
}
func (s *testSiloService) GetExtensionManifest(string) (*domain.ExtensionManifest, error) {
	return nil, nil
}
func (s *testSiloService) SaveExtensionManifest(string, *domain.ExtensionManifest) error {
	return nil
}

// testRegistryService implements AssemblerRegistry without importing the registry module.
type testRegistryService struct{}

func (r *testRegistryService) Get(name, version string) (domain.Registry, error) {
	return domain.Registry{}, nil
}
func (r *testRegistryService) List(name string) ([]domain.Registry, error) {
	if name == "nginx" {
		return []domain.Registry{{Name: "nginx", Version: "1.28.0", URL: "https://example.test/nginx.tar.gz"}}, nil
	}
	return nil, nil
}

// testPatcherService implements AssemblerPatcher without importing the patcher module.
type testPatcherService struct{}

func (p *testPatcherService) Prepare(name, version, sourceDir string) (*domain.PreparedPatches, error) {
	return &domain.PreparedPatches{}, nil
}

// testMirrorService implements AssemblerMirror without importing the mirror module.
type testMirrorService struct {
	fail   error
	files  int
	primed bool
}

func (m *testMirrorService) EnsureFresh(context.Context) (domain.PatchSyncResult, error) {
	if m.fail != nil {
		return domain.PatchSyncResult{}, m.fail
	}
	if m.primed {
		return domain.PatchSyncResult{SkippedCheck: true}, nil
	}
	res := domain.PatchSyncResult{}
	for i := 0; i < m.files; i++ {
		res.Downloaded = append(res.Downloaded, "file")
	}
	return res, nil
}

func (m *testMirrorService) Update(context.Context, *domain.PatchSyncScope) (domain.PatchSyncResult, error) {
	m.primed = true
	return domain.PatchSyncResult{}, nil
}
