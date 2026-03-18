package forge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/supanadit/phpv/internal/repository/disk"
)

type mockForgeRepository struct {
	flags    []string
	expanded []string
	ok       bool
}

func (m *mockForgeRepository) PHPVRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".phpv")
}

func (m *mockForgeRepository) VersionsPath() string {
	return filepath.Join(m.PHPVRoot(), "versions")
}

func (m *mockForgeRepository) BuildPrefix(version string) string {
	return filepath.Join(m.VersionsPath(), version)
}

func (m *mockForgeRepository) SourcePath(version string) string {
	return filepath.Join(m.PHPVRoot(), "sources", version, "php")
}

func (m *mockForgeRepository) LedgerPath(version string) string {
	return filepath.Join(m.PHPVRoot(), "ledger", version+".json")
}

func (m *mockForgeRepository) GetConfigureFlags(version string) ([]string, bool) {
	return m.flags, m.ok
}

func (m *mockForgeRepository) ExpandConfigureFlags(version string) ([]string, bool) {
	return m.expanded, m.ok
}

type mockBuildRepository struct{}

func (m *mockBuildRepository) Configure(sourceDir string, flags []string) error {
	return nil
}

func (m *mockBuildRepository) Make(sourceDir string, jobs int) error {
	return nil
}

func (m *mockBuildRepository) Install(sourceDir string) error {
	return nil
}

func TestNewForgeService(t *testing.T) {
	forgeRepo := &mockForgeRepository{}
	buildRepo := &mockBuildRepository{}
	svc := NewService(forgeRepo, buildRepo)

	if svc == nil {
		t.Error("expected service to not be nil")
	}

	if svc.forgeRepository != forgeRepo {
		t.Error("expected forgeRepository to be set")
	}

	if svc.buildRepository != buildRepo {
		t.Error("expected buildRepository to be set")
	}
}

func TestService_GetConfigureFlags(t *testing.T) {
	expectedFlags := []string{"--prefix=/path", "--disable-all"}
	forgeRepo := &mockForgeRepository{
		flags: expectedFlags,
		ok:    true,
	}
	buildRepo := &mockBuildRepository{}

	svc := NewService(forgeRepo, buildRepo)
	flags, ok := svc.GetConfigureFlags("8.2.0")

	if !ok {
		t.Error("expected ok to be true")
	}

	if len(flags) != len(expectedFlags) {
		t.Errorf("expected %d flags, got %d", len(expectedFlags), len(flags))
	}
}

func TestService_GetConfigureFlags_NotFound(t *testing.T) {
	forgeRepo := &mockForgeRepository{
		ok: false,
	}
	buildRepo := &mockBuildRepository{}

	svc := NewService(forgeRepo, buildRepo)
	_, ok := svc.GetConfigureFlags("9.0.0")

	if ok {
		t.Error("expected ok to be false for unknown version")
	}
}

func TestService_ExpandConfigureFlags(t *testing.T) {
	expectedFlags := []string{"--prefix=~/.phpv/versions/8.2.0", "--disable-all"}
	forgeRepo := &mockForgeRepository{
		expanded: expectedFlags,
		ok:       true,
	}
	buildRepo := &mockBuildRepository{}

	svc := NewService(forgeRepo, buildRepo)
	flags, ok := svc.ExpandConfigureFlags("8.2.0")

	if !ok {
		t.Error("expected ok to be true")
	}

	if len(flags) != len(expectedFlags) {
		t.Errorf("expected %d flags, got %d", len(expectedFlags), len(flags))
	}
}

func TestService_GetBuildPrefix(t *testing.T) {
	forgeRepo := disk.NewForgeRepository()
	buildRepo := disk.NewBuildRepository()
	svc := NewService(forgeRepo, buildRepo)

	prefix := svc.GetBuildPrefix("8.2.0")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".phpv", "versions", "8.2.0")

	if prefix != expected {
		t.Errorf("expected %s, got %s", expected, prefix)
	}
}

func TestService_GetSourcePath(t *testing.T) {
	forgeRepo := disk.NewForgeRepository()
	buildRepo := disk.NewBuildRepository()
	svc := NewService(forgeRepo, buildRepo)

	path := svc.GetSourcePath("8.2.0")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".phpv", "sources", "8.2.0", "php")

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
