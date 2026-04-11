package disk

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) buildMakeOnly(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	ctx := utils.NewExecContext(config.Verbose)
	jobs := utils.GetJobs(config.Jobs)

	mk := ctx.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env

	if err := ctx.Run(mk); err != nil {
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	if err := r.makeInstall(sourcePath, jobs, env, config.Verbose); err != nil {
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	return domain.Forge{Prefix: prefix}, nil
}
