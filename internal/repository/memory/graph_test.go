package memory

import (
	"testing"

	"github.com/supanadit/phpv/graph"
)

func TestGraphRepository_GetOrderedDependencies_PHP7_4(t *testing.T) {
	repo := NewGraphRepository()
	svc := graph.NewService(repo)

	defaults := []string{
		"bcmath", "curl", "dom", "fileinfo", "filter", "gd",
		"iconv", "intl", "json", "mbstring", "openssl", "opcache",
		"pdo", "pdo_mysql", "pdo_sqlite", "phar", "session",
		"simplexml", "sqlite3", "tokenizer", "xml", "xmlreader",
		"xmlwriter", "zip", "zlib",
	}

	plan, err := svc.GetBuildPlan("php", "7.4.33", defaults)
	if err != nil {
		t.Fatalf("GetBuildPlan(php, 7.4.33) returned error: %v", err)
	}

	depNames := make(map[string]bool)
	for _, dep := range plan.Deps {
		depNames[dep.Name] = true
	}

	expectedDirect := []string{"openssl", "libxml2", "zlib", "oniguruma", "curl"}
	for _, name := range expectedDirect {
		if !depNames[name] {
			t.Errorf("expected dependency %q in result, not found", name)
		}
	}

	expectedTransitive := []string{"m4", "autoconf", "automake", "libtool", "perl"}
	for _, name := range expectedTransitive {
		if !depNames[name] {
			t.Errorf("expected transitive dependency %q in result, not found", name)
		}
	}

	if depNames["php"] {
		t.Error("root package 'php' should not appear in its own dependency list")
	}
}

func TestGraphRepository_GetOrderedDependencies_PHP8_1(t *testing.T) {
	repo := NewGraphRepository()
	svc := graph.NewService(repo)

	defaults := []string{
		"bcmath", "curl", "dom", "fileinfo", "filter", "gd",
		"iconv", "intl", "json", "mbstring", "openssl", "opcache",
		"pdo", "pdo_mysql", "pdo_sqlite", "phar", "session",
		"simplexml", "sqlite3", "tokenizer", "xml", "xmlreader",
		"xmlwriter", "zip", "zlib",
	}

	plan, err := svc.GetBuildPlan("php", "8.1.0", defaults)
	if err != nil {
		t.Fatalf("GetBuildPlan(php, 8.1.0) returned error: %v", err)
	}

	depMap := make(map[string]string)
	for _, dep := range plan.Deps {
		depMap[dep.Name] = dep.Version
	}

	if v, ok := depMap["openssl"]; !ok || v != "1.1.1w|>=1.0.2,<4.0.0" {
		t.Errorf("openssl version = %q, want %q", v, "1.1.1w|>=1.0.2,<4.0.0")
	}
	if v, ok := depMap["curl"]; !ok || v != "8.10.1|>=8.0.0" {
		t.Errorf("curl version = %q, want %q", v, "8.10.1|>=8.0.0")
	}
}

func TestGraphRepository_GetOrderedDependencies_Zlib_NoDeps(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("zlib", "1.2.13")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(zlib, 1.2.13) returned error: %v", err)
	}

	if len(deps) != 0 {
		t.Fatalf("zlib should have zero dependencies, got %d: %v", len(deps), deps)
	}
}

func TestGraphRepository_GetOrderedDependencies_M4_NoDeps(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("m4", "1.4.19")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(m4, 1.4.19) returned error: %v", err)
	}

	if len(deps) != 0 {
		t.Fatalf("m4 should have zero dependencies, got %d: %v", len(deps), deps)
	}
}

func TestGraphRepository_GetOrderedDependencies_Deduplication(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("php", "7.4.33")
	if err != nil {
		t.Fatalf("GetOrderedDependencies returned error: %v", err)
	}

	seen := make(map[string]int)
	for _, dep := range deps {
		seen[dep.Name]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("dependency %q appears %d times, should appear only once", name, count)
		}
	}
}

func TestGraphRepository_GetOrderedDependencies_Ordering(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("autoconf", "2.50")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(autoconf, 2.50) returned error: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("autoconf 2.50 should have 1 dependency (m4), got %d: %v", len(deps), deps)
	}
	if deps[0].Name != "m4" {
		t.Fatalf("expected m4, got %s", deps[0].Name)
	}
}

func TestGraphRepository_GetOrderedDependencies_UnknownPackage(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("unknown", "1.0.0")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(unknown) returned unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Fatalf("unknown package should have zero deps, got %d", len(deps))
	}
}

func TestDefaultExtensions_SkipsBuiltIn(t *testing.T) {
	repo := NewGraphRepository()

	// PHP 8.5+: iconv is shared-only (IsBuiltIn: false, Flag: ""), so it
	// should be included in the default set (it needs a phpize build).
	included, skipped := repo.DefaultExtensions("8.5.8")
	hasIconv := false
	for _, name := range included {
		if name == "iconv" {
			hasIconv = true
			break
		}
	}
	if !hasIconv {
		t.Error("iconv should be in default extensions for PHP 8.5+ (shared-only, not built-in)")
	}
	_ = skipped

	// If we had a built-in extension (IsBuiltIn: true), it should be skipped.
	// Currently no extension uses IsBuiltIn: true, so this is a future-proofing test.
	// When a future PHP version makes an extension built-in, add a test here.
}

func TestSharedOnlyExtensions_IncludesSharedOnly(t *testing.T) {
	repo := NewGraphRepository()

	// PHP 8.5+: iconv is shared-only (IsBuiltIn: false, Flag: ""), so it
	// should appear in the shared-only list (needs phpize build).
	sharedOnly := repo.SharedOnlyExtensions("8.5.8", []string{"iconv"})
	hasIconv := false
	for _, name := range sharedOnly {
		if name == "iconv" {
			hasIconv = true
			break
		}
	}
	if !hasIconv {
		t.Error("iconv should be in shared-only extensions for PHP 8.5+ (Flag: '', IsBuiltIn: false)")
	}
}

func TestIsBuiltInForVersion(t *testing.T) {
	repo := NewGraphRepository()

	// iconv for PHP 8.5+: IsBuiltIn: false, so isBuiltInForVersion should return false.
	def, ok := repo.extensions["iconv"]
	if !ok {
		t.Fatal("iconv not found in extensions registry")
	}
	if isBuiltInForVersion(def, "8.5.8") {
		t.Error("iconv should NOT be built-in for PHP 8.5+ (IsBuiltIn: false)")
	}
	if isBuiltInForVersion(def, "8.4.0") {
		t.Error("iconv should NOT be built-in for PHP 8.4 (uses --with-iconv flag)")
	}
	if isBuiltInForVersion(def, "7.4.0") {
		t.Error("iconv should NOT be built-in for PHP 7.4 (uses --with-iconv flag)")
	}
}

func TestGetExtensionConfigureFlags_Iconv_PHP8_0(t *testing.T) {
	repo := NewGraphRepository()
	flags := repo.GetExtensionConfigureFlags("iconv", "8.0.30")
	if len(flags) == 0 {
		t.Fatal("iconv should have configure flags for PHP 8.0")
	}
	if flags[0] != "--with-iconv" {
		t.Errorf("iconv flag for PHP 8.0 = %q, want %q", flags[0], "--with-iconv")
	}
}

func TestGetExtensionConfigureFlags_Iconv_PHP8_5(t *testing.T) {
	repo := NewGraphRepository()
	flags := repo.GetExtensionConfigureFlags("iconv", "8.5.8")
	if len(flags) != 0 {
		t.Errorf("iconv should have no configure flags for PHP 8.5+, got %v", flags)
	}
}

func TestGetExtensionConfigureFlags_Gd_PHP7_4(t *testing.T) {
	repo := NewGraphRepository()
	// gd uses FlagVersions: >=7.4 uses --enable-gd, <7.4 uses --with-gd
	flags := repo.GetExtensionConfigureFlags("gd", "7.4.33")
	if len(flags) == 0 {
		t.Fatal("gd should have configure flags for PHP 7.4")
	}
	if flags[0] != "--enable-gd" {
		t.Errorf("gd flag for PHP 7.4 = %q, want %q", flags[0], "--enable-gd")
	}
}

func TestGetExtensionConfigureFlags_Gd_PHP7_3(t *testing.T) {
	repo := NewGraphRepository()
	// gd uses FlagVersions: >=7.4 uses --enable-gd, <7.4 uses --with-gd
	flags := repo.GetExtensionConfigureFlags("gd", "7.3.0")
	if len(flags) == 0 {
		t.Fatal("gd should have configure flags for PHP 7.3")
	}
	if flags[0] != "--with-gd" {
		t.Errorf("gd flag for PHP 7.3 = %q, want %q", flags[0], "--with-gd")
	}
}

func TestGetExtensionConfigureFlags_Zip_PHP7_4(t *testing.T) {
	repo := NewGraphRepository()
	// zip uses FlagVersions: >=7.4 uses --with-zip, <7.4 uses --enable-zip
	flags := repo.GetExtensionConfigureFlags("zip", "7.4.33")
	if len(flags) == 0 {
		t.Fatal("zip should have configure flags for PHP 7.4")
	}
	if flags[0] != "--with-zip" {
		t.Errorf("zip flag for PHP 7.4 = %q, want %q", flags[0], "--with-zip")
	}
}

func TestGetExtensionConfigureFlags_Zip_PHP7_3(t *testing.T) {
	repo := NewGraphRepository()
	// zip uses FlagVersions: >=7.4 uses --with-zip, <7.4 uses --enable-zip
	flags := repo.GetExtensionConfigureFlags("zip", "7.3.0")
	if len(flags) == 0 {
		t.Fatal("zip should have configure flags for PHP 7.3")
	}
	if flags[0] != "--enable-zip" {
		t.Errorf("zip flag for PHP 7.3 = %q, want %q", flags[0], "--enable-zip")
	}
}

func TestGetBuildPlan_PHP8_4_HasDeps(t *testing.T) {
	repo := NewGraphRepository()
	svc := graph.NewService(repo)

	defaults := []string{
		"bcmath", "curl", "dom", "fileinfo", "filter", "gd",
		"iconv", "intl", "json", "mbstring", "openssl", "opcache",
		"pdo", "pdo_mysql", "pdo_sqlite", "phar", "session",
		"simplexml", "sqlite3", "tokenizer", "xml", "xmlreader",
		"xmlwriter", "zip", "zlib",
	}

	plan, err := svc.GetBuildPlan("php", "8.4.0", defaults)
	if err != nil {
		t.Fatalf("GetBuildPlan(php, 8.4.0) returned error: %v", err)
	}

	depMap := make(map[string]string)
	for _, dep := range plan.Deps {
		depMap[dep.Name] = dep.Version
	}

	// PHP 8.4 should get deps from extension Versions, not from php Constraints
	if v, ok := depMap["openssl"]; !ok {
		t.Error("openssl dep missing for PHP 8.4")
	} else if v != "1.1.1w|>=1.1.1,<4.0.0" {
		t.Errorf("openssl version = %q, want %q", v, "1.1.1w|>=1.1.1,<4.0.0")
	}
	if v, ok := depMap["libxml2"]; !ok {
		t.Error("libxml2 dep missing for PHP 8.4")
	} else if v != "2.12.7|~2.12.0" {
		t.Errorf("libxml2 version = %q, want %q", v, "2.12.7|~2.12.0")
	}
	if v, ok := depMap["curl"]; !ok {
		t.Error("curl dep missing for PHP 8.4")
	} else if v != "8.10.1|>=8.0.0" {
		t.Errorf("curl version = %q, want %q", v, "8.10.1|>=8.0.0")
	}
	if v, ok := depMap["zlib"]; !ok {
		t.Error("zlib dep missing for PHP 8.4")
	} else if v != "1.3.1|>=1.3.0" {
		t.Errorf("zlib version = %q, want %q", v, "1.3.1|>=1.3.0")
	}
	if v, ok := depMap["oniguruma"]; !ok {
		t.Error("oniguruma dep missing for PHP 8.4")
	} else if v != "6.9.9|~6.9.0" {
		t.Errorf("oniguruma version = %q, want %q", v, "6.9.9|~6.9.0")
	}
	if v, ok := depMap["icu"]; !ok {
		t.Error("icu dep missing for PHP 8.4")
	} else if v != "74.2|>=74.2" {
		t.Errorf("icu version = %q, want %q", v, "74.2|>=74.2")
	}
}

func TestGetBuildPlan_PHP8_2_HasDeps(t *testing.T) {
	repo := NewGraphRepository()
	svc := graph.NewService(repo)

	defaults := []string{
		"bcmath", "curl", "dom", "fileinfo", "filter", "gd",
		"iconv", "intl", "json", "mbstring", "openssl", "opcache",
		"pdo", "pdo_mysql", "pdo_sqlite", "phar", "session",
		"simplexml", "sqlite3", "tokenizer", "xml", "xmlreader",
		"xmlwriter", "zip", "zlib",
	}

	plan, err := svc.GetBuildPlan("php", "8.2.1", defaults)
	if err != nil {
		t.Fatalf("GetBuildPlan(php, 8.2.1) returned error: %v", err)
	}

	depMap := make(map[string]string)
	for _, dep := range plan.Deps {
		depMap[dep.Name] = dep.Version
	}

	if v, ok := depMap["openssl"]; !ok {
		t.Error("openssl dep missing for PHP 8.2")
	} else if v != "1.1.1w|>=1.0.2,<4.0.0" {
		t.Errorf("openssl version = %q, want %q", v, "1.1.1w|>=1.0.2,<4.0.0")
	}
	if v, ok := depMap["libxml2"]; !ok {
		t.Error("libxml2 dep missing for PHP 8.2")
	} else if v != "2.12.7|~2.12.0" {
		t.Errorf("libxml2 version = %q, want %q", v, "2.12.7|~2.12.0")
	}
}

func TestGetBuildPlan_Minimal_NoDeps(t *testing.T) {
	repo := NewGraphRepository()
	svc := graph.NewService(repo)

	plan, err := svc.GetBuildPlan("php", "8.4.0", nil)
	if err != nil {
		t.Fatalf("GetBuildPlan(php, 8.4.0, nil) returned error: %v", err)
	}
	if len(plan.Deps) != 0 {
		t.Errorf("expected 0 deps for --minimal, got %d: %v", len(plan.Deps), plan.Deps)
	}
}

func TestGetBuildPlan_ImpliedChains(t *testing.T) {
	repo := NewGraphRepository()
	svc := graph.NewService(repo)

	// Request only 'dom' — it should imply 'libxml' ext, which requires 'libxml2' pkg.
	plan, err := svc.GetBuildPlan("php", "8.4.0", []string{"dom"})
	if err != nil {
		t.Fatalf("GetBuildPlan(php, 8.4.0, [dom]) returned error: %v", err)
	}

	depMap := make(map[string]string)
	for _, dep := range plan.Deps {
		depMap[dep.Name] = dep.Version
	}

	if v, ok := depMap["libxml2"]; !ok {
		t.Error("libxml2 dep missing — dom should imply libxml ext which requires libxml2")
	} else if v != "2.12.7|~2.12.0" {
		t.Errorf("libxml2 version = %q, want %q", v, "2.12.7|~2.12.0")
	}
}
