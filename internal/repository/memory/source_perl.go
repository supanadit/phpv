package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type PerlRepository struct{}

func NewPerlRepository() *PerlRepository {
	return &PerlRepository{}
}

func (r *PerlRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(5, 28, 0, 18)
	versions = append(versions, r.generateRangeVersions(5, 26, 0, 7)...)
	versions = append(versions, r.generateRangeVersions(5, 24, 0, 7)...)
	versions = append(versions, r.generateRangeVersions(5, 22, 0, 7)...)
	versions = append(versions, r.generateRangeVersions(5, 20, 0, 3)...)
	versions = append(versions, r.generateRangeVersions(5, 18, 0, 4)...)
	versions = append(versions, r.generateRangeVersions(5, 16, 0, 4)...)
	versions = append(versions, r.generateRangeVersions(5, 14, 0, 6)...)
	versions = append(versions, r.generateRangeVersions(5, 12, 0, 5)...)
	versions = append(versions, r.generateRangeVersions(5, 10, 0, 5)...)

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *PerlRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "perl",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://www.cpan.org/src/5.0/perl-%d.%d.%d.tar.gz", major, minor, patch),
		})
	}
	return versions
}
