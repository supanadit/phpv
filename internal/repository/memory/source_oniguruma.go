package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type OnigurumaRepository struct{}

func NewOnigurumaRepository() *OnigurumaRepository {
	return &OnigurumaRepository{}
}

func (r *OnigurumaRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(6, 9, 211, 213)
	versions = append(versions, r.generateRangeVersions(6, 9, 200, 210)...)
	versions = append(versions, r.generateRangeVersions(6, 9, 100, 110)...)
	versions = append(versions, r.generateRangeVersions(6, 9, 0, 9)...)

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *OnigurumaRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "oniguruma",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://github.com/kkos/oniguruma/releases/download/v%d.%d.%d/onig-%d.%d.%d.tar.gz", major, minor, patch, major, minor, patch),
		})
	}
	return versions
}
