package terminal

import (
	"fmt"
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/system"
)

// fakeSystemService drives checkSystemDeps in tests. checkResults is consumed
// one call at a time so a test can model "state before install" then "state
// after install"; the last one repeats if more calls happen.
type fakeSystemService struct {
	checkResults []*system.CheckResult
	checkCalls   int
	installErr   error
	installed    []system.Package
	// lastDeps records the dependency set the handler asked about.
	lastDeps []system.Package
}

func (f *fakeSystemService) Check(deps []system.Package) (*system.CheckResult, error) {
	f.lastDeps = deps
	idx := f.checkCalls
	if idx >= len(f.checkResults) {
		idx = len(f.checkResults) - 1
	}
	f.checkCalls++
	return f.checkResults[idx], nil
}

func (f *fakeSystemService) CheckBuildTools(toolNames []string) (*system.CheckResult, error) {
	return &system.CheckResult{Distro: system.Distro{PM: "dnf"}}, nil
}

func (f *fakeSystemService) Install(packages []system.Package) error {
	if f.installErr != nil {
		return f.installErr
	}
	f.installed = append(f.installed, packages...)
	return nil
}

func (f *fakeSystemService) InstallCommand(packages []system.Package) string {
	return "dnf install -y"
}

// fakeAssemblerService supplies extension definitions to checkSystemDeps.
type fakeAssemblerService struct {
	AssemblerService
	defs map[string]domain.ExtensionDef
}

func (f *fakeAssemblerService) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	def, ok := f.defs[name]
	return def, ok
}

func missingResult(names ...string) *system.CheckResult {
	r := &system.CheckResult{Distro: system.Distro{PM: "dnf"}}
	for _, n := range names {
		r.Missing = append(r.Missing, system.Package{Name: n, SystemName: n + "-devel"})
	}
	return r
}

func availableResult(pkgs ...system.Package) *system.CheckResult {
	return &system.CheckResult{Distro: system.Distro{PM: "dnf"}, Available: pkgs}
}

func TestCheckSystemDeps_InstalledPackagesCarryVersion(t *testing.T) {
	sysSvc := &fakeSystemService{
		checkResults: []*system.CheckResult{
			missingResult("curl"),
			availableResult(system.Package{Name: "curl", SystemName: "libcurl-devel", Version: "8.18.0", Installed: true}),
		},
	}
	h := &PHPHandler{systemSvc: sysSvc, assemblerSvc: &fakeAssemblerService{}}

	got, err := h.checkSystemDeps(nil, true, false) // autoDeps: installs without prompting
	if err != nil {
		t.Fatalf("checkSystemDeps: %v", err)
	}
	p, ok := got["curl"]
	if !ok {
		t.Fatal("curl missing from returned map after successful install")
	}
	if !p.Installed || p.Version == "" {
		t.Fatalf("curl = {%v, %q}, want Installed=true and a non-empty Version", p.Installed, p.Version)
	}
	if sysSvc.checkCalls != 2 {
		t.Fatalf("Check called %d times, want 2 (before and after install)", sysSvc.checkCalls)
	}
}

func TestCheckSystemDeps_NoVersionMeansUnusable(t *testing.T) {
	sysSvc := &fakeSystemService{
		checkResults: []*system.CheckResult{
			missingResult("curl"),
			// Installed, but the package manager yields no comparable version.
			availableResult(system.Package{Name: "curl", SystemName: "libcurl-devel", Installed: true}),
		},
	}
	h := &PHPHandler{systemSvc: sysSvc, assemblerSvc: &fakeAssemblerService{}}

	got, err := h.checkSystemDeps(nil, true, false)
	if err != nil {
		t.Fatalf("checkSystemDeps: %v", err)
	}
	if p, ok := got["curl"]; ok && p.Version == "" {
		t.Fatalf("curl returned as available with empty Version; the assembler gate would reject it and rebuild from source anyway")
	}
}

func TestCheckSystemDeps_DeclinedOrFailedInstall(t *testing.T) {
	tests := []struct {
		name     string
		dryRun   bool
		instErr  error
		wantErr  bool
		wantCurl bool
	}{
		{name: "dry run does not install", dryRun: true, wantCurl: false},
		{name: "install failure is reported", instErr: fmt.Errorf("dnf exploded"), wantErr: true, wantCurl: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sysSvc := &fakeSystemService{
				checkResults: []*system.CheckResult{missingResult("curl")},
				installErr:   tt.instErr,
			}
			h := &PHPHandler{systemSvc: sysSvc, assemblerSvc: &fakeAssemblerService{}}

			// autoDeps=false + dryRun=false needs a real terminal on stdin, so
			// use autoDeps for the failure case and dryRun for the skip case.
			got, err := h.checkSystemDeps(nil, tt.instErr != nil, tt.dryRun)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error when the install command fails")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := got["curl"]; ok != tt.wantCurl {
				t.Fatalf("curl present = %v, want %v (map must reflect what Check reports)", ok, tt.wantCurl)
			}
		})
	}
}
