package registry

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
)

type RegistryRepository interface {
	List(name string) (r []domain.Registry, err error)
	Get(name string, version string) (r domain.Registry, err error)
}

type Service struct {
	registryRepository RegistryRepository
}

func NewService(rr RegistryRepository) *Service {
	return &Service{
		registryRepository: rr,
	}
}

func (reg *Service) List(name string) (r []domain.Registry, err error) {
	return reg.registryRepository.List(name)
}

func (reg *Service) Get(name string, version string) (r domain.Registry, err error) {
	registries, err := reg.List(name)
	if err != nil {
		return r, err
	}
	for _, registry := range registries {
		if registry.Name == version {
			return registry, nil
		}
	}
	return r, fmt.Errorf("registry %s version %s not found", name, version)
}
