package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type FlexRepository struct{}

func NewFlexRepository() *FlexRepository {
	return &FlexRepository{}
}

func (r *FlexRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(2, 6, 23, 25)
	versions = append(versions, r.generateRangeVersions(2, 6, 21, 22)...)
	versions = append(versions, r.generateRangeVersions(2, 6, 20, 20)...)
	versions = append(versions, r.generateRangeVersions(2, 6, 18, 19)...)
	versions = append(versions, r.generateRangeVersions(2, 6, 13, 17)...)
	versions = append(versions, r.generateRangeVersions(2, 6, 8, 12)...)
	versions = append(versions, domain.Source{Name: "flex", Version: "2.5.39", URL: "https://github.com/westes/flex/releases/download/flex-2.5.39/flex-2.5.39.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *FlexRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "flex",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://github.com/westes/flex/releases/download/v%d.%d.%d/flex-%d.%d.%d.tar.gz", major, minor, patch, major, minor, patch),
		})
	}
	return versions
}
