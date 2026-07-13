package registry

import (
	"runtime"

	"github.com/supanadit/phpv/domain"
)

// RegistryRepository resolves download URLs and checksums for packages.
type RegistryRepository interface {
	List(name string, checksum bool, os string) (r []domain.Registry, err error)
	Get(name string, version string, checksum bool, os string) (r domain.Registry, err error)
}

type Service struct {
	repo RegistryRepository
	os   string
}

func NewService(r RegistryRepository) *Service {
	return &Service{repo: r, os: runtime.GOOS}
}

func (s *Service) Get(name, version string) (domain.Registry, error) {
	return s.repo.Get(name, version, false, s.os)
}

func (s *Service) List(name string) ([]domain.Registry, error) {
	return s.repo.List(name, false, s.os)
}

func (s *Service) ListWithChecksum(name string) ([]domain.Registry, error) {
	return s.repo.List(name, true, s.os)
}
