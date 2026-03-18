package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type BisonRepository struct{}

func NewBisonRepository() *BisonRepository {
	return &BisonRepository{}
}

func (r *BisonRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(3, 8, 0, 5)
	versions = append(versions, r.generateRangeVersions(3, 7, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(3, 6, 0, 2)...)
	versions = append(versions, r.generateRangeVersions(3, 5, 0, 1)...)
	versions = append(versions, domain.Source{Name: "bison", Version: "1.35", URL: "https://mirror.freedif.org/GNU/bison/bison-1.35.tar.gz"})
	versions = append(versions, domain.Source{Name: "bison", Version: "1.28", URL: "https://mirror.freedif.org/GNU/bison/bison-1.28.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *BisonRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "bison",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://mirror.freedif.org/GNU/bison/bison-%d.%d.%d.tar.xz", major, minor, patch),
		})
	}
	return versions
}
