package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type M4Repository struct{}

func NewM4Repository() *M4Repository {
	return &M4Repository{}
}

func (r *M4Repository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "m4", Version: "1.4.19", URL: "https://mirror.freedif.org/GNU/m4/m4-1.4.19.tar.xz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
