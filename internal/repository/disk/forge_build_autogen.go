package disk

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) buildAutogen(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	jobs := config.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	var filter *utils.ErrorWarningFilter
	if !config.Verbose {
		stdout = io.Discard
		filter = utils.NewErrorWarningFilter(os.Stderr)
		stderr = filter
	}

	autogenPath := filepath.Join(sourcePath, "autogen.sh")
	if _, err := os.Stat(autogenPath); err == nil {
		autogen := exec.Command("./autogen.sh")
		autogen.Dir = sourcePath
		autogen.Env = env
		autogen.Stdout = stdout
		autogen.Stderr = stderr
		if config.Verbose {
			fmt.Println("Running autogen.sh for", config.Name)
		}
		if err := autogen.Run(); err != nil {
			if filter != nil {
				filter.Flush()
			}
			return domain.Forge{}, fmt.Errorf("autogen failed: %w", err)
		}
	}

	if filter != nil {
		filter.Flush()
	}

	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); err == nil {
		if err := os.Chmod(configurePath, 0o755); err != nil {
			return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
		}

		args := []string{fmt.Sprintf("--prefix=%s", prefix)}
		args = append(args, config.ConfigureFlags...)

		configure := exec.Command("./configure", args...)
		configure.Dir = sourcePath
		configure.Env = env
		configure.Stdout = stdout
		configure.Stderr = stderr

		if config.Verbose {
			fmt.Println("Running configure for", config.Name)
		}
		if err := configure.Run(); err != nil {
			if filter != nil {
				filter.Flush()
			}
			return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
		}
	}

	if filter != nil {
		filter.Flush()
	}

	if err := r.makeWithName(sourcePath, jobs, env, config.Name, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, jobs, env, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}
