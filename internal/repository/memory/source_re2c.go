package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type Re2cRepository struct{}

func NewRe2cRepository() *Re2cRepository {
	return &Re2cRepository{}
}

func (r *Re2cRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "re2c", Version: "3.1", URL: "https://github.com/skvadrik/re2c/releases/download/3.1/re2c-3.1.tar.xz"},
		{Name: "re2c", Version: "2.2", URL: "https://github.com/skvadrik/re2c/releases/download/2.2/re2c-2.2.tar.xz"},
		{Name: "re2c", Version: "1.3", URL: "https://github.com/skvadrik/re2c/releases/download/1.3/re2c-1.3.tar.xz"},
		{Name: "re2c", Version: "0.16", URL: "https://github.com/skvadrik/re2c/releases/download/0.16/re2c-0.16.tar.gz"},
		{Name: "re2c", Version: "0.14", URL: "https://github.com/skvadrik/re2c/releases/download/0.14/re2c-0.14.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
