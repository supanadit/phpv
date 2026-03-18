package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type OpensslRepository struct{}

func NewOpensslRepository() *OpensslRepository {
	return &OpensslRepository{}
}

func (r *OpensslRepository) GetVersions() ([]domain.Source, error) {
	versions := r.generateRangeVersions(3, 0, 18, 24)
	versions = append(versions, r.generateRangeVersions(3, 0, 15, 17)...)
	versions = append(versions, r.generateRangeVersions(3, 0, 12, 14)...)
	versions = append(versions, r.generateRangeVersions(3, 0, 8, 11)...)
	versions = append(versions, r.generateRangeVersions(3, 0, 5, 7)...)
	versions = append(versions, r.generateRangeVersions(3, 0, 1, 4)...)
	versions = append(versions, domain.Source{Name: "openssl", Version: "1.0.1u", URL: "https://www.openssl.org/source/openssl-1.0.1u.tar.gz"})
	versions = append(versions, domain.Source{Name: "openssl", Version: "0.9.8zh", URL: "https://www.openssl.org/source/openssl-0.9.8zh.tar.gz"})

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *OpensslRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "openssl",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://github.com/openssl/openssl/releases/download/openssl-%d.%d.%d/openssl-%d.%d.%d.tar.gz", major, minor, patch, major, minor, patch),
		})
	}
	return versions
}
