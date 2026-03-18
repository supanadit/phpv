package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type AutoconfRepository struct{}

func NewAutoconfRepository() *AutoconfRepository {
	return &AutoconfRepository{}
}

func (r *AutoconfRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "autoconf", Version: "2.72", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.72.tar.xz"},
		{Name: "autoconf", Version: "2.71", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.71.tar.xz"},
		{Name: "autoconf", Version: "2.69", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.69.tar.xz"},
		{Name: "autoconf", Version: "2.59", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.59.tar.gz"},
		{Name: "autoconf", Version: "2.13", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.13.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
