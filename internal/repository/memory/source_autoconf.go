package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type AutoconfRepository struct{}

func NewAutoconfRepository() *AutoconfRepository {
	return &AutoconfRepository{}
}

func (r *AutoconfRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(2, 71, 0, 3)
	versions = append(versions, r.generateRangeVersions(2, 69, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(2, 13, 0, 1)...)
	versions = append(versions, domain.Source{Name: "autoconf", Version: "2.13", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.13.tar.gz"})
	versions = append(versions, domain.Source{Name: "autoconf", Version: "2.59", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.59.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *AutoconfRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "autoconf",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://mirror.freedif.org/GNU/autoconf/autoconf-%d.%d.%d.tar.xz", major, minor, patch),
		})
	}
	return versions
}
