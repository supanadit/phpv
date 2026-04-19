package disk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) buildCMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	buildDir := filepath.Join(sourcePath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to create build directory: %w", err)
	}

	ctx := utils.NewExecContext(config.Verbose)
	jobs := utils.GetJobs(config.Jobs)

	cmakeArgs := []string{
		"-DCMAKE_INSTALL_PREFIX=" + prefix,
		sourcePath,
	}

	cmakeCmd := ctx.Command("cmake", cmakeArgs...)
	cmakeCmd.Dir = buildDir
	cmakeCmd.Env = env

	if err := ctx.Run(cmakeCmd); err != nil {
		return domain.Forge{}, fmt.Errorf("cmake failed: %w", err)
	}

	mk := ctx.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = buildDir
	mk.Env = env

	if err := ctx.Run(mk); err != nil {
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	if err := r.makeInstall(buildDir, jobs, env, config.Verbose, config.Name); err != nil {
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	return domain.Forge{Prefix: prefix}, nil
}
