package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type FlexRepository struct{}

func NewFlexRepository() *FlexRepository {
	return &FlexRepository{}
}

func (r *FlexRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "flex", Version: "2.5.39", URL: "https://github.com/westes/flex/releases/download/flex-2.5.39/flex-2.5.39.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
