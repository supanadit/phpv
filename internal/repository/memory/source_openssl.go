package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type OpensslRepository struct{}

func NewOpensslRepository() *OpensslRepository {
	return &OpensslRepository{}
}

func (r *OpensslRepository) GetVersions() ([]domain.Source, error) {
	versions := []domain.Source{
		{Name: "openssl", Version: "3.3.2", URL: "https://github.com/openssl/openssl/releases/download/openssl-3.3.2/openssl-3.3.2.tar.gz"},
		{Name: "openssl", Version: "3.0.14", URL: "https://github.com/openssl/openssl/releases/download/openssl-3.0.14/openssl-3.0.14.tar.gz"},
		{Name: "openssl", Version: "1.1.1w", URL: "https://github.com/openssl/openssl/releases/download/openssl-1.1.1w/openssl-1.1.1w.tar.gz"},
		{Name: "openssl", Version: "1.0.1u", URL: "https://www.openssl.org/source/openssl-1.0.1u.tar.gz"},
		{Name: "openssl", Version: "0.9.8zh", URL: "https://www.openssl.org/source/openssl-0.9.8zh.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
