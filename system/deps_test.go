package system

import (
	"testing"

	"github.com/supanadit/phpv/domain"
)

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
	result, err := (&Service{}).Check(deps)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	for _, p := range result.Available {
		if p.Name == deps[0].Name {
			t.Fatal("unmapped package should be skipped, not reported")
		}
	}
}
