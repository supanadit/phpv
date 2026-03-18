package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type Re2cRepository struct{}

func NewRe2cRepository() *Re2cRepository {
	return &Re2cRepository{}
}

func (r *Re2cRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(3, 0, 0, 7)
	versions = append(versions, r.generateRangeVersions(2, 2, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(2, 1, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(2, 0, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(1, 2, 0, 4)...)
	versions = append(versions, r.generateRangeVersions(1, 1, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(1, 0, 0, 2)...)
	versions = append(versions, domain.Source{Name: "re2c", Version: "0.16", URL: "https://github.com/skvadrik/re2c/releases/download/0.16/re2c-0.16.tar.gz"})
	versions = append(versions, domain.Source{Name: "re2c", Version: "0.14", URL: "https://github.com/skvadrik/re2c/releases/download/0.14/re2c-0.14.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *Re2cRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "re2c",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://github.com/skvadrik/re2c/releases/download/%d.%d.%d/re2c-%d.%d.%d.tar.xz", major, minor, patch, major, minor, patch),
		})
	}
	return versions
}
