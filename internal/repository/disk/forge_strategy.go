package disk

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) detectStrategy(name, version string) domain.BuildStrategy {
	switch name {
	case "zlib":
		return domain.StrategyConfigureMake
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

func (r *ForgeRepository) logInfo(msg string, args ...interface{}) {
	if r.logger != nil {
		r.logger.Info(msg, args...)
	}
}

func (r *ForgeRepository) BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy, sourceDir string) (domain.Forge, error) {
	r.logInfo("[forge] Compiling %s %s...", config.Name, config.Version)

	silo, err := r.siloRepo.GetSilo()
	if err != nil {
		return domain.Forge{}, err
	}

	installDir := config.Prefix
	if installDir == "" {
		installDir = utils.GetVersionPath(silo, config.Name, config.Version)
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
		return domain.Forge{}, fmt.Errorf("[forge] unsupported build strategy: %s", strategy)
	}
}
