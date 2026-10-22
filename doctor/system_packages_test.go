package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/system"
)

// fakeSystemService supplies package-manager state without touching the host.
type fakeSystemService struct {
	missing []system.Package
	checked []system.Package
}

func (f *fakeSystemService) DistroInfo() system.Distro {
	return system.Distro{Name: "fedora", PM: "dnf"}
}

func (f *fakeSystemService) Check(deps []system.Package) (*system.CheckResult, error) {
	f.checked = deps
	return &system.CheckResult{Distro: f.DistroInfo(), Missing: f.missing}, nil
}

func (f *fakeSystemService) InstallCommand(packages []system.Package) string {
	names := make([]string, 0, len(packages))
	for _, p := range packages {
		names = append(names, p.SystemName)
	}
	return "dnf install -y " + strings.Join(names, " ")
}

// installExtensions writes an extension manifest so installedExtensions sees
// these extensions present for version 8.4.0.
func installExtensions(t *testing.T, root string, exts ...string) {
	t.Helper()
	dir := filepath.Join(root, "packages", "php", "8.4.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	parts := make([]string, 0, len(exts))
	for _, e := range exts {
		parts = append(parts, `{"name":"`+e+`","type":"builtin","version":"8.4.0"}`)
	}
	manifest := `{"extensions":[` + strings.Join(parts, ",") + `]}`
	if err := os.WriteFile(filepath.Join(dir, "extensions.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestCheckSystemPackages_DoesNotCheckArgon2(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	installExtensions(t, root, "zip")

	sysSvc := &fakeSystemService{}
	svc := NewService(newFakeRepository(), nil, newTestGraphService())
	svc.sysSvc = sysSvc

	svc.checkSystemPackages()

	for _, p := range sysSvc.checked {
		if p.Name == "argon2" {
			t.Fatalf("argon2 checked; the build cannot consume it. checked=%v", system.Names(sysSvc.checked))
		}
	}
}

func TestCheckSystemPackages_SplitsRequiredAndAdvisory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	installExtensions(t, root, "zip")

	sysSvc := &fakeSystemService{missing: []system.Package{
		{Name: "libzip", SystemName: "libzip-devel", Required: true},
		{Name: "pcre2", SystemName: "pcre2-devel", Required: false},
	}}
	svc := NewService(newFakeRepository(), nil, newTestGraphService())
	svc.sysSvc = sysSvc

	issues := svc.checkSystemPackages()

	var requiredIssue, advisoryIssue *Issue
	for i := range issues {
		switch {
		case strings.Contains(issues[i].Title, "Missing build dependencies"):
			requiredIssue = &issues[i]
		case strings.Contains(issues[i].Title, "Optional system libraries"):
			advisoryIssue = &issues[i]
		}
	}
	if requiredIssue == nil {
		t.Fatalf("no build-dependency issue in %+v", issues)
	}
	if advisoryIssue == nil {
		t.Fatalf("no advisory issue in %+v", issues)
	}
	if requiredIssue.Severity != SeverityWarning {
		t.Errorf("required severity = %q, want warning", requiredIssue.Severity)
	}
	if advisoryIssue.Severity != SeverityInfo {
		t.Errorf("advisory severity = %q, want info", advisoryIssue.Severity)
	}
	if !strings.Contains(requiredIssue.Detail, "libzip") {
		t.Errorf("required detail %q does not name libzip", requiredIssue.Detail)
	}
	if strings.Contains(requiredIssue.Detail, "pcre2") {
		t.Errorf("required detail %q should not name advisory pcre2", requiredIssue.Detail)
	}
	if !strings.Contains(advisoryIssue.Detail, "pcre2") {
		t.Errorf("advisory detail %q does not name pcre2", advisoryIssue.Detail)
	}
}

func TestCheckSystemPackages_AgreesWithInstallDerivation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	installExtensions(t, root, "zip")

	sysSvc := &fakeSystemService{}
	svc := NewService(newFakeRepository(), nil, newTestGraphService())
	svc.sysSvc = sysSvc
	svc.checkSystemPackages()

	// Rebuild the set the way `phpv install` does, from the same definitions.
	var defs []domain.ExtensionDef
	for _, ext := range svc.installedExtensions() {
		if def, ok := svc.graphSvc.GetExtensionDef(ext); ok {
			defs = append(defs, def)
		}
	}
	want := system.Names(system.DependencySet(defs))
	got := system.Names(sysSvc.checked)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("doctor checked %v, install would check %v", got, want)
	}
}
