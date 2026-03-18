package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type CmakeRepository struct{}

func NewCmakeRepository() *CmakeRepository {
	return &CmakeRepository{}
}

func (r *CmakeRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(3, 28, 0, 5)
	versions = append(versions, r.generateRangeVersions(3, 27, 0, 6)...)
	versions = append(versions, r.generateRangeVersions(3, 26, 0, 5)...)
	versions = append(versions, r.generateRangeVersions(3, 25, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(3, 24, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(3, 23, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(3, 22, 0, 1)...)
	versions = append(versions, r.generateRangeVersions(3, 21, 0, 4)...)

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *CmakeRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "cmake",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://github.com/Kitware/CMake/releases/download/v%d.%d.%d/cmake-%d.%d.%d-linux-x86_64.tar.gz", major, minor, patch, major, minor, patch),
		})
	}
	return versions
}
