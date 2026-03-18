package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type OnigurumaRepository struct{}

func NewOnigurumaRepository() *OnigurumaRepository {
	return &OnigurumaRepository{}
}

func (r *OnigurumaRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "oniguruma", Version: "6.9.9", URL: "https://github.com/kkos/oniguruma/releases/download/v6.9.9/onig-6.9.9.tar.gz"},
		{Name: "oniguruma", Version: "6.9.8", URL: "https://github.com/kkos/oniguruma/releases/download/v6.9.8/onig-6.9.8.tar.gz"},
		{Name: "oniguruma", Version: "5.9.6", URL: "https://github.com/kkos/oniguruma/releases/download/v5.9.6/onig-5.9.6.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
