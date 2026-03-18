package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type CurlRepository struct{}

func NewCurlRepository() *CurlRepository {
	return &CurlRepository{}
}

func (r *CurlRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "curl", Version: "8.10.1", URL: "https://curl.se/download/curl-8.10.1.tar.gz"},
		{Name: "curl", Version: "7.88.1", URL: "https://curl.se/download/curl-7.88.1.tar.gz"},
		{Name: "curl", Version: "7.20.0", URL: "https://curl.se/download/curl-7.20.0.tar.gz"},
		{Name: "curl", Version: "7.12.1", URL: "https://curl.se/download/curl-7.12.1.tar.gz"},
		{Name: "curl", Version: "7.12.0", URL: "https://curl.se/download/curl-7.12.0.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
