package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type AutomakeRepository struct{}

func NewAutomakeRepository() *AutomakeRepository {
	return &AutomakeRepository{}
}

func (r *AutomakeRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(1, 16, 0, 3)
	versions = append(versions, r.generateRangeVersions(1, 15, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(1, 14, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(1, 13, 0, 5)...)
	versions = append(versions, r.generateRangeVersions(1, 11, 0, 7)...)
	versions = append(versions, r.generateRangeVersions(1, 10, 0, 7)...)
	versions = append(versions, domain.Source{Name: "automake", Version: "1.9.6", URL: "https://mirror.freedif.org/GNU/automake/automake-1.9.6.tar.gz"})
	versions = append(versions, domain.Source{Name: "automake", Version: "1.4-p6", URL: "https://mirror.freedif.org/GNU/automake/automake-1.4-p6.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *AutomakeRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "automake",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://mirror.freedif.org/GNU/automake/automake-%d.%d.%d.tar.xz", major, minor, patch),
		})
	}
	return versions
}
