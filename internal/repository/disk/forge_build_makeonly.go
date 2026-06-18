package disk

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
)

func (r *ForgeRepository) buildMakeOnly(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	ctx := NewExecContext(config.Verbose)
	jobs := Jobs(config.Jobs)

	makeArgs := []string{fmt.Sprintf("-j%d", jobs)}
	if config.Name == "automake" || config.Name == "autoconf" {
		makeArgs = []string{"-j1"}
	}

	mk := ctx.Command("make", makeArgs...)
	mk.Dir = sourcePath
	mk.Env = env

	if err := ctx.Run(mk); err != nil {
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	if err := r.makeInstall(sourcePath, jobs, env, config.Verbose, config.Name); err != nil {
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	return domain.Forge{Prefix: prefix}, nil
}
