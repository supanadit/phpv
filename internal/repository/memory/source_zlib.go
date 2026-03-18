package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type ZlibRepository struct{}

func NewZlibRepository() *ZlibRepository {
	return &ZlibRepository{}
}

func (r *ZlibRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "zlib", Version: "1.3.1", URL: "https://github.com/madler/zlib/releases/download/v1.3.1/zlib-1.3.1.tar.gz"},
		{Name: "zlib", Version: "1.2.13", URL: "https://github.com/madler/zlib/releases/download/v1.2.13/zlib-1.2.13.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
