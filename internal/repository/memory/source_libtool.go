package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type LibtoolRepository struct{}

func NewLibtoolRepository() *LibtoolRepository {
	return &LibtoolRepository{}
}

func (r *LibtoolRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(2, 4, 9, 23)
	versions = append(versions, r.generateRangeVersions(2, 4, 0, 8)...)
	versions = append(versions, domain.Source{Name: "libtool", Version: "1.5.26", URL: "https://mirror.freedif.org/GNU/libtool/libtool-1.5.26.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *LibtoolRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "libtool",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://mirror.freedif.org/GNU/libtool/libtool-%d.%d.%d.tar.xz", major, minor, patch),
		})
	}
	return versions
}
