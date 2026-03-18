package unload

import (
	"github.com/supanadit/phpv/domain"
)

type UnloadRepository interface {
	Unpack(source, destination string) (*domain.Unload, error)
}

type Service struct {
	unloadRepository UnloadRepository
}

func NewService(unloadRepository UnloadRepository) *Service {
	return &Service{
		unloadRepository: unloadRepository,
	}
}

func (s *Service) Unpack(source, destination string) (*domain.Unload, error) {
	return s.unloadRepository.Unpack(source, destination)
}
