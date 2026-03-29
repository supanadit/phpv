package disk

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/supanadit/phpv/domain"
)

func (r *ForgeRepository) buildMakeOnly(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	jobs := config.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", config.Name)
	if err := mk.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env
	mkInstall.Stdout = os.Stdout
	mkInstall.Stderr = os.Stderr

	fmt.Println("Running make install for", config.Name)
	if err := mkInstall.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	return domain.Forge{Prefix: prefix}, nil
}
