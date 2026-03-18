package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type SourceRepository struct {
	// In Memory doesn't receive any external input
}

func NewSourceRepository() *SourceRepository {
	return &SourceRepository{}
}

func (r *SourceRepository) GetVersions() ([]domain.Source, error) {
	// TODO: Add RCx, beta, alpha versions
	versions := r.generateRangeVersions(8, 5, 0, 4)
	versions = append(
		versions,
		r.generateRangeVersions(8, 4, 0, 19)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 3, 0, 27)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 2, 0, 29)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 1, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 0, 0, 30)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 4, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 3, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 2, 0, 34)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 1, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 0, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 6, 0, 40)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 5, 0, 38)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 4, 0, 45)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 3, 0, 29)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 2, 0, 17)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 1, 0, 6)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 0, 0, 5)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 4, 0, 9)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 3, 0, 11)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 2, 0, 3)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 1, 0, 2)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 0, 0, 6)...,
	)
	// Sort versions descending
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *SourceRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	count := endPatch - startPatch + 1
	versions := make([]domain.Source, 0, count)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "php",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     r.buildDownloadURL(major, minor, patch),
		})
	}
	return versions
}

func (r *SourceRepository) buildDownloadURL(major, minor, patch int) string {
	// TODO: Adding Checksum for download
	// TODO: Adding fallback logic ( Maybe not here but still we need this features )
	versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	if major == 4 {
		return fmt.Sprintf("https://museum.php.net/php4/php-%s.tar.gz", versionStr)
	}
	if major == 5 && minor <= 2 {
		return fmt.Sprintf("https://museum.php.net/php5/php-%s.tar.gz", versionStr)
	}
	return fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", versionStr)
}
