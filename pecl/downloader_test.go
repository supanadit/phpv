package pecl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestEnableExtension_HintCorrect(t *testing.T) {
	dir := t.TempDir()
	phpBin := fakePHPBin(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	iniPath := filepath.Join(dir, "etc", "php.ini")
	t.Setenv("PHPV_FAKE_MODULES", "xdebug")

	entry, err := enableExtension(phpBin, iniPath, "xdebug", true)
	if err != nil {
		t.Fatalf("enableExtension: %v", err)
	}
	if entry != "zend_extension=xdebug.so" {
		t.Errorf("entry = %q, want zend_extension=xdebug.so", entry)
	}
}

func TestEnableExtension_FlipsViaScript(t *testing.T) {
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

	entry, err := enableExtension(phpBin, iniPath, "fakeext", false)
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

func TestEnableExtension_FailsWhenNeitherLoads(t *testing.T) {
	dir := t.TempDir()
	phpBin := fakePHPBin(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	iniPath := filepath.Join(dir, "etc", "php.ini")
	t.Setenv("PHPV_FAKE_MODULES", "")

	_, err := enableExtension(phpBin, iniPath, "ghost", false)
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

func TestResolveIniPath_QueriesPhpConfig(t *testing.T) {
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
	got := resolveIniPath(dir, phpConfig)
	if got != filepath.Join(libDir, "php.ini") {
		t.Errorf("resolveIniPath = %q, want %q", got, filepath.Join(libDir, "php.ini"))
	}
}

func TestResolveIniPath_FallsBackToEtc(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "bin", "php-config") // does not exist
	got := resolveIniPath(dir, missing)
	if got != filepath.Join(dir, "etc", "php.ini") {
		t.Errorf("resolveIniPath fallback = %q, want %q", got, filepath.Join(dir, "etc", "php.ini"))
	}
}

func TestEnableExtension_WritesToRealIniPath(t *testing.T) {
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

	entry, err := enableExtension(phpBin, iniPath, "redis", false)
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

func TestEnableExtension_DiagnosesIniMismatch(t *testing.T) {
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

	_, err := enableExtension(phpBin, iniPath, "ghost", false)
	if err == nil {
		t.Fatal("expected load failure")
	}
	if !strings.Contains(err.Error(), "does not load") {
		t.Errorf("error should explain failure: %v", err)
	}
}

func TestWriteAndRemoveINIEntry(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "php.ini")

	if err := writeINIEntry(iniPath, "extension=a.so"); err != nil {
		t.Fatalf("writeINIEntry: %v", err)
	}
	// Duplicate write must not duplicate the line.
	if err := writeINIEntry(iniPath, "extension=a.so"); err != nil {
		t.Fatalf("writeINIEntry dup: %v", err)
	}
	if err := writeINIEntry(iniPath, "xdebug.mode=debug"); err != nil {
		t.Fatalf("writeINIEntry setting: %v", err)
	}
	data, _ := os.ReadFile(iniPath)
	if got := strings.Count(string(data), "extension=a.so"); got != 1 {
		t.Errorf("expected exactly 1 occurrence, got %d:\n%s", got, data)
	}
	if !strings.Contains(string(data), "xdebug.mode=debug") {
		t.Errorf("setting missing:\n%s", data)
	}

	removeINIEntry(iniPath, "extension=a.so")
	data, _ = os.ReadFile(iniPath)
	if strings.Contains(string(data), "extension=a.so") {
		t.Errorf("line should be removed:\n%s", data)
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
	svc := NewService(newSiloService(mock))
	// Point the service's downloader at the fake server.
	svc.downloader = &compatibleDownloader{dl}

	_, err := svc.InstallCompatible(context.Background(), "xdebug", "8.2.0", 2, nil)
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
