package args

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	vsn "github.com/supanadit/phpv/pkg/version"
)

// testSiloDeps implements SiloService against a temporary PHPV_ROOT.
type testSiloDeps struct{ root string }

func (s *testSiloDeps) Download(name, version string) (bool, error) { return false, nil }
func (s *testSiloDeps) DownloadURL(url, checksumType, checksumValue string) (bool, error) {
	return false, nil
}
func (s *testSiloDeps) Extract(archivePath, destDir string) (bool, error) { return false, nil }
func (s *testSiloDeps) GetSilo() domain.Silo                              { return domain.Silo{Root: s.root} }
func (s *testSiloDeps) GetState(name, version string) (domain.InstallState, error) {
	return domain.StateNone, nil
}
func (s *testSiloDeps) GetDefault() (string, error) {
	data, err := os.ReadFile(filepath.Join(s.root, "default"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
func (s *testSiloDeps) SetDefault(version string) error {
	return os.WriteFile(filepath.Join(s.root, "default"), []byte(version+"\n"), 0o644)
}
func (s *testSiloDeps) PackagePrefix(name, version string) string {
	return filepath.Join(s.root, "packages", name, version)
}
func (s *testSiloDeps) SourcePath(pkg, version string) string {
	return filepath.Join(s.root, "sources", pkg, version)
}
func (s *testSiloDeps) PHPOutputPath(phpVersion string) string {
	return filepath.Join(s.root, "packages", "php", phpVersion)
}
func (s *testSiloDeps) GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error) {
	path := filepath.Join(s.root, "packages", "php", phpVersion, "extensions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &domain.ExtensionManifest{}, nil
	}
	var m domain.ExtensionManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
func (s *testSiloDeps) SaveExtensionManifest(phpVersion string, m *domain.ExtensionManifest) error {
	return nil
}

// testConfigDeps implements ConfigService with an in-memory map.
type testConfigDeps struct {
	ConfigService
	values map[string]string
}

func (c *testConfigDeps) Get(key string) (string, error) { return c.values[key], nil }
func (c *testConfigDeps) Set(key, value string) error {
	if c.values == nil {
		c.values = make(map[string]string)
	}
	c.values[key] = value
	return nil
}
func (c *testConfigDeps) List() ([]string, error) { return nil, nil }

// The remaining dependencies are unused by the handler tests that share the
// test handler; embedding the interface keeps the fakes minimal.
type testAssemblerDeps struct{ root string }

func (a *testAssemblerDeps) ResolveVersion(name, constraint string) (string, error) {
	root := a.root
	if root == "" {
		root = os.Getenv("PHPV_ROOT")
	}
	entries, err := os.ReadDir(filepath.Join(root, "packages", name))
	if err != nil {
		return "", fmt.Errorf("no version matching %q found", constraint)
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	if v := vsn.LatestMatching(versions, constraint+"."); v != "" {
		return v, nil
	}
	for _, v := range versions {
		if v == constraint {
			return v, nil
		}
	}
	return "", fmt.Errorf("no version matching %q found", constraint)
}
func (a *testAssemblerDeps) Assemble(ctx context.Context, spec domain.AssembleSpec) (*domain.AssemblerResult, error) {
	return nil, fmt.Errorf("not available in tests")
}
func (a *testAssemblerDeps) InstallExtension(ctx context.Context, req domain.ExtensionInstallRequest) error {
	return nil
}
func (a *testAssemblerDeps) RemoveExtension(phpVersion, extName, phpPrefix string) error {
	return nil
}
func (a *testAssemblerDeps) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	return nil
}
func (a *testAssemblerDeps) DefaultExtensions(phpVersion string) ([]string, []string) {
	return nil, nil
}
func (a *testAssemblerDeps) ExpandImplied(extensions []string) ([]string, []string) {
	return extensions, nil
}
func (a *testAssemblerDeps) SharedOnlyExtensions(phpVersion string, requested []string) []string {
	return nil
}
func (a *testAssemblerDeps) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return domain.ExtensionDef{}, false
}

type testRegistryDeps struct{ RegistryService }
type testBundleDeps struct{ BundleService }
type testSystemDeps struct{}

func (s *testSystemDeps) Check(deps []domain.SystemPackage) (*domain.SystemCheckResult, error) {
	return &domain.SystemCheckResult{}, nil
}
func (s *testSystemDeps) CheckBuildTools(names []string) (*domain.SystemCheckResult, error) {
	return &domain.SystemCheckResult{}, nil
}
func (s *testSystemDeps) Install(packages []domain.SystemPackage) error         { return nil }
func (s *testSystemDeps) InstallCommand(packages []domain.SystemPackage) string { return "" }
func (s *testSystemDeps) DependencySet(exts []domain.ExtensionDef) []domain.SystemPackage {
	return nil
}

type testShimDeps struct{}

func (s *testShimDeps) RegenerateAll() error                     { return nil }
func (s *testShimDeps) WritePhar(name, pharRelPath string) error { return nil }
func (s *testShimDeps) SetSystemMode(enabled bool) error         { return nil }

type testPECLDeps struct{}

func (p *testPECLDeps) Install(ctx context.Context, source, phpVersion string, jobs int) (*domain.PECLInstallResult, error) {
	return nil, nil
}
func (p *testPECLDeps) InstallCompatible(ctx context.Context, req domain.PECLInstallRequest) (*domain.PECLInstallResult, error) {
	return nil, nil
}
func (p *testPECLDeps) ResolveCompatibleRelease(name, phpVersion string) (string, error) {
	return "1.2.3", nil
}
func (p *testPECLDeps) List(phpVersion string) ([]domain.ExtensionState, error) { return nil, nil }
func (p *testPECLDeps) Uninstall(name, phpVersion string) error                 { return nil }

type testDoctorDeps struct{ DoctorService }
type testUpdateDeps struct{ UpdateService }
type testPatchSyncDeps struct{ PatchSync }
type testPatchListerDeps struct{ PatchLister }
type testFPMDeps struct{ FPMService }
type testApacheDeps struct{ ApacheService }
type testServerAsmDeps struct{ ServerAssembler }

func newTestHandlerDeps(root string) *PHPHandler {
	return &PHPHandler{
		ctx:          context.Background(),
		version:      "dev",
		siloSvc:      &testSiloDeps{root: root},
		configSvc:    &testConfigDeps{values: make(map[string]string)},
		assemblerSvc: &testAssemblerDeps{root: root},
		registrySvc:  &testRegistryDeps{},
		bundleSvc:    &testBundleDeps{},
		systemSvc:    &testSystemDeps{},
		shimSvc:      &testShimDeps{},
		peclSvc:      &testPECLDeps{},
		doctorSvc:    &testDoctorDeps{},
		updateSvc:    &testUpdateDeps{},
		patchSync:    &testPatchSyncDeps{},
		patchLister:  &testPatchListerDeps{},
		fpmSvc:       &testFPMDeps{},
		apacheSvc:    &testApacheDeps{},
		serverAsbl:   &testServerAsmDeps{},
	}
}
