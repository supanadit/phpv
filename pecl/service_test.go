package pecl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockSiloRepo struct {
	manifest *domain.ExtensionManifest
}

func (m *mockSiloRepo) Download(url, checksumType, checksumValue string) (bool, error) {
	return false, nil
}
func (m *mockSiloRepo) Extract(archivePath, destDir string) (bool, error) {
	return false, nil
}
func (m *mockSiloRepo) GetSilo() domain.Silo {
	return domain.Silo{Root: "/test/root"}
}
func (m *mockSiloRepo) GetState(name, version string) (domain.InstallState, error) {
	return "", nil
}
func (m *mockSiloRepo) MarkInProgress(name, version string) error {
	return nil
}
func (m *mockSiloRepo) MarkComplete(name, version string) error {
	return nil
}
func (m *mockSiloRepo) MarkFailed(name, version string) error {
	return nil
}
func (m *mockSiloRepo) MarkInterrupted(name, version string) error {
	return nil
}
func (m *mockSiloRepo) GetDefault() (string, error) {
	return "", nil
}
func (m *mockSiloRepo) SetDefault(version string) error {
	return nil
}
func (m *mockSiloRepo) PHPOutputPath(phpVersion string) string {
	return "/test/root/packages/php/" + phpVersion
}
func (m *mockSiloRepo) SourcePath(pkg, version string) string {
	return "/test/root/sources/" + pkg + "/" + version
}
func (m *mockSiloRepo) PackagePrefix(name, version string) string {
	return "/test/root/packages/" + name + "/" + version
}
func (m *mockSiloRepo) PECLArchivePath(name, version string) string {
	return "/test/root/packages/pecl/" + name + "-" + version + ".tgz"
}
func (m *mockSiloRepo) BuildLogPath(pkg, version, logName string) string {
	return "/test/root/logs/" + pkg + "/" + version + "/" + logName + ".log"
}
func (m *mockSiloRepo) GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error) {
	if m.manifest != nil {
		return m.manifest, nil
	}
	return &domain.ExtensionManifest{PHPVersion: phpVersion}, nil
}
func (m *mockSiloRepo) SaveExtensionManifest(phpVersion string, manifest *domain.ExtensionManifest) error {
	m.manifest = manifest
	return nil
}
func (m *mockSiloRepo) IsSystemMode() bool {
	return false
}
func (m *mockSiloRepo) SetSystemMode(enabled bool) error {
	return nil
}

type mockRegistryRepo struct{}

func (m *mockRegistryRepo) List(name string, checksum bool, os string) ([]domain.Registry, error) {
	return nil, nil
}
func (m *mockRegistryRepo) Get(name, version string, checksum bool, os string) (domain.Registry, error) {
	return domain.Registry{}, nil
}

func newSiloService(mock *mockSiloRepo) PECLSilo {
	return mock
}

func TestParseNameVersion(t *testing.T) {
	tests := []struct {
		path        string
		wantName    string
		wantVersion string
	}{
		{"redis-6.0.2.tgz", "redis", "6.0.2"},
		{"imagick-3.7.0.tar.gz", "imagick", "3.7.0"},
		{"xdebug-3.3.0RC1.tgz", "xdebug", "3.3.0RC1"},
		{"memcached-3.2.0.tar.gz", "memcached", "3.2.0"},
		{"apcu-5.1.23.tgz", "apcu", "5.1.23"},
		{"yaml-2.2.3.tgz", "yaml", "2.2.3"},
		{"no-version.tgz", "no", "version"},
		{"foo-bar-1.2.3-alpha1.tgz", "foo-bar", "1.2.3-alpha1"},
	}
	for _, tt := range tests {
		name, ver, err := parseNameVersion(tt.path)
		if err != nil {
			t.Errorf("parseNameVersion(%q) unexpected error: %v", tt.path, err)
			continue
		}
		if name != tt.wantName {
			t.Errorf("parseNameVersion(%q) name = %q, want %q", tt.path, name, tt.wantName)
		}
		if ver != tt.wantVersion {
			t.Errorf("parseNameVersion(%q) version = %q, want %q", tt.path, ver, tt.wantVersion)
		}
	}
}

func TestIsLocalArchive(t *testing.T) {
	if !isLocalArchive("foo.tgz") {
		t.Error("isLocalArchive('foo.tgz') should be true")
	}
	if !isLocalArchive("foo.tar.gz") {
		t.Error("isLocalArchive('foo.tar.gz') should be true")
	}
	if !isLocalArchive("foo.tar.bz2") {
		t.Error("isLocalArchive('foo.tar.bz2') should be true")
	}
	if isLocalArchive("redis") {
		t.Error("isLocalArchive('redis') should be false")
	}
	if isLocalArchive("") {
		t.Error("isLocalArchive('') should be false")
	}
}

func TestList(t *testing.T) {
	mock := &mockSiloRepo{
		manifest: &domain.ExtensionManifest{
			PHPVersion: "8.4.0",
			Extensions: []domain.ExtensionState{
				{Name: "redis", Type: domain.ExtensionTypePECL, Version: "6.0.2"},
				{Name: "gd", Type: domain.ExtensionTypeBuiltin},
			},
		},
	}
	svc := NewService(newSiloService(mock), nil)

	exts, err := svc.List("8.4.0")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(exts) != 1 {
		t.Fatalf("List returned %d extensions, want 1", len(exts))
	}
	if exts[0].Name != "redis" {
		t.Fatalf("List[0].Name = %q, want redis", exts[0].Name)
	}
}

func TestListEmpty(t *testing.T) {
	mock := &mockSiloRepo{}
	svc := NewService(newSiloService(mock), nil)

	exts, err := svc.List("8.4.0")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(exts) != 0 {
		t.Fatalf("List returned %d extensions, want 0", len(exts))
	}
}

func TestUninstallNotFound(t *testing.T) {
	mock := &mockSiloRepo{
		manifest: &domain.ExtensionManifest{
			PHPVersion: "8.4.0",
			Extensions: []domain.ExtensionState{
				{Name: "gd", Type: domain.ExtensionTypeBuiltin},
			},
		},
	}
	svc := NewService(newSiloService(mock), nil)

	err := svc.Uninstall("nonexistent", "8.4.0")
	if err == nil {
		t.Fatal("Uninstall expected error for nonexistent extension")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("Uninstall error = %q, want 'not installed'", err)
	}
}

func TestUninstallBuiltin(t *testing.T) {
	mock := &mockSiloRepo{
		manifest: &domain.ExtensionManifest{
			PHPVersion: "8.4.0",
			Extensions: []domain.ExtensionState{
				{Name: "gd", Type: domain.ExtensionTypeBuiltin},
			},
		},
	}
	svc := NewService(newSiloService(mock), nil)

	err := svc.Uninstall("gd", "8.4.0")
	if err == nil {
		t.Fatal("Uninstall expected error for built-in extension")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("Uninstall error = %q, want 'not installed'", err)
	}
}

func TestExtractArchive(t *testing.T) {
	dir, err := os.MkdirTemp("", "pecl-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	tgzPath := filepath.Join(dir, "test.tgz")
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "config.m4"), []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := exec.Command("tar", "-czf", tgzPath, "-C", dir, "content")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tar create: %v\n%s", err, out)
	}

	extractDir := filepath.Join(dir, "extracted")
	if err := extractArchive(tgzPath, extractDir); err != nil {
		t.Fatalf("extractArchive error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(extractDir, "content", "config.m4")); os.IsNotExist(err) {
		t.Fatal("extracted file not found")
	}
}

func TestFindSourceDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "pecl-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	extDir := filepath.Join(dir, "redis-6.0.2")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "config.m4"), []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if got := findSourceDir(dir, "redis"); got != extDir {
		t.Fatalf("findSourceDir = %q, want %q", got, extDir)
	}
}

func TestFindSourceDirNested(t *testing.T) {
	dir, err := os.MkdirTemp("", "pecl-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	nested := filepath.Join(dir, "xdebug-3.3.0", "src")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "config.m4"), []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if got := findSourceDir(dir, "xdebug"); got != nested {
		t.Fatalf("findSourceDir = %q, want %q", got, nested)
	}
}

func TestFindSourceDirNotFound(t *testing.T) {
	dir, err := os.MkdirTemp("", "pecl-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	if got := findSourceDir(dir, "nonexistent"); got != "" {
		t.Fatalf("findSourceDir = %q, want empty", got)
	}
}

// fakePECLServer serves allreleases.xml and deps.<ver>.txt payloads from a
// fixture map so ResolveCompatible is exercised without touching
// pecl.php.net.
func fakePECLServer(t *testing.T, allreleases string, deps map[string]string) (*httptest.Server, *peclDownloader) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/r/xdebug/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/r/xdebug/allreleases.xml":
			fmt.Fprint(w, allreleases)
		case strings.HasPrefix(r.URL.Path, "/rest/r/xdebug/deps."):
			ver := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/rest/r/xdebug/deps."), ".txt")
			body, ok := deps[ver]
			if !ok {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, body)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	dl := &peclDownloader{client: srv.Client(), base: srv.URL}
	return srv, dl
}

const allreleasesXdebug = `<?xml version="1.0" encoding="UTF-8" ?>
<a xmlns="http://pear.php.net/dtd/rest.allreleases">
 <p>xdebug</p>
 <c>pecl.php.net</c>
 <r><v>3.3.2</v><s>stable</s></r>
 <r><v>3.2.2</v><s>stable</s></r>
 <r><v>3.2.0RC2</v><s>beta</s></r>
 <r><v>2.9.8</v><s>stable</s></r>
 <r><v>2.9.9RC1</v><s>beta</s></r>
</a>
`

const depsPHP8 = `a:1:{s:8:"required";a:2:{s:3:"php";a:2:{s:3:"min";s:5:"8.0.0";s:3:"max";s:6:"8.3.99";}}}`
const depsPHP7 = `a:1:{s:8:"required";a:2:{s:3:"php";a:2:{s:3:"min";s:5:"7.1.0";s:3:"max";s:6:"7.4.99";}}}`

func TestResolveCompatible_PicksNewestSupporting(t *testing.T) {
	_, dl := fakePECLServer(t, allreleasesXdebug, map[string]string{
		"3.3.2": depsPHP8,
		"3.2.2": depsPHP8,
		"2.9.8": depsPHP7,
	})

	pkg, err := dl.ResolveCompatible("xdebug", "8.2.0")
	if err != nil {
		t.Fatalf("ResolveCompatible: %v", err)
	}
	if pkg.Version != "3.3.2" {
		t.Errorf("ResolveCompatible(8.2.0) = %q, want 3.3.2", pkg.Version)
	}
}

func TestResolveCompatible_FallsBackToOlderRelease(t *testing.T) {
	_, dl := fakePECLServer(t, allreleasesXdebug, map[string]string{
		"3.3.2": depsPHP8,
		"3.2.2": depsPHP8,
		"2.9.8": depsPHP7,
	})

	pkg, err := dl.ResolveCompatible("xdebug", "7.4.33")
	if err != nil {
		t.Fatalf("ResolveCompatible: %v", err)
	}
	if pkg.Version != "2.9.8" {
		t.Errorf("ResolveCompatible(7.4.33) = %q, want 2.9.8", pkg.Version)
	}
}

func TestResolveCompatible_SkipsPrereleases(t *testing.T) {
	_, dl := fakePECLServer(t, allreleasesXdebug, map[string]string{
		"3.3.2": depsPHP8,
		"3.2.2": depsPHP8,
		"2.9.8": depsPHP7,
	})

	// PHP 8.4 is beyond every stable release's max (8.3.99): must error,
	// never pick the 3.2.0RC2 beta or any RC.
	_, err := dl.ResolveCompatible("xdebug", "8.4.0")
	if err == nil {
		t.Fatal("ResolveCompatible(8.4.0) should fail: no stable release supports it")
	}
	if !strings.Contains(err.Error(), "tried") {
		t.Errorf("error should list tried releases, got: %v", err)
	}
}

func TestResolveCompatible_NoneCompatibleListsTried(t *testing.T) {
	_, dl := fakePECLServer(t, allreleasesXdebug, map[string]string{
		"3.3.2": depsPHP8,
		"2.9.8": depsPHP7,
	})

	_, err := dl.ResolveCompatible("xdebug", "5.6.40")
	if err == nil {
		t.Fatal("expected error for ancient PHP")
	}
	if !strings.Contains(err.Error(), "3.3.2") || !strings.Contains(err.Error(), "2.9.8") {
		t.Errorf("error should list tried releases, got: %v", err)
	}
}

func TestParseDepsPHP(t *testing.T) {
	c, err := parseDepsPHP(`a:1:{s:8:"required";a:2:{s:3:"php";a:2:{s:3:"min";s:5:"8.0.0";s:3:"max";s:6:"8.3.99";}}}`)
	if err != nil {
		t.Fatalf("parseDepsPHP: %v", err)
	}
	if c.Min != "8.0.0" || c.Max != "8.3.99" {
		t.Errorf("parseDepsPHP = %+v, want min 8.0.0 max 8.3.99", c)
	}

	// max absent (unbounded upper) must not error
	c, err = parseDepsPHP(`a:1:{s:8:"required";a:1:{s:3:"php";a:1:{s:3:"min";s:5:"7.0.0";}}}`)
	if err != nil {
		t.Fatalf("parseDepsPHP without max: %v", err)
	}
	if c.Min != "7.0.0" || c.Max != "" {
		t.Errorf("parseDepsPHP = %+v, want min 7.0.0 max empty", c)
	}
}

func TestVersionInRange(t *testing.T) {
	cases := []struct {
		v, min, max string
		want        bool
	}{
		{"8.2.0", "8.0.0", "8.3.99", true},
		{"8.4.0", "8.0.0", "8.3.99", false},
		{"7.4.33", "7.1.0", "7.4.99", true},
		{"7.0.0", "7.1.0", "7.4.99", false},
		{"8.5.0", "8.0.0", "", true},  // no max
		{"7.9.9", "", "8.3.99", true}, // no min
	}
	for _, c := range cases {
		if got := versionInRange(c.v, c.min, c.max); got != c.want {
			t.Errorf("versionInRange(%q, %q, %q) = %v, want %v", c.v, c.min, c.max, got, c.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"8.2.0", "8.0.0", 1},
		{"7.4.33", "7.4.99", -1},
		{"8.3.99", "8.4.0", -1},
		{"3.3.2", "3.3.2", 0},
		{"8.2", "8.2.0", 0},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// InstallCompatible end-to-end against the fake server requires the full
// build toolchain; the routing layer (terminal) covers that in its own
// tests. Here we only assert the resolve→download plumbing via
// installPackage's error paths for a missing PHP install.
func TestInstallCompatible_MissingPHP(t *testing.T) {
	_, dl := fakePECLServer(t, allreleasesXdebug, map[string]string{
		"3.3.2": depsPHP8,
	})
	mock := &mockSiloRepo{}
	svc := NewService(newSiloService(mock), nil)
	// Point the service's downloader at the fake server.
	svc.downloader = &compatibleDownloader{dl}

	_, err := svc.InstallCompatible(context.Background(), domain.PECLInstallRequest{Name: "xdebug", PHPVersion: "8.2.0", Jobs: 2})
	if err == nil {
		t.Fatal("expected error: PHP not installed")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// compatibleDownloader adapts the fake downloader to the Downloader
// interface (Download hits the network; the missing-PHP guard fires first
// in this test, so it can be a stub).
type compatibleDownloader struct {
	*peclDownloader
}

func (c *compatibleDownloader) Download(pkg peclPackage, destDir string) (string, error) {
	return "", fmt.Errorf("network disabled in test")
}
