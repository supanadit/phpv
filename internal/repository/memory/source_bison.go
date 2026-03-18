package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type BisonRepository struct{}

func NewBisonRepository() *BisonRepository {
	return &BisonRepository{}
}

func (r *BisonRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "bison", Version: "1.28", URL: "https://mirror.freedif.org/GNU/bison/bison-1.28.tar.gz"},
		{Name: "bison", Version: "1.35", URL: "https://mirror.freedif.org/GNU/bison/bison-1.35.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
