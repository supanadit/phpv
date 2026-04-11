package disk

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) buildMakeOnly(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
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

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
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
	mkInstall.Dir = sourcePath
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
