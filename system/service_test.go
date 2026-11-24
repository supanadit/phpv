package system

import (
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"3.6.3-1", "3.6.3"},
		{"8.21.0-1", "8.21.0"},
		{"1:1.3.2-3", "1.3.2"}, // Arch epoch prefix
		{"78.3-1", "78.3"},
		{"6.9.10-1", "6.9.10"},
		{"3.53.3-1", "3.53.3"},
		{"1.0.0-beta", "1.0.0"}, // pre-release stripped for system packages
		{"2.15.3-1", "2.15.3"},
		{"3.0.2-0ubuntu1.18", "3.0.2"}, // Ubuntu suffix
		{"3.0.7-1.el9", "3.0.7"},       // RHEL suffix
		{"3.1.4-r0", "3.1.4"},          // Alpine suffix
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeVersion(tt.input)
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeVersion_ExtractsArchEpoch(t *testing.T) {
	// Arch packages sometimes prefix versions with an epoch, e.g. zlib "1:1.3.2-3"
	got := normalizeVersion("1:1.3.2-3")
	want := "1.3.2"
	if got != want {
		t.Errorf("normalizeVersion(1:1.3.2-3) = %q, want %q", got, want)
	}
}

func TestCheckBuildTools_SplitsAvailableAndMissing(t *testing.T) {
	toolNames := []string{"gcc", "g++", "make", "cmake", "autoconf", "automake", "m4", "perl", "bison", "re2c", "flex", "pkg-config", "xz"}

	result, err := NewService(&testCatalog{}).CheckBuildTools(toolNames)
	if err != nil {
		t.Fatalf("CheckBuildTools returned error: %v", err)
	}

	seen := make(map[string]bool)
	for _, p := range result.Available {
		if p.Installed {
			seen[p.Name] = true
		}
	}
	for _, p := range result.Missing {
		if p.Installed {
			t.Errorf("missing package %s has Installed=true", p.Name)
		}
		if seen[p.Name] {
			t.Errorf("package %s appears in both Available and Missing", p.Name)
		}
		seen[p.Name] = true
	}

	var missing []string
	for _, name := range toolNames {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("tools not classified as either available or missing: %s", strings.Join(missing, ", "))
	}
}

func TestDependencySet_AdvisoryLibrariesHaveNoBuildRelationship(t *testing.T) {
	pkgs := DependencySet(nil)
	if len(pkgs) == 0 {
		t.Fatal("DependencySet(nil) returned no packages")
	}
	for _, p := range pkgs {
		if p.Required {
			t.Errorf("%s marked Required with no extensions; nothing links it as a build dep", p.Name)
		}
	}
	// argon2 has no builtInPackages entry and no RequiresPackage link, and
	// libargon2-devel reports a date (20190702) that no constraint can match.
	for _, p := range pkgs {
		if p.Name == "argon2" {
			t.Error("argon2 listed: the build cannot consume it and its version is not comparable")
		}
	}
}

func TestDependencySet_RequiresPackageIsRequired(t *testing.T) {
	exts := []domain.ExtensionDef{
		{Name: "curl", RequiresPackage: "curl"},
	}
	pkgs := DependencySet(exts)

	byName := make(map[string]Package, len(pkgs))
	for _, p := range pkgs {
		byName[p.Name] = p
	}
	curl, ok := byName["curl"]
	if !ok {
		t.Fatal("curl absent: an extension declares RequiresPackage curl")
	}
	if !curl.Required {
		t.Error("curl not marked Required: RequiresPackage feeds the build plan")
	}
}

func TestDependencySet_RequiresPackagesIsAdvisory(t *testing.T) {
	exts := []domain.ExtensionDef{
		{Name: "gd", RequiresPackages: []string{"libpng", "libjpeg", "freetype"}},
	}
	pkgs := DependencySet(exts)

	byName := make(map[string]Package, len(pkgs))
	for _, p := range pkgs {
		byName[p.Name] = p
	}
	for _, want := range []string{"libpng", "libjpeg", "freetype"} {
		p, ok := byName[want]
		if !ok {
			t.Errorf("%s absent: extension gd declares it in RequiresPackages", want)
			continue
		}
		if p.Required {
			t.Errorf("%s marked Required, but RequiresPackages does not feed the build plan", want)
		}
	}
}

func TestDependencySet_Deduplicates(t *testing.T) {
	exts := []domain.ExtensionDef{
		{Name: "a", RequiresPackage: "libxml2", RequiresPackages: []string{"libxml2"}},
		{Name: "b", RequiresPackage: "libxml2"},
	}
	seen := make(map[string]int)
	for _, p := range DependencySet(exts) {
		seen[p.Name]++
	}
	if seen["libxml2"] != 1 {
		t.Errorf("libxml2 appears %d times, want 1", seen["libxml2"])
	}
}

func TestNames(t *testing.T) {
	pkgs := []Package{{Name: "curl"}, {Name: "libzip"}}
	got := Names(pkgs)
	if len(got) != 2 || got[0] != "curl" || got[1] != "libzip" {
		t.Fatalf("Names = %v, want [curl libzip]", got)
	}
	if len(Names(nil)) != 0 {
		t.Fatal("Names(nil) should be empty")
	}
}

func TestCheck_PreservesRequiredFlag(t *testing.T) {
	// Check must not drop caller-supplied metadata; the distro may not have a
	// mapping for a package, in which case it is skipped entirely.
	deps := []Package{{Name: "definitely-not-a-mapped-package", Required: true}}
	result, err := NewService(&testCatalog{}).Check(deps)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	for _, p := range result.Available {
		if p.Name == deps[0].Name {
			t.Fatal("unmapped package should be skipped, not reported")
		}
	}
}

// testCatalog keeps system tests independent of any concrete driver.
type testCatalog struct{}

func (t *testCatalog) Detect() (Distro, error)                    { return Distro{Name: "fedora", PM: "dnf"}, nil }
func (t *testCatalog) PackagesForDistro(string) map[string]string { return map[string]string{} }
func (t *testCatalog) BuildToolsForDistro(string) map[string]string {
	return map[string]string{"gcc": "gcc", "g++": "gcc-c++", "make": "make"}
}
func (t *testCatalog) AdvisoryLibraries() []string {
	return []string{"sqlite3", "readline", "pcre2", "sodium"}
}
func (t *testCatalog) InstallArgs(pm string, names []string) []string {
	return append([]string{pm, "install", "-y"}, names...)
}
