package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type CurlRepository struct{}

func NewCurlRepository() *CurlRepository {
	return &CurlRepository{}
}

func (r *CurlRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(8, 10, 0, 12)
	versions = append(versions, r.generateRangeVersions(8, 9, 0, 7)...)
	versions = append(versions, r.generateRangeVersions(8, 8, 0, 4)...)
	versions = append(versions, r.generateRangeVersions(8, 7, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(8, 6, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(8, 5, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(8, 4, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(8, 3, 0, 2)...)

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *CurlRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "curl",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://curl.se/download/curl-%d.%d.%d.tar.gz", major, minor, patch),
		})
	}
	return versions
}
