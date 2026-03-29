package disk

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
)

func (r *ForgeRepository) detectStrategy(name, version string) domain.BuildStrategy {
	switch name {
	case "zlib":
		return domain.StrategyMakeOnly
	case "cmake":
		return domain.StrategyCMake
	case "autoconf", "automake", "flex", "bison", "perl":
		return domain.StrategyMakeOnly
	case "openssl":
		return domain.StrategyConfigureMake
	case "php":
		return domain.StrategyConfigureMake
	default:
		return domain.StrategyConfigureMake
	}
}

func (r *ForgeRepository) BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error) {
	url, err := r.resolveURL(config.Name, config.Version)
	if err != nil {
		return domain.Forge{}, err
	}

	if err := r.ensureSource(config.Name, config.Version, url); err != nil {
		return domain.Forge{}, err
	}

	fmt.Printf("Compiling %s %s...\n", config.Name, config.Version)

	silo, err := r.siloRepo.GetSilo()
	if err != nil {
		return domain.Forge{}, err
	}

	sourceDir := silo.GetSourceDirPath(config.Name, config.Version)
	installDir := config.Prefix
	if installDir == "" {
		installDir = silo.GetVersionPath(config.Name, config.Version)
	}

	r.ensureFs()

	r.chmodBuildScripts(sourceDir)

	env := r.buildEnv(config)

	switch strategy {
	case domain.StrategyCMake:
		return r.buildCMake(sourceDir, installDir, config, env)
	case domain.StrategyMakeOnly:
		return r.buildMakeOnly(sourceDir, installDir, config, env)
	case domain.StrategyConfigureMake:
		return r.buildConfigureMake(sourceDir, installDir, config, env)
	case domain.StrategyAutogen:
		return r.buildAutogen(sourceDir, installDir, config, env)
	default:
		return domain.Forge{}, fmt.Errorf("unsupported build strategy: %s", strategy)
	}
}
