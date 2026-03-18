package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type Libxml2Repository struct{}

func NewLibxml2Repository() *Libxml2Repository {
	return &Libxml2Repository{}
}

func (r *Libxml2Repository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "libxml2", Version: "2.12.7", URL: "https://download.gnome.org/sources/libxml2/2.12/libxml2-2.12.7.tar.xz"},
		{Name: "libxml2", Version: "2.11.7", URL: "https://download.gnome.org/sources/libxml2/2.11/libxml2-2.11.7.tar.xz"},
		{Name: "libxml2", Version: "2.9.14", URL: "https://download.gnome.org/sources/libxml2/2.9/libxml2-2.9.14.tar.xz"},
		{Name: "libxml2", Version: "2.6.30", URL: "https://github.com/GNOME/libxml2/archive/refs/tags/LIBXML2_2_6_30.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
