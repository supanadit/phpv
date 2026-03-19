package forge

import "github.com/supanadit/phpv/domain"

type ForgeRepository interface {
	Build(version string) (domain.Forge, error)
}

type Service struct {
	forgeRepository ForgeRepository
}

func NewService(forgeRepository ForgeRepository) *Service {
	return &Service{
		forgeRepository: forgeRepository,
	}
}

func (s *Service) Build(version string) (domain.Forge, error) {
	return s.forgeRepository.Build(version)
}
