package pecl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
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
func (m *mockSiloRepo) ToolchainPath(arch string) string {
	return "/test/root/toolchains/" + arch
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

func newSiloService(mock *mockSiloRepo) *silo.Service {
	return silo.NewService(mock, registry.NewService(&mockRegistryRepo{}))
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
	svc := NewService(newSiloService(mock))

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
	svc := NewService(newSiloService(mock))

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
	svc := NewService(newSiloService(mock))

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
	svc := NewService(newSiloService(mock))

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
