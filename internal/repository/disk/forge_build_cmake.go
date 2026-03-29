package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/supanadit/phpv/domain"
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

	cmakeArgs := []string{
		"-DCMAKE_INSTALL_PREFIX=" + prefix,
		sourcePath,
	}

	cmakeCmd := exec.Command("cmake", cmakeArgs...)
	cmakeCmd.Dir = buildDir
	cmakeCmd.Env = env
	cmakeCmd.Stdout = os.Stdout
	cmakeCmd.Stderr = os.Stderr

	fmt.Println("Running cmake for", config.Name)
	if err := cmakeCmd.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("cmake failed: %w", err)
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = buildDir
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", config.Name)
	if err := mk.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = buildDir
	mkInstall.Env = env
	mkInstall.Stdout = os.Stdout
	mkInstall.Stderr = os.Stderr

	fmt.Println("Running make install for", config.Name)
	if err := mkInstall.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	return domain.Forge{Prefix: prefix}, nil
}
