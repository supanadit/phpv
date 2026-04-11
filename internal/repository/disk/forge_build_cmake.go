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

func (r *ForgeRepository) buildCMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	buildDir := filepath.Join(sourcePath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to create build directory: %w", err)
	}

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

	cmakeArgs := []string{
		"-DCMAKE_INSTALL_PREFIX=" + prefix,
		sourcePath,
	}

	cmakeCmd := exec.Command("cmake", cmakeArgs...)
	cmakeCmd.Dir = buildDir
	cmakeCmd.Env = env
	cmakeCmd.Stdout = stdout
	cmakeCmd.Stderr = stderr

	if config.Verbose {
		fmt.Println("Running cmake for", config.Name)
	}
	if err := cmakeCmd.Run(); err != nil {
		if filter != nil {
			filter.Flush()
		}
		return domain.Forge{}, fmt.Errorf("cmake failed: %w", err)
	}

	if filter != nil {
		filter.Flush()
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = buildDir
	mk.Env = env
	mk.Stdout = stdout
	mk.Stderr = stderr

	if config.Verbose {
		fmt.Println("Running make for", config.Name)
	}
	if err := mk.Run(); err != nil {
		if filter != nil {
			filter.Flush()
		}
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	if filter != nil {
		filter.Flush()
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = buildDir
	mkInstall.Env = env
	mkInstall.Stdout = stdout
	mkInstall.Stderr = stderr

	if config.Verbose {
		fmt.Println("Running make install for", config.Name)
	}
	if err := mkInstall.Run(); err != nil {
		if filter != nil {
			filter.Flush()
		}
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	if filter != nil {
		filter.Flush()
	}

	return domain.Forge{Prefix: prefix}, nil
}
