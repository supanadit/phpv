package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type ZlibRepository struct{}

func NewZlibRepository() *ZlibRepository {
	return &ZlibRepository{}
}

func (r *ZlibRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(1, 3, 0, 5)
	versions = append(versions, r.generateRangeVersions(1, 2, 11, 13)...)
	versions = append(versions, r.generateRangeVersions(1, 2, 10, 9)...)

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *ZlibRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "zlib",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://github.com/madler/zlib/releases/download/v%d.%d.%d/zlib-%d.%d.%d.tar.gz", major, minor, patch, major, minor, patch),
		})
	}
	return versions
}
