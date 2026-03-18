package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type Libxml2Repository struct{}

func NewLibxml2Repository() *Libxml2Repository {
	return &Libxml2Repository{}
}

func (r *Libxml2Repository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(2, 12, 0, 8)
	versions = append(versions, r.generateRangeVersions(2, 11, 0, 5)...)
	versions = append(versions, r.generateRangeVersions(2, 10, 0, 8)...)
	versions = append(versions, r.generateRangeVersions(2, 9, 0, 18)...)
	versions = append(versions, r.generateRangeVersions(2, 7, 0, 28)...)
	versions = append(versions, r.generateRangeVersions(2, 6, 32, 33)...)
	versions = append(versions, domain.Source{Name: "libxml2", Version: "2.6.30", URL: "https://github.com/GNOME/libxml2/archive/refs/tags/LIBXML2_2_6_30.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *Libxml2Repository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "libxml2",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://download.gnome.org/sources/libxml2/%d.%d/libxml2-%d.%d.%d.tar.xz", major, minor, major, minor, patch),
		})
	}
	return versions
}
