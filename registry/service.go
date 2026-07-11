package registry

import (
	"github.com/supanadit/phpv/domain"
)

type RegistryRepository interface {
	List(name string, checksum bool, os string) (r []domain.Registry, err error)
	Get(name string, version string, checksum bool, os string) (r domain.Registry, err error)
}

type Service struct {
	registryRepository RegistryRepository
}

func NewService(rr RegistryRepository) *Service {
	return &Service{
		registryRepository: rr,
	}
}

func (reg *Service) List(name string, checksum bool, os string) (r []domain.Registry, err error) {
	return reg.registryRepository.List(name, checksum, os)
}

func (reg *Service) Get(name string, checksum bool, version string, os string) (r domain.Registry, err error) {
	return reg.registryRepository.Get(name, version, checksum, os)
}
