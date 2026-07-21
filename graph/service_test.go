package graph

import (
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockGraphRepo struct {
	deps []domain.Dependency
	err  error
	// extensionDeps maps extName -> (pkgName, versionWithConstraint, ok)
	extensionDeps map[string]struct {
		pkgName string
		version string
		ok      bool
	}
	// transitiveDeps maps pkgName -> deps (for GetOrderedDependencies on non-php packages)
	transitiveDeps map[string][]domain.Dependency
}

func (m *mockGraphRepo) GetOrderedDependencies(name, version string) ([]domain.Dependency, error) {
	if name == "php" {
		return m.deps, m.err
	}
	if m.transitiveDeps != nil {
		if deps, ok := m.transitiveDeps[name]; ok {
			return deps, nil
		}
	}
	return nil, nil
}
func (m *mockGraphRepo) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return domain.ExtensionDef{}, false
}
func (m *mockGraphRepo) IsExtensionValidForPHPVersion(name, phpVersion string) bool {
	return true
}
func (m *mockGraphRepo) GetConflictingExtensions(name string) []string {
	return nil
}
func (m *mockGraphRepo) GetExtensionDependency(name string) (string, bool) {
	return "", false
}
func (m *mockGraphRepo) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	if m.extensionDeps != nil {
		if ed, ok := m.extensionDeps[extName]; ok {
			return ed.pkgName, ed.version, ed.ok
		}
	}
	return "", "", false
}
func (m *mockGraphRepo) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	return nil, nil
}
func (m *mockGraphRepo) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	return nil, nil
}
func (m *mockGraphRepo) ListExtensions() []domain.ExtensionInfo {
	return nil
}
func (m *mockGraphRepo) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	return nil
}
func (m *mockGraphRepo) ExpandImplied(extensions []string) ([]string, []string) {
	return extensions, nil
}
func (m *mockGraphRepo) DefaultExtensions(phpVersion string) ([]string, []string) {
	return nil, nil
}
func (m *mockGraphRepo) SharedOnlyExtensions(phpVersion string, requested []string) []string {
	return nil
}
func (m *mockGraphRepo) GetConfigureFlags(name, version string) []string {
	return nil
}
func (m *mockGraphRepo) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return nil
}
func (m *mockGraphRepo) GetExtensionConfigureFlags(name string, phpVersion string) []string {
	return nil
}
func (m *mockGraphRepo) GetCompilerStdRule(phpVersion string) domain.CompilerRule {
	return domain.CompilerRule{CStd: "-std=gnu11", CXXStd: "-std=gnu++17"}
}
func (m *mockGraphRepo) GetCompilerFlags(compiler, phpVersion string) []string {
	return nil
}

func TestService_GetBuildPlan(t *testing.T) {
	mock := &mockGraphRepo{
		deps: []domain.Dependency{
			{Name: "openssl", Version: "1.1.1w"},
		},
	}
	svc := NewService(mock)

	plan, err := svc.GetBuildPlan("php", "8.4.0", nil)
	if err != nil {
		t.Fatalf("GetBuildPlan returned error: %v", err)
	}
	if len(plan.Deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(plan.Deps))
	}
	if len(plan.CompilerFlags) != 1 {
		t.Fatalf("expected 1 compiler flag, got %d", len(plan.CompilerFlags))
	}
	if len(plan.CXXCompilerFlags) != 1 {
		t.Fatalf("expected 1 CXX compiler flag, got %d", len(plan.CXXCompilerFlags))
	}
}

func TestService_GetBuildPlan_WithExtensions(t *testing.T) {
	mock := &mockGraphRepo{
		deps: []domain.Dependency{},
		extensionDeps: map[string]struct {
			pkgName string
			version string
			ok      bool
		}{
			"openssl": {"openssl", "1.1.1w|>=1.0.2,<4.0.0", true},
			"curl":    {"curl", "8.10.1|>=8.0.0", true},
		},
		transitiveDeps: map[string][]domain.Dependency{
			"openssl": {
				{Name: "perl", Version: "5.38.2|>=5.32.0"},
				{Name: "m4", Version: "1.4.19"},
			},
			"curl": {
				{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
			},
		},
	}
	svc := NewService(mock)

	plan, err := svc.GetBuildPlan("php", "8.4.0", []string{"openssl", "curl"})
	if err != nil {
		t.Fatalf("GetBuildPlan returned error: %v", err)
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
	if v, ok := depMap["perl"]; !ok || v != "5.38.2|>=5.32.0" {
		t.Errorf("perl version = %q, want %q", v, "5.38.2|>=5.32.0")
	}
	if v, ok := depMap["m4"]; !ok || v != "1.4.19" {
		t.Errorf("m4 version = %q, want %q", v, "1.4.19")
	}
	if v, ok := depMap["zlib"]; !ok || v != "1.2.13|>=1.2.0,<1.3.0" {
		t.Errorf("zlib version = %q, want %q", v, "1.2.13|>=1.2.0,<1.3.0")
	}
}

func TestService_GetBuildPlan_Minimal_NoDeps(t *testing.T) {
	mock := &mockGraphRepo{
		deps: []domain.Dependency{},
	}
	svc := NewService(mock)

	plan, err := svc.GetBuildPlan("php", "8.4.0", nil)
	if err != nil {
		t.Fatalf("GetBuildPlan returned error: %v", err)
	}
	if len(plan.Deps) != 0 {
		t.Errorf("expected 0 deps for --minimal, got %d: %v", len(plan.Deps), plan.Deps)
	}
}

func TestService_GetBuildPlan_DedupConflictingVersions(t *testing.T) {
	mock := &mockGraphRepo{
		deps: []domain.Dependency{},
		extensionDeps: map[string]struct {
			pkgName string
			version string
			ok      bool
		}{
			"ext1": {"openssl", "1.1.1w|>=1.0.2,<4.0.0", true},
			"ext2": {"openssl", "1.0.1u|>=0.9.8,<1.2.0", true},
		},
	}
	svc := NewService(mock)

	plan, err := svc.GetBuildPlan("php", "8.4.0", []string{"ext1", "ext2"})
	if err != nil {
		t.Fatalf("GetBuildPlan returned error: %v", err)
	}

	depMap := make(map[string]string)
	for _, dep := range plan.Deps {
		depMap[dep.Name] = dep.Version
	}

	if v, ok := depMap["openssl"]; !ok || v != "1.1.1w|>=1.0.2,<4.0.0" {
		t.Errorf("expected higher version 1.1.1w to win, got %q", v)
	}
}

func TestService_GetOrderedDependencies(t *testing.T) {
	mock := &mockGraphRepo{
		deps: []domain.Dependency{
			{Name: "openssl", Version: "1.1.1w"},
			{Name: "zlib", Version: "1.2.13"},
		},
	}
	svc := NewService(mock)

	deps, err := svc.GetOrderedDependencies("php", "8.4.0")
	if err != nil {
		t.Fatalf("GetOrderedDependencies returned error: %v", err)
	}
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}
}
