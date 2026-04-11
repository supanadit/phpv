package disk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) buildAutogen(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	ctx := utils.NewExecContext(config.Verbose)
	jobs := utils.GetJobs(config.Jobs)

	autogenPath := filepath.Join(sourcePath, "autogen.sh")
	if _, err := os.Stat(autogenPath); err == nil {
		autogen := ctx.Command("./autogen.sh")
		autogen.Dir = sourcePath
		autogen.Env = env

		if err := ctx.Run(autogen); err != nil {
			return domain.Forge{}, fmt.Errorf("autogen failed: %w", err)
		}
	}

	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); err == nil {
		if err := os.Chmod(configurePath, 0o755); err != nil {
			return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
		}

		args := []string{fmt.Sprintf("--prefix=%s", prefix)}
		args = append(args, config.ConfigureFlags...)

		configure := ctx.Command("./configure", args...)
		configure.Dir = sourcePath
		configure.Env = env

		if err := ctx.Run(configure); err != nil {
			return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
		}
	}

	if err := r.makeWithName(sourcePath, jobs, env, config.Name, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, jobs, env, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}
