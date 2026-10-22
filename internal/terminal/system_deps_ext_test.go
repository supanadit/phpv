package terminal

import (
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/system"
)

// extension add shares checkSystemDeps with install, so the dependency-set
// derivation must hold there too: the deps handed to the system check come from
// the extension definitions, and are marked Required when they feed the build.
func TestCheckSystemDeps_ExtensionAddPath(t *testing.T) {
	sysSvc := &fakeSystemService{
		checkResults: []*system.CheckResult{
			missingResult("libzip"),
			availableResult(system.Package{Name: "libzip", SystemName: "libzip-devel", Version: "1.11.4", Installed: true}),
		},
	}
	asm := &fakeAssemblerService{defs: map[string]domain.ExtensionDef{
		"zip": {Name: "zip", RequiresPackage: "libzip", RequiresPackages: []string{"libzip"}},
	}}
	h := &PHPHandler{systemSvc: sysSvc, assemblerSvc: asm}

	got, err := h.checkSystemDeps([]string{"zip"}, true, false)
	if err != nil {
		t.Fatalf("checkSystemDeps: %v", err)
	}

	var sawLibzip bool
	for _, dep := range sysSvc.lastDeps {
		if dep.Name == "libzip" {
			sawLibzip = true
			if !dep.Required {
				t.Error("libzip not marked Required: zip declares it via RequiresPackage")
			}
		}
	}
	if !sawLibzip {
		t.Fatalf("libzip absent from the checked dependency set %v", system.Names(sysSvc.lastDeps))
	}

	p, ok := got["libzip"]
	if !ok || p.Version == "" {
		t.Fatalf("libzip = %+v, want installed with a version after the refresh", p)
	}
}

func TestExtensionDefsSkipsUnknownExtensions(t *testing.T) {
	asm := &fakeAssemblerService{defs: map[string]domain.ExtensionDef{
		"zip": {Name: "zip", RequiresPackage: "libzip"},
	}}
	h := &PHPHandler{assemblerSvc: asm}

	defs := h.extensionDefs([]string{"zip", "not-a-real-extension"})
	if len(defs) != 1 {
		t.Fatalf("extensionDefs returned %d defs, want 1 (unknown extensions skipped)", len(defs))
	}
	if defs[0].Name != "zip" {
		t.Fatalf("defs[0].Name = %q, want zip", defs[0].Name)
	}
}
