package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
)

func TestIsValidVersion(t *testing.T) {
	cases := []struct {
		v    string
		want bool
	}{
		{"8.4.4", true},
		{"8.4", true},
		{"8", true},
		{"0.1.0", true},
		{"", false},
		{"; rm -rf /", false},
		{"$(whoami)", false},
		{"8.4.4; echo pwned", false},
		{"a.b.c", false},
		{"8.4.4.4", true},
	}
	for _, c := range cases {
		got := IsValidVersion(c.v)
		if got != c.want {
			t.Errorf("IsValidVersion(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestRenderShim_Binary(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, "#!/bin/bash") {
		t.Error("shim should start with #!/bin/bash")
	}
	if !strings.Contains(content, "PHPV_ROOT") {
		t.Error("shim should reference PHPV_ROOT")
	}
	if !strings.Contains(content, "PHPV_CURRENT") {
		t.Error("shim should reference PHPV_CURRENT")
	}
	if !strings.Contains(content, "LD_LIBRARY_PATH") {
		t.Error("shim should reference LD_LIBRARY_PATH")
	}
	if !strings.Contains(content, "DYLD_LIBRARY_PATH") {
		t.Error("shim should reference DYLD_LIBRARY_PATH for macOS")
	}
	if !strings.Contains(content, "LIB_PATH_VAR") {
		t.Error("shim should use LIB_PATH_VAR for platform detection")
	}
	if !strings.Contains(content, `exec "$PHPV_PREFIX/bin/php" "$@"`) {
		t.Error("binary shim should exec php from prefix")
	}
	if !strings.Contains(content, "command -v php") {
		t.Error("binary shim should have system block with command -v")
	}
}

func TestRenderShim_DropsParentLdLibraryPath(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if strings.Contains(content, `"$PHPV_PREFIX/lib:$LD_LIBRARY_PATH"`) {
		t.Error("shim should NOT prepend to parent LD_LIBRARY_PATH")
	}
	if strings.Contains(content, `"$PHPV_PREFIX/lib:$DYLD_LIBRARY_PATH"`) {
		t.Error("shim should NOT prepend to parent DYLD_LIBRARY_PATH")
	}
	if !strings.Contains(content, "PHPV_EXTRA_LD_LIBRARY_PATH") {
		t.Error("shim should support PHPV_EXTRA_LD_LIBRARY_PATH allowlist")
	}
}

func TestRenderShim_HasCleanRoomBlock(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, `eval "$LIB_PATH_VAR=\"\$PHPV_PREFIX/lib\""`) {
		t.Error("shim should start LIB_PATH_VAR with only PHP prefix")
	}
	if !strings.Contains(content, "export $LIB_PATH_VAR") {
		t.Error("shim should export LIB_PATH_VAR")
	}
}

func TestRenderShim_SystemModeUnaffected(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	// The system block should exec the system binary without touching LD_LIBRARY_PATH.
	if !strings.Contains(content, `exec "$PHP_PATH" "$@"`) {
		t.Error("system block should exec system binary as-is")
	}
}

func TestRenderShim_Phar(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "wp", Kind: kindPhar, PharRel: "phar/wp-cli.phar"})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, `exec "$PHPV_PREFIX/bin/php" "$PHPV_PREFIX/phar/wp-cli.phar" "$@"`) {
		t.Error("phar shim should exec php with phar path")
	}
	if !strings.Contains(content, "command -v php") {
		t.Error("phar shim should have system block with command -v")
	}
	if strings.Contains(content, "COMPOSER_HOME") {
		t.Error("generic phar shim should not scope COMPOSER_HOME")
	}
}

func TestRenderShim_ComposerScopesHome(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "composer", Kind: kindPhar, PharRel: "phar/composer.phar"})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, `export COMPOSER_HOME="${COMPOSER_HOME:-$PHPV_PREFIX/composer}"`) {
		t.Error("composer shim should scope COMPOSER_HOME to the version prefix")
	}
	if !strings.Contains(content, `"$PHPV_PREFIX/bin/php" "$PHPV_PREFIX/phar/composer.phar" "$@"`) {
		t.Error("composer shim should still run the phar with the version's php")
	}
	// The per-version home must only apply after the system-mode block
	// (system mode delegates before reaching it).
	sysIdx := strings.Index(content, ".phpv_system")
	homeIdx := strings.Index(content, "COMPOSER_HOME")
	if sysIdx == -1 || homeIdx == -1 || homeIdx < sysIdx {
		t.Error("COMPOSER_HOME scoping should come after the system-mode block")
	}
}

func TestRenderShim_ComposerRehashAfterGlobal(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "composer", Kind: kindPhar, PharRel: "phar/composer.phar"})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, `[ "${1:-}" = "global" ]`) {
		t.Error("composer shim should detect `global` as first argument")
	}
	if !strings.Contains(content, `rehash`) {
		t.Error("composer shim should trigger a rehash after global commands")
	}
	if !strings.Contains(content, `exit "$_composer_status"`) {
		t.Error("composer shim must preserve composer's exit code")
	}
	// The main path must NOT exec: it has to fall through to the rehash
	// logic after composer exits. (The system-mode block still execs.)
	if strings.Contains(content, `exec "$PHPV_PREFIX/bin/php" "$PHPV_PREFIX/phar/composer.phar"`) {
		t.Error("composer main path must not exec (rehash runs after composer exits)")
	}
}

func TestRenderShim_VendorBin(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "laravel", Kind: kindVendorBin})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, `VBIN="$PHPV_PREFIX/composer/vendor/bin/laravel"`) {
		t.Error("vendor-bin shim should point at the version-scoped vendor bin")
	}
	if !strings.Contains(content, `exec "$PHPV_PREFIX/bin/php" "$VBIN" "$@"`) {
		t.Error("vendor-bin shim should run the bin through the version's php")
	}
	if !strings.Contains(content, `command -v laravel`) {
		t.Error("vendor-bin shim should have a system-mode fallback via command -v")
	}
}

func TestWriteShim_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	if err := s.WriteShim(shimDef{Name: "php", Kind: kindBinary}); err != nil {
		t.Fatalf("WriteShim: %v", err)
	}
	path := filepath.Join(dir, "php")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat shim: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("shim should be executable")
	}
}

func TestWriteAll_CreatesAllShims(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	if err := s.WriteAll(); err != nil {
		t.Fatalf("WriteAll: %v", err)
	}
	expected := []string{"php", "phpize", "php-config", "php-cgi", "phpdbg"}
	for _, name := range expected {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("shim %s not created", name)
		}
	}
}

func TestRegenerateAll_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	oldPath := filepath.Join(dir, "php")
	if err := os.WriteFile(oldPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.RegenerateAll(); err != nil {
		t.Fatalf("RegenerateAll: %v", err)
	}
	data, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "old content") {
		t.Error("shim was not overwritten")
	}
}

func TestWritePhar(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	if err := s.WritePhar("composer", "phar/composer.phar"); err != nil {
		t.Fatalf("WritePhar: %v", err)
	}
	path := filepath.Join(dir, "composer")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("phar shim not created")
	}
}

// fakeSiloRepo is a minimal SiloRepository that only answers GetSilo.
type fakeSiloRepo struct {
	root string
}

func (f *fakeSiloRepo) Download(url string, checksumType string, checksumValue string) (bool, error) {
	return false, nil
}
func (f *fakeSiloRepo) Extract(archivePath string, destDir string) (bool, error) {
	return false, nil
}
func (f *fakeSiloRepo) GetSilo() domain.Silo { return domain.Silo{Root: f.root} }
func (f *fakeSiloRepo) GetState(name, version string) (domain.InstallState, error) {
	return domain.StateNone, nil
}
func (f *fakeSiloRepo) MarkInProgress(name, version string) error  { return nil }
func (f *fakeSiloRepo) MarkComplete(name, version string) error    { return nil }
func (f *fakeSiloRepo) MarkFailed(name, version string) error      { return nil }
func (f *fakeSiloRepo) MarkInterrupted(name, version string) error { return nil }
func (f *fakeSiloRepo) GetDefault() (string, error)                { return "", nil }
func (f *fakeSiloRepo) SetDefault(version string) error            { return nil }
func (f *fakeSiloRepo) PHPOutputPath(phpVersion string) string     { return "" }
func (f *fakeSiloRepo) SourcePath(pkg, version string) string      { return "" }
func (f *fakeSiloRepo) PackagePrefix(name, version string) string {
	return filepath.Join(f.root, "packages", name, version)
}
func (f *fakeSiloRepo) PECLArchivePath(name, version string) string { return "" }
func (f *fakeSiloRepo) BuildLogPath(pkg, version, logName string) string {
	return ""
}
func (f *fakeSiloRepo) GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error) {
	return nil, nil
}
func (f *fakeSiloRepo) SaveExtensionManifest(phpVersion string, m *domain.ExtensionManifest) error {
	return nil
}
func (f *fakeSiloRepo) IsSystemMode() bool               { return false }
func (f *fakeSiloRepo) SetSystemMode(enabled bool) error { return nil }

func newRegenerateService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	siloSvc := silo.NewService(&fakeSiloRepo{root: root}, nil)
	return NewService(siloSvc)
}

func TestRegenerateAll_VendorBinShims(t *testing.T) {
	s := newRegenerateService(t)
	root := s.silo.GetSilo().Root

	// PHP 8.4.0 with a phar composer and a PHP vendor bin.
	phpDir := filepath.Join(root, "packages", "php", "8.4.0")
	vendorBin := filepath.Join(phpDir, "composer", "vendor", "bin")
	if err := os.MkdirAll(filepath.Join(phpDir, "phar"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(vendorBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(phpDir, "phar", "composer.phar"), []byte("phar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorBin, "laravel"), []byte("#!/usr/bin/env php\n<?php echo 'laravel';\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A non-PHP vendor bin must not get a shim.
	if err := os.WriteFile(filepath.Join(vendorBin, "some-shell-tool"), []byte("#!/bin/bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := s.RegenerateAll(); err != nil {
		t.Fatalf("RegenerateAll: %v", err)
	}

	for _, name := range []string{"php", "composer", "laravel"} {
		if _, err := os.Stat(filepath.Join(s.bin, name)); os.IsNotExist(err) {
			t.Errorf("shim %s not created", name)
		}
	}
	if _, err := os.Stat(filepath.Join(s.bin, "some-shell-tool")); !os.IsNotExist(err) {
		t.Errorf("non-PHP vendor bin should not get a shim")
	}

	// The laravel shim must be the vendor-bin kind (version-scoped path).
	data, err := os.ReadFile(filepath.Join(s.bin, "laravel"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `VBIN="$PHPV_PREFIX/composer/vendor/bin/laravel"`) {
		t.Error("laravel shim should be a vendor-bin shim")
	}

	// The composer shim must be the special composer kind.
	data, err = os.ReadFile(filepath.Join(s.bin, "composer"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "COMPOSER_HOME") {
		t.Error("composer shim should scope COMPOSER_HOME")
	}
}

func TestRegenerateAll_VendorBinNamePrecedence(t *testing.T) {
	s := newRegenerateService(t)
	root := s.silo.GetSilo().Root

	// PHP 8.4.0 with a phar named `wp` and a vendor bin also named `wp`:
	// the phar must win.
	phpDir := filepath.Join(root, "packages", "php", "8.4.0")
	vendorBin := filepath.Join(phpDir, "composer", "vendor", "bin")
	if err := os.MkdirAll(filepath.Join(phpDir, "phar"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(vendorBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(phpDir, "phar", "wp.phar"), []byte("phar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorBin, "wp"), []byte("#!/usr/bin/env php\n<?php\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := s.RegenerateAll(); err != nil {
		t.Fatalf("RegenerateAll: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(s.bin, "wp"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `phar/wp.phar`) {
		t.Error("phar shim should win over vendor bin with the same name")
	}
}

func TestRegenerateAll_PruneStaleShims(t *testing.T) {
	s := newRegenerateService(t)

	// An orphaned phpv-generated shim (e.g. vendor bin removed via
	// `composer global remove`) must be pruned.
	stale := filepath.Join(s.bin, "orphan-tool")
	if err := os.MkdirAll(s.bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("#!/bin/bash\n# Auto-generated by phpv. Do not edit.\nexec nothing\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A phpv binary and a user file must survive.
	phpvBin := filepath.Join(s.bin, "phpv")
	if err := os.WriteFile(phpvBin, []byte("ELF-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(s.bin, "my-custom-tool")
	if err := os.WriteFile(userFile, []byte("#!/bin/bash\necho custom\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := s.RegenerateAll(); err != nil {
		t.Fatalf("RegenerateAll: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("orphaned phpv-generated shim should be pruned")
	}
	if _, err := os.Stat(phpvBin); os.IsNotExist(err) {
		t.Error("phpv binary must never be pruned")
	}
	if _, err := os.Stat(userFile); os.IsNotExist(err) {
		t.Error("non-phpv files in bin dir must never be pruned")
	}
}

func TestRegenerateAll_BinaryShimNameWinsOverVendorBin(t *testing.T) {
	s := newRegenerateService(t)
	root := s.silo.GetSilo().Root

	// A vendor bin literally named `php` must not clobber the default
	// binary shim.
	phpDir := filepath.Join(root, "packages", "php", "8.4.0")
	vendorBin := filepath.Join(phpDir, "composer", "vendor", "bin")
	if err := os.MkdirAll(vendorBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorBin, "php"), []byte("#!/usr/bin/env php\n<?php\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := s.RegenerateAll(); err != nil {
		t.Fatalf("RegenerateAll: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(s.bin, "php"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `exec "$PHPV_PREFIX/bin/php" "$@"`) {
		t.Error("default binary shim must win over vendor bin named php")
	}
}
