package usecase

import (
	"context"

	"github.com/supanadit/phpv/domain"
)

// PHPVersionRepository represent the PHP version's repository contract
type PHPVersionRepository interface {
	GetAvailableVersions(ctx context.Context) ([]domain.PHPVersion, error)
	GetVersionByString(ctx context.Context, version string) (domain.PHPVersion, error)
	SaveVersion(ctx context.Context, version domain.PHPVersion) error
}

// InstallationRepository represent the installation's repository contract
type InstallationRepository interface {
	GetAllInstallations(ctx context.Context) ([]domain.Installation, error)
	GetInstallationByVersion(ctx context.Context, version domain.PHPVersion) (domain.Installation, error)
	GetActiveInstallation(ctx context.Context) (domain.Installation, error)
	SaveInstallation(ctx context.Context, installation domain.Installation) error
	SetActiveInstallation(ctx context.Context, installation domain.Installation) error
	DeleteInstallation(ctx context.Context, version domain.PHPVersion) error
}

// Downloader represent the downloader contract for downloading PHP source
type Downloader interface {
	DownloadSource(ctx context.Context, version domain.PHPVersion, destPath string) error
}

// Builder represent the builder contract for building PHP from source
type Builder interface {
	Build(ctx context.Context, sourcePath string, installPath string, config map[string]string) error
}

// FileSystem represent the filesystem operations contract
type FileSystem interface {
	CreateDirectory(path string) error
	RemoveDirectory(path string) error
	FileExists(path string) bool
	DirectoryExists(path string) bool
}
