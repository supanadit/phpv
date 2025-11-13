package usecase

import (
	"context"

	"github.com/supanadit/phpv/domain"
)

// InstallationUsecase represent the installation usecase interface
type InstallationUsecase interface {
	InstallVersion(ctx context.Context, version string) error
	SwitchVersion(ctx context.Context, version string) error
	ListInstalledVersions(ctx context.Context) ([]domain.Installation, error)
	GetActiveVersion(ctx context.Context) (domain.Installation, error)
	UninstallVersion(ctx context.Context, version string) error
}
