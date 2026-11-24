package args

import (
	"context"

	"github.com/supanadit/phpv/domain"
)

// AssemblerService is the set of assembly operations the terminal handler needs.
type AssemblerService interface {
	ResolveVersion(name, constraint string) (string, error)
	Assemble(ctx context.Context, spec domain.AssembleSpec) (*domain.AssemblerResult, error)
	InstallExtension(ctx context.Context, req domain.ExtensionInstallRequest) error
	RemoveExtension(phpVersion, extName, phpPrefix string) error
	ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo
	DefaultExtensions(phpVersion string) (included []string, skipped []string)
	ExpandImplied(extensions []string) (expanded []string, added []string)
	SharedOnlyExtensions(phpVersion string, requested []string) []string
	GetExtensionDef(name string) (domain.ExtensionDef, bool)
}

// RegistryService is the set of registry operations the terminal handler needs.
type RegistryService interface {
	Get(name, version string) (domain.Registry, error)
	List(name string) ([]domain.Registry, error)
}

// BundleService is the set of bundle operations the terminal handler needs.
type BundleService interface {
	Export(phpVersion, outputPath string) error
	Import(bundlePath, phpVersion string) error
}

// SystemService is the set of system operations the terminal handler needs.
type SystemService interface {
	Check(phpDeps []domain.SystemPackage) (*domain.SystemCheckResult, error)
	CheckBuildTools(toolNames []string) (*domain.SystemCheckResult, error)
	Install(packages []domain.SystemPackage) error
	InstallCommand(packages []domain.SystemPackage) string
	DependencySet(exts []domain.ExtensionDef) []domain.SystemPackage
}

// ShimService is the set of shim operations the terminal handler needs.
type ShimService interface {
	RegenerateAll() error
	WritePhar(name, pharRelPath string) error
	SetSystemMode(enabled bool) error
}
