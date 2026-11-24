package args

import (
	"context"

	"github.com/supanadit/phpv/domain"
)

// PECLService is the set of PECL operations the terminal handler needs.
type PECLService interface {
	Install(ctx context.Context, source, phpVersion string, jobs int) (*domain.PECLInstallResult, error)
	InstallCompatible(ctx context.Context, req domain.PECLInstallRequest) (*domain.PECLInstallResult, error)
	ResolveCompatibleRelease(name, phpVersion string) (string, error)
	List(phpVersion string) ([]domain.ExtensionState, error)
	Uninstall(name, phpVersion string) error
}

// PatchLister lists patches that apply to a package version.
type PatchLister interface {
	EntriesFor(name, version string) []domain.PatchEntry
}

// ConfigService is the set of config operations the terminal handler needs.
type ConfigService interface {
	Get(key string) (string, error)
	Set(key, value string) error
	List() ([]string, error)
}

// DoctorService is the set of doctor operations the terminal handler needs.
type DoctorService interface {
	Check(root string) []domain.Issue
}

// UpdateService is the set of update operations the terminal handler needs.
type UpdateService interface {
	CheckForUpdate() (latest string, hasUpdate bool, err error)
	SelfUpdate(ctx context.Context) error
}
