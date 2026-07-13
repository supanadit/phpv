package graph

import (
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockGraphRepo struct {
	deps []domain.Dependency
	err  error
}

func (m *mockGraphRepo) GetOrderedDependencies(name, version string) ([]domain.Dependency, error) {
	return m.deps, m.err
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
func (m *mockGraphRepo) GetConfigureFlags(name, version string) []string {
	return nil
}
func (m *mockGraphRepo) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
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
	if len(plan.CompilerFlags) != 2 {
		t.Fatalf("expected 2 compiler flags, got %d", len(plan.CompilerFlags))
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
