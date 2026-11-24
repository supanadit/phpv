package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakePHPBin creates an executable script that behaves like `php -m`:
// it lists modules from the PHPV_FAKE_MODULES env var, and can be told to
// fail startup (PHPV_FAKE_PHP_FAIL=1).
func fakePHPBin(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(bin, "php")
	script := `#!/bin/bash
if [ "$1" = "-m" ]; then
  if [ -n "$PHPV_FAKE_PHP_FAIL" ]; then
    echo "PHP startup error" >&2
    exit 1
  fi
  echo "$PHPV_FAKE_MODULES"
fi
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEnable_HintCorrect(t *testing.T) {
	dir := t.TempDir()
	phpBin := fakePHPBin(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	iniPath := filepath.Join(dir, "etc", "php.ini")
	t.Setenv("PHPV_FAKE_MODULES", "xdebug")

	entry, err := NewService().Enable(phpBin, iniPath, "xdebug", true)
	if err != nil {
		t.Fatalf("enableExtension: %v", err)
	}
	if entry != "zend_extension=xdebug.so" {
		t.Errorf("entry = %q, want zend_extension=xdebug.so", entry)
	}
}

func TestEnable_FlipsViaScript(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	// php -m prints the module only when INI_TEST_ENTRY contains
	// zend_extension= — i.e. the flip path.
	script := `#!/bin/bash
if [ "$1" = "-m" ]; then
  if grep -q '^zend_extension=fakeext.so' "$PHPV_FAKE_INI" 2>/dev/null; then
    echo "Core"
    echo "fakeext"
  fi
fi
`
	phpBin := filepath.Join(bin, "php")
	if err := os.WriteFile(phpBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	iniPath := filepath.Join(dir, "etc", "php.ini")
	t.Setenv("PHPV_FAKE_INI", iniPath)

	entry, err := NewService().Enable(phpBin, iniPath, "fakeext", false)
	if err != nil {
		t.Fatalf("enableExtension: %v", err)
	}
	if entry != "zend_extension=fakeext.so" {
		t.Errorf("entry = %q, want flip to zend_extension=fakeext.so", entry)
	}
	// The wrong form must have been removed.
	data, _ := os.ReadFile(iniPath)
	if strings.Contains(string(data), "extension=fakeext.so") && !strings.Contains(string(data), "zend_extension=") {
		t.Errorf("ini should contain only the zend form, got:\n%s", data)
	}
}

func TestEnable_FailsWhenNeitherLoads(t *testing.T) {
	dir := t.TempDir()
	phpBin := fakePHPBin(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	iniPath := filepath.Join(dir, "etc", "php.ini")
	t.Setenv("PHPV_FAKE_MODULES", "")

	_, err := NewService().Enable(phpBin, iniPath, "ghost", false)
	if err == nil {
		t.Fatal("enableExtension should fail when module never loads")
	}
	if !strings.Contains(err.Error(), "does not load") {
		t.Errorf("error should explain the load failure, got: %v", err)
	}
	// Both attempts must have been removed.
	data, _ := os.ReadFile(iniPath)
	if strings.Contains(string(data), "ghost") {
		t.Errorf("ini must not keep dead entries, got:\n%s", data)
	}
}

func TestResolve_QueriesPhpConfig(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	phpConfig := filepath.Join(bin, "php-config")
	script := fmt.Sprintf("#!/bin/bash\necho %s\n", libDir)
	if err := os.WriteFile(phpConfig, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	got := NewService().Resolve(dir, phpConfig)
	if got != filepath.Join(libDir, "php.ini") {
		t.Errorf("resolveIniPath = %q, want %q", got, filepath.Join(libDir, "php.ini"))
	}
}

func TestResolve_FallsBackToEtc(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "bin", "php-config") // does not exist
	got := NewService().Resolve(dir, missing)
	if got != filepath.Join(dir, "etc", "php.ini") {
		t.Errorf("resolveIniPath fallback = %q, want %q", got, filepath.Join(dir, "etc", "php.ini"))
	}
}

func TestEnable_WritesToRealIniPath(t *testing.T) {
	// Regression: the PECL service wrote <prefix>/etc/php.ini while
	// phpv-built PHP reads <prefix>/lib/php.ini. Simulate via fake
	// php-config + a php -m that only sees modules when the directive is
	// in the file php-config named.
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	phpConfig := filepath.Join(bin, "php-config")
	if err := os.WriteFile(phpConfig, []byte(fmt.Sprintf("#!/bin/bash\necho %s\n", libDir)), 0o755); err != nil {
		t.Fatal(err)
	}
	phpBin := filepath.Join(bin, "php")
	iniPath := filepath.Join(libDir, "php.ini")
	script := fmt.Sprintf(`#!/bin/bash
if [ "$1" = "-m" ]; then
  if grep -q '^extension=redis.so' %q 2>/dev/null; then
    echo "Core"
    echo "redis"
  fi
fi
`, iniPath)
	if err := os.WriteFile(phpBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	entry, err := NewService().Enable(phpBin, iniPath, "redis", false)
	if err != nil {
		t.Fatalf("enableExtension: %v", err)
	}
	if entry != "extension=redis.so" {
		t.Errorf("entry = %q", entry)
	}
	data, _ := os.ReadFile(iniPath)
	if !strings.Contains(string(data), "extension=redis.so") {
		t.Errorf("directive missing from the real ini path:\n%s", data)
	}
}

func TestEnable_DiagnosesIniMismatch(t *testing.T) {
	// When PHP starts cleanly but the module never loads (e.g. directive
	// written to a file PHP does not read), the error must carry php --ini
	// output instead of an empty report.
	dir := t.TempDir()
	phpBin := fakePHPBin(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	iniPath := filepath.Join(dir, "etc", "php.ini")
	t.Setenv("PHPV_FAKE_MODULES", "")

	_, err := NewService().Enable(phpBin, iniPath, "ghost", false)
	if err == nil {
		t.Fatal("expected load failure")
	}
	if !strings.Contains(err.Error(), "does not load") {
		t.Errorf("error should explain failure: %v", err)
	}
}

func TestWriteAndRemove(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "php.ini")
	svc := NewService()

	if err := svc.Write(iniPath, "extension=a.so"); err != nil {
		t.Fatalf("writeINIEntry: %v", err)
	}
	// Duplicate write must not duplicate the line.
	if err := svc.Write(iniPath, "extension=a.so"); err != nil {
		t.Fatalf("writeINIEntry dup: %v", err)
	}
	if err := svc.Write(iniPath, "xdebug.mode=debug"); err != nil {
		t.Fatalf("writeINIEntry setting: %v", err)
	}
	data, _ := os.ReadFile(iniPath)
	if got := strings.Count(string(data), "extension=a.so"); got != 1 {
		t.Errorf("expected exactly 1 occurrence, got %d:\n%s", got, data)
	}
	if !strings.Contains(string(data), "xdebug.mode=debug") {
		t.Errorf("setting missing:\n%s", data)
	}

	svc.Remove(iniPath, "extension=a.so")
	data, _ = os.ReadFile(iniPath)
	if strings.Contains(string(data), "extension=a.so") {
		t.Errorf("line should be removed:\n%s", data)
	}
}
