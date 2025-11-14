package memory

import (
	"context"

	"github.com/supanadit/phpv/domain"
)

type VersionRepository struct {
}

func NewVersionRepository() *VersionRepository {
	return &VersionRepository{}
}

func (r *VersionRepository) GetVersions(ctx context.Context) ([]domain.Version, error) {
	return []domain.Version{
		{Major: 8, Minor: 1, Patch: 0},
		{Major: 8, Minor: 0, Patch: 0},
		{Major: 7, Minor: 4, Patch: 0},
	}, nil
}
