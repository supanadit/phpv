package forge

import "github.com/supanadit/phpv/domain"

type ForgeRepository interface {
	Build(config domain.ForgeConfig) (domain.Forge, error)
	BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error)
	GetConfigureFlags(name string) []string
	GetPHPConfigureFlags(phpVersion string, extensions []string) []string
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

func (s *Service) GetConfigureFlags(name string) []string {
	if repo, ok := s.forgeRepository.(interface {
		GetConfigureFlags(string) []string
	}); ok {
		return repo.GetConfigureFlags(name)
	}
	return nil
}

func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	if repo, ok := s.forgeRepository.(interface {
		GetPHPConfigureFlags(string, []string) []string
	}); ok {
		return repo.GetPHPConfigureFlags(phpVersion, extensions)
	}
	return nil
}
