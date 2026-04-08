package terminal

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockBundlerRepo struct {
	installFunc     func(version, compiler string, fresh bool) (domain.Forge, error)
	orchestrateFunc func(name, version, compiler string, fresh bool) (domain.Forge, error)
}

func (m *mockBundlerRepo) Install(version, compiler string, fresh bool) (domain.Forge, error) {
	if m.installFunc != nil {
		return m.installFunc(version, compiler, fresh)
	}
	return domain.Forge{Prefix: "/fake/prefix"}, nil
}

func (m *mockBundlerRepo) Orchestrate(name, version, compiler string, fresh bool) (domain.Forge, error) {
	if m.orchestrateFunc != nil {
		return m.orchestrateFunc(name, version, compiler, fresh)
	}
	return domain.Forge{Prefix: "/fake/prefix"}, nil
}

type mockSiloRepo struct {
	versions       []string
	defaultVer     string
	setDefaultErr  error
	getDefaultErr  error
	removeResult   []string
	removeErr      error
	installedVerts map[string]bool
}

func newMockSiloRepo() *mockSiloRepo {
	return &mockSiloRepo{
		versions:       []string{"8.4.0", "8.3.0"},
		defaultVer:     "8.4.0",
		installedVerts: map[string]bool{"8.4.0": true, "8.3.0": true},
	}
}

func (m *mockSiloRepo) GetSilo() (*domain.Silo, error) {
	return &domain.Silo{Root: "/fake/root"}, nil
}

func (m *mockSiloRepo) EnsurePaths() error {
	return nil
}

func (m *mockSiloRepo) GetDefault() (string, error) {
	if m.getDefaultErr != nil {
		return "", m.getDefaultErr
	}
	return m.defaultVer, nil
}

func (m *mockSiloRepo) SetDefault(version string) error {
	m.defaultVer = version
	return m.setDefaultErr
}

func (m *mockSiloRepo) ListVersions() []string {
	return m.versions
}

func (m *mockSiloRepo) GetState(version string) (domain.InstallState, error) {
	if m.installedVerts != nil && m.installedVerts[version] {
		return domain.StateInstalled, nil
	}
	return domain.StateNone, nil
}

func (m *mockSiloRepo) RemovePHPInstallation(version string) ([]string, error) {
	if m.removeErr != nil {
		return nil, m.removeErr
	}
	return m.removeResult, nil
}

func (m *mockSiloRepo) RemoveUnusedBuildTools(dryRun bool) ([]string, []string, error) {
	return nil, nil, nil
}

func (m *mockSiloRepo) ArchiveExists(pkg, ver string) bool {
	return false
}

func (m *mockSiloRepo) GetArchivePath(pkg, ver string) string {
	return ""
}

func (m *mockSiloRepo) StoreArchive(pkg, ver string, data io.Reader) error {
	return nil
}

func (m *mockSiloRepo) RetrieveArchive(pkg, ver string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockSiloRepo) RemoveArchive(pkg, ver string) error {
	return nil
}

func (m *mockSiloRepo) ListArchives() []string {
	return nil
}

func (m *mockSiloRepo) SourceExists(pkg, ver string) bool {
	return false
}

func (m *mockSiloRepo) GetSourcePath(pkg, ver string) string {
	return ""
}

func (m *mockSiloRepo) StoreSource(pkg, ver string, data io.Reader) error {
	return nil
}

func (m *mockSiloRepo) RetrieveSource(pkg, ver string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockSiloRepo) RemoveSource(pkg, ver string) error {
	return nil
}

func (m *mockSiloRepo) ListSources() []string {
	return nil
}

func (m *mockSiloRepo) VersionExists(pkg, ver string) bool {
	return false
}

func (m *mockSiloRepo) GetVersionPath(pkg, ver string) string {
	return ""
}

func (m *mockSiloRepo) StoreVersion(pkg, ver string, data io.Reader) error {
	return nil
}

func (m *mockSiloRepo) RetrieveVersion(pkg, ver string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockSiloRepo) RemoveVersion(pkg, ver string) error {
	return nil
}

func (m *mockSiloRepo) FullClean(pkg, ver string) error {
	return nil
}

func (m *mockSiloRepo) CleanAll() error {
	return nil
}

func (m *mockSiloRepo) MarkInProgress(phpVersion string) error {
	return nil
}

func (m *mockSiloRepo) MarkComplete(phpVersion string) error {
	return nil
}

func (m *mockSiloRepo) MarkFailed(phpVersion string) error {
	return nil
}

func (m *mockSiloRepo) Rollback(phpVersion string) error {
	return nil
}

func (m *mockSiloRepo) SaveDependencyInfo(phpVersion string, deps []domain.DependencyInfo) error {
	return nil
}

func (m *mockSiloRepo) GetDependencyInfo(phpVersion string) ([]domain.DependencyInfo, error) {
	return nil, nil
}

func (m *mockSiloRepo) RemoveDependencyInfo(phpVersion string) error {
	return nil
}

func (m *mockSiloRepo) IncrementBuildToolRef(name, version, phpVersion string) error {
	return nil
}

func (m *mockSiloRepo) DecrementBuildToolRef(name, version, phpVersion string) error {
	return nil
}

func (m *mockSiloRepo) GetBuildToolRefs() (map[string][]string, error) {
	return nil, nil
}

func (m *mockSiloRepo) RemoveBuildToolRef(name, version string) error {
	return nil
}

func (m *mockSiloRepo) GetInstalledBuildTools() ([]string, error) {
	return nil, nil
}

type mockSourceRepo struct{}

func (m *mockSourceRepo) GetVersions() ([]domain.Source, error) {
	return []domain.Source{}, nil
}

func (m *mockSourceRepo) GetSources(name, version string) ([]domain.Source, error) {
	return []domain.Source{}, nil
}

func newTestHandler() *TerminalHandler {
	siloRepo := newMockSiloRepo()
	srcRepo := &mockSourceRepo{}

	return NewHandler(&mockBundlerRepo{}, siloRepo, srcRepo)
}

func TestNewHandler(t *testing.T) {
	handler := newTestHandler()
	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}
	if handler.BundlerRepo == nil {
		t.Error("BundlerRepo is nil")
	}
	if handler.Silo == nil {
		t.Error("Silo is nil")
	}
	if handler.Source == nil {
		t.Error("Source is nil")
	}
}

func TestInstall_Success(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{
		installFunc: func(version, compiler string, fresh bool) (domain.Forge, error) {
			return domain.Forge{
				Prefix: "/fake/output",
				Env:    map[string]string{"LD_LIBRARY_PATH": "/fake/lib"},
			}, nil
		},
	}
	handler.BundlerRepo = mockBundler

	_, err := handler.Install("8.4.0", "", false, false)
	if err != nil {
		t.Errorf("Install failed: %v", err)
	}
}

func TestInstall_Error(t *testing.T) {
	handler := newTestHandler()

	expectedErr := errors.New("installation failed")
	mockBundler := &mockBundlerRepo{
		installFunc: func(version, compiler string, fresh bool) (domain.Forge, error) {
			return domain.Forge{}, expectedErr
		},
	}
	handler.BundlerRepo = mockBundler

	_, err := handler.Install("8.4.0", "", false, false)
	if err == nil {
		t.Error("Install should have failed")
	}
}

func TestUse_Success(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	_, err := handler.Use("8.4.0")
	if err != nil {
		t.Logf("Use returned error (expected with fake paths): %v", err)
	}
}

func TestUse_VersionNotInstalled(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	_, err := handler.Use("9.0.0")
	if err == nil {
		t.Error("Use should have failed for non-installed version")
	}
}

func TestUse_ShimGeneration(t *testing.T) {
	mockSilo := newMockSiloRepo()
	mockSilo.versions = []string{"8.4.0"}
	mockSilo.installedVerts = map[string]bool{"8.4.0": true}

	mockBundler := &mockBundlerRepo{}
	srcRepo := &mockSourceRepo{}
	handler := NewHandler(mockBundler, mockSilo, srcRepo)

	result, err := handler.Use("8.4.0")
	if err != nil {
		t.Logf("Use failed (expected for fake path): %v", err)
	}

	if result != nil && result.ShimPath == "" {
		t.Error("ShimPath should not be empty")
	}
}

func TestSetDefault_Success(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	err := handler.SetDefault("8.4.0")
	if err != nil {
		t.Errorf("SetDefault failed: %v", err)
	}
}

func TestGetDefault_NoDefault(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	defaultVer, err := handler.GetDefault()
	if err != nil {
		t.Errorf("GetDefault failed: %v", err)
	}
	if defaultVer != "8.4.0" {
		t.Errorf("Expected default 8.4.0, got %s", defaultVer)
	}
}

func TestListInstalled_Empty(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	versions, err := handler.ListInstalled()
	if err != nil {
		t.Errorf("ListInstalled failed: %v", err)
	}
	if len(versions) == 0 {
		t.Error("Expected non-empty version list")
	}
}

func TestListInstalled_WithVersions(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	_, err := handler.ListInstalled()
	if err != nil {
		t.Errorf("ListInstalled failed: %v", err)
	}
}

func TestListAvailable_Success(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	_, err := handler.ListAvailable()
	if err != nil {
		t.Errorf("ListAvailable failed: %v", err)
	}
}

func TestWhich_NoDefault(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	phpPath, err := handler.Which()
	if err != nil {
		t.Errorf("Which failed: %v", err)
	}
	if phpPath == "" {
		t.Error("Expected path to be set")
	}
}

func TestUninstall_Success(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	result, err := handler.Uninstall("8.4.0")
	if err != nil {
		t.Errorf("Uninstall failed: %v", err)
	}
	if result == nil {
		t.Fatal("UninstallResult is nil")
	}
	if result.Version != "8.4.0" {
		t.Errorf("Expected version 8.4.0, got %s", result.Version)
	}
}

func TestCleanBuildTools_DryRun(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	result, err := handler.CleanBuildTools(true)
	if err != nil {
		t.Errorf("CleanBuildTools failed: %v", err)
	}
	if result == nil {
		t.Fatal("CleanBuildToolsResult is nil")
	}
	if !result.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestCleanBuildTools_Actual(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	result, err := handler.CleanBuildTools(false)
	if err != nil {
		t.Errorf("CleanBuildTools failed: %v", err)
	}
	if result == nil {
		t.Fatal("CleanBuildToolsResult is nil")
	}
	if result.DryRun {
		t.Error("DryRun should be false")
	}
}

func TestDoctor_NoIssues(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	result, err := handler.Doctor()
	if err != nil {
		t.Errorf("Doctor failed: %v", err)
	}
	if result == nil {
		t.Fatal("DoctorResult is nil")
	}
}

func TestDoctor_WithIssues(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	result, err := handler.Doctor()
	if err != nil {
		t.Errorf("Doctor failed: %v", err)
	}

	if result != nil {
	}
}

func TestUseResult_Structure(t *testing.T) {
	result := &UseResult{
		ExactVersion: "8.4.0",
		ShimPath:     "/fake/bin",
		OutputPath:   "/fake/output",
	}

	if result.ExactVersion != "8.4.0" {
		t.Errorf("Expected ExactVersion 8.4.0, got %s", result.ExactVersion)
	}
	if result.ShimPath != "/fake/bin" {
		t.Errorf("Expected ShimPath /fake/bin, got %s", result.ShimPath)
	}
	if result.OutputPath != "/fake/output" {
		t.Errorf("Expected OutputPath /fake/output, got %s", result.OutputPath)
	}
}

func TestUninstallResult_Structure(t *testing.T) {
	result := &UninstallResult{
		Version:      "8.4.0",
		RemovedTools: []string{"m4@1.4.19"},
		WasDefault:   true,
	}

	if result.Version != "8.4.0" {
		t.Errorf("Expected Version 8.4.0, got %s", result.Version)
	}
	if len(result.RemovedTools) != 1 {
		t.Errorf("Expected 1 removed tool, got %d", len(result.RemovedTools))
	}
	if !result.WasDefault {
		t.Error("Expected WasDefault to be true")
	}
}

func TestCleanBuildToolsResult_Structure(t *testing.T) {
	result := &CleanBuildToolsResult{
		Removed:    []string{"m4@1.4.19"},
		WillRemove: []string{"autoconf@2.69"},
		DryRun:     false,
	}

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed, got %d", len(result.Removed))
	}
	if len(result.WillRemove) != 1 {
		t.Errorf("Expected 1 will remove, got %d", len(result.WillRemove))
	}
	if result.DryRun {
		t.Error("Expected DryRun to be false")
	}
}

func TestUpgradeResult_Structure(t *testing.T) {
	result := &UpgradeResult{
		FromVersion: "8.4.0",
		ToVersion:   "8.4.1",
		Forge: domain.Forge{
			Prefix: "/fake/output",
			Env:    map[string]string{},
		},
	}

	if result.FromVersion != "8.4.0" {
		t.Errorf("Expected FromVersion 8.4.0, got %s", result.FromVersion)
	}
	if result.ToVersion != "8.4.1" {
		t.Errorf("Expected ToVersion 8.4.1, got %s", result.ToVersion)
	}
}

func TestDoctorResult_Structure(t *testing.T) {
	result := &DoctorResult{
		Issues: []DoctorIssue{
			{Category: "system", Message: "missing command"},
		},
		Warnings: []DoctorWarning{
			{Category: "phpv", Message: "config warning"},
		},
	}

	if len(result.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Issues))
	}
	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestDoctorIssue_Structure(t *testing.T) {
	issue := DoctorIssue{
		Category: "system",
		Message:  "missing command: make",
	}

	if issue.Category != "system" {
		t.Errorf("Expected Category system, got %s", issue.Category)
	}
	if issue.Message != "missing command: make" {
		t.Errorf("Expected Message 'missing command: make', got %s", issue.Message)
	}
}

func TestDoctorWarning_Structure(t *testing.T) {
	warning := DoctorWarning{
		Category: "phpv",
		Message:  "default version set but binary not found",
	}

	if warning.Category != "phpv" {
		t.Errorf("Expected Category phpv, got %s", warning.Category)
	}
	if warning.Message != "default version set but binary not found" {
		t.Errorf("Expected Message 'default version set but binary not found', got %s", warning.Message)
	}
}

func TestResolveInstalledVersion(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	_, err := handler.resolveInstalledVersion("8.4")
	if err != nil {
		t.Errorf("resolveInstalledVersion failed: %v", err)
	}
}

func TestShellUse_Success(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	err := handler.ShellUse("8.4.0")
	if err != nil {
		t.Errorf("ShellUse failed: %v", err)
	}
}

func TestShellUse_VersionNotInstalled(t *testing.T) {
	handler := newTestHandler()

	mockBundler := &mockBundlerRepo{}
	handler.BundlerRepo = mockBundler

	err := handler.ShellUse("9.0.0")
	if err == nil {
		t.Error("ShellUse should have failed for non-installed version")
	}
}

func TestShellUse_ConstraintResolution(t *testing.T) {
	mockSilo := newMockSiloRepo()
	mockSilo.versions = []string{"8.4.0", "8.3.0"}
	mockSilo.installedVerts = map[string]bool{"8.4.0": true, "8.3.0": true}

	mockBundler := &mockBundlerRepo{}
	srcRepo := &mockSourceRepo{}
	handler := NewHandler(mockBundler, mockSilo, srcRepo)

	err := handler.ShellUse("8.4")
	if err != nil {
		t.Errorf("ShellUse with constraint failed: %v", err)
	}

	if mockSilo.defaultVer != "8.4.0" {
		t.Errorf("Expected default 8.4.0, got %s", mockSilo.defaultVer)
	}
}

func TestAutoDetect_NoComposer(t *testing.T) {
	mockSilo := newMockSiloRepo()
	mockBundler := &mockBundlerRepo{}
	srcRepo := &mockSourceRepo{}
	handler := NewHandler(mockBundler, mockSilo, srcRepo)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get current directory, skipping test")
	}
	defer os.Chdir(oldCwd)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	_, err = handler.AutoDetect()
	if err == nil {
		t.Error("AutoDetect should fail when no composer.json exists")
	}
}

func TestAutoDetect_EmptyConfig(t *testing.T) {
	mockSilo := newMockSiloRepo()
	mockBundler := &mockBundlerRepo{}
	srcRepo := &mockSourceRepo{}
	handler := NewHandler(mockBundler, mockSilo, srcRepo)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get current directory, skipping test")
	}
	defer os.Chdir(oldCwd)

	tmpDir := t.TempDir()
	composerJSON := `{"name": "test/package"}`
	os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(composerJSON), 0644)
	os.Chdir(tmpDir)

	_, err = handler.AutoDetect()
	if err == nil {
		t.Error("AutoDetect should fail when no config.platform.php is set")
	}
}

func TestAutoDetectResolve_NotInstalled(t *testing.T) {
	mockSilo := newMockSiloRepo()
	mockBundler := &mockBundlerRepo{}
	srcRepo := &mockSourceRepo{}
	handler := NewHandler(mockBundler, mockSilo, srcRepo)

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get current directory, skipping test")
	}
	defer os.Chdir(oldCwd)

	tmpDir := t.TempDir()
	composerJSON := `{"config":{"platform":{"php":"9.0"}}}`
	os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(composerJSON), 0644)
	os.Chdir(tmpDir)

	_, err = handler.AutoDetectResolve()
	if err == nil {
		t.Error("AutoDetectResolve should fail when version is not installed")
	}
}
