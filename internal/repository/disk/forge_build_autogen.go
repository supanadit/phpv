package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/supanadit/phpv/domain"
)

func (r *ForgeRepository) buildAutogen(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	jobs := config.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	autogenPath := filepath.Join(sourcePath, "autogen.sh")
	if _, err := os.Stat(autogenPath); err == nil {
		autogen := exec.Command("./autogen.sh")
		autogen.Dir = sourcePath
		autogen.Env = env
		autogen.Stdout = os.Stdout
		autogen.Stderr = os.Stderr
		fmt.Println("Running autogen.sh for", config.Name)
		if err := autogen.Run(); err != nil {
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

		configure := exec.Command("./configure", args...)
		configure.Dir = sourcePath
		configure.Env = env
		configure.Stdout = os.Stdout
		configure.Stderr = os.Stderr

		fmt.Println("Running configure for", config.Name)
		if err := configure.Run(); err != nil {
			return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
		}
	}

	if err := r.makeWithName(sourcePath, jobs, env, config.Name); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, jobs, env); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}
