package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type M4Repository struct{}

func NewM4Repository() *M4Repository {
	return &M4Repository{}
}

func (r *M4Repository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(1, 4, 25, 26)
	versions = append(versions, r.generateRangeVersions(1, 4, 22, 24)...)
	versions = append(versions, r.generateRangeVersions(1, 4, 19, 21)...)
	versions = append(versions, r.generateRangeVersions(1, 4, 10, 18)...)

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *M4Repository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "m4",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://mirror.freedif.org/GNU/m4/m4-%d.%d.%d.tar.xz", major, minor, patch),
		})
	}
	return versions
}
