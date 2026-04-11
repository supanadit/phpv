package forge

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/flagresolver"
)

type ForgeRepository interface {
	Build(config domain.ForgeConfig, sourceDir string) (domain.Forge, error)
	BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy, sourceDir string) (domain.Forge, error)
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

func (s *Service) Build(config domain.ForgeConfig, sourceDir string) (domain.Forge, error) {
	return s.forgeRepository.Build(config, sourceDir)
}

func (s *Service) BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy, sourceDir string) (domain.Forge, error) {
	return s.forgeRepository.BuildWithStrategy(config, strategy, sourceDir)
}

func (s *Service) GetConfigureFlags(name string, version string) []string {
	return s.flagResolver.GetConfigureFlags(name, version)
}

func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return s.flagResolver.GetPHPConfigureFlags(phpVersion, extensions)
}

func (s *Service) GetBuildStrategyForTool(name string) domain.BuildStrategy {
	switch name {
	case "autoconf", "automake", "libtool":
		return domain.StrategyConfigureMake
	case "m4":
		return domain.StrategyMakeOnly
	default:
		return domain.StrategyMakeOnly
	}
}
