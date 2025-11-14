package version

import (
	"context"

	"github.com/supanadit/phpv/domain"
)

type VersionRepository interface {
	GetVersions(ctx context.Context) ([]domain.Version, error)
}

type Service struct {
	repoVersion VersionRepository
}

func NewService(repoVersion VersionRepository) *Service {
	return &Service{
		repoVersion: repoVersion,
	}
}

func (s *Service) GetVersions(ctx context.Context) ([]domain.Version, error) {
	return s.repoVersion.GetVersions(ctx)
}
