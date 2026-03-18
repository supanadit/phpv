package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type AutomakeRepository struct{}

func NewAutomakeRepository() *AutomakeRepository {
	return &AutomakeRepository{}
}

func (r *AutomakeRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "automake", Version: "1.17", URL: "https://mirror.freedif.org/GNU/automake/automake-1.17.tar.xz"},
		{Name: "automake", Version: "1.16.5", URL: "https://mirror.freedif.org/GNU/automake/automake-1.16.5.tar.xz"},
		{Name: "automake", Version: "1.15.1", URL: "https://mirror.freedif.org/GNU/automake/automake-1.15.1.tar.xz"},
		{Name: "automake", Version: "1.9.6", URL: "https://mirror.freedif.org/GNU/automake/automake-1.9.6.tar.gz"},
		{Name: "automake", Version: "1.4-p6", URL: "https://mirror.freedif.org/GNU/automake/automake-1.4-p6.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
