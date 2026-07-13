package registry

import (
	"github.com/supanadit/phpv/domain"
)

// RegistryRepository resolves download URLs and checksums for packages.
type RegistryRepository interface {
	List(name string, checksum bool, os string) (r []domain.Registry, err error)
	Get(name string, version string, checksum bool, os string) (r domain.Registry, err error)
}
