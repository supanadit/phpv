package forge

import "github.com/supanadit/phpv/domain"

type ForgeRepository interface {
	Build(config domain.ForgeConfig) (domain.Forge, error)
	BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error)
}

type Service struct {
	forgeRepository ForgeRepository
}

func NewService(forgeRepository ForgeRepository) *Service {
	return &Service{
		forgeRepository: forgeRepository,
	}
}

func (s *Service) Build(config domain.ForgeConfig) (domain.Forge, error) {
	return s.forgeRepository.Build(config)
}

func (s *Service) BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error) {
	return s.forgeRepository.BuildWithStrategy(config, strategy)
}
