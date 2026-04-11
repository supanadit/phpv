package disk

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) makeWithName(sourcePath string, jobs int, env []string, pkgName string, verbose bool) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	if pkgName == "m4" {
		env = append(env, "M4_MAINTAINER_MODE=no")
	}

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	var filter *utils.ErrorWarningFilter
	if !verbose {
		stdout = io.Discard
		filter = utils.NewErrorWarningFilter(os.Stderr)
		stderr = filter
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env
	mk.Stdout = stdout
	mk.Stderr = stderr

	if verbose {
		fmt.Println("Running make for", sourcePath)
	}
	if err := mk.Run(); err != nil {
		if filter != nil {
			filter.Flush()
		}
		return fmt.Errorf("make failed: %w", err)
	}

	if filter != nil {
		filter.Flush()
	}

	return nil
}

func (r *ForgeRepository) makeInstall(sourcePath string, jobs int, env []string, verbose bool) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	var filter *utils.ErrorWarningFilter
	if !verbose {
		stdout = io.Discard
		filter = utils.NewErrorWarningFilter(os.Stderr)
		stderr = filter
	}

	mkInstall := exec.Command("make", fmt.Sprintf("-j%d", jobs), "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env
	mkInstall.Stdout = stdout
	mkInstall.Stderr = stderr

	if verbose {
		fmt.Println("Running make install for", sourcePath)
	}
	if err := mkInstall.Run(); err != nil {
		if filter != nil {
			filter.Flush()
		}
		return fmt.Errorf("make install failed: %w", err)
	}

	if filter != nil {
		filter.Flush()
	}

	return nil
}
