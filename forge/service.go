package forge

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/flagresolver"
)

type ForgeRepository interface {
	Build(config domain.ForgeConfig) (domain.Forge, error)
	BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error)
}

type Service struct {
	forgeRepository ForgeRepository
	flagResolver    *flagresolver.Service
}

func NewService(forgeRepository ForgeRepository, flagResolver *flagresolver.Service) *Service {
	return &Service{
		forgeRepository: forgeRepository,
		flagResolver:    flagResolver,
	}
}

func (s *Service) Build(config domain.ForgeConfig) (domain.Forge, error) {
	return s.forgeRepository.Build(config)
}

func (s *Service) BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error) {
	return s.forgeRepository.BuildWithStrategy(config, strategy)
}

func (s *Service) GetConfigureFlags(name string) []string {
	return s.flagResolver.GetConfigureFlags(name)
}

func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return s.flagResolver.GetPHPConfigureFlags(phpVersion, extensions)
}
