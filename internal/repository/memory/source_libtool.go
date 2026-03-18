package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type LibtoolRepository struct{}

func NewLibtoolRepository() *LibtoolRepository {
	return &LibtoolRepository{}
}

func (r *LibtoolRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "libtool", Version: "2.5.4", URL: "https://mirror.freedif.org/GNU/libtool/libtool-2.5.4.tar.xz"},
		{Name: "libtool", Version: "2.4.7", URL: "https://mirror.freedif.org/GNU/libtool/libtool-2.4.7.tar.xz"},
		{Name: "libtool", Version: "2.4.6", URL: "https://mirror.freedif.org/GNU/libtool/libtool-2.4.6.tar.xz"},
		{Name: "libtool", Version: "1.5.26", URL: "https://mirror.freedif.org/GNU/libtool/libtool-1.5.26.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
