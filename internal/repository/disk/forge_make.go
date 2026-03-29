package disk

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func (r *ForgeRepository) make(sourcePath string, jobs int, env []string) error {
	return r.makeWithName(sourcePath, jobs, env, "")
}

func (r *ForgeRepository) makeWithName(sourcePath string, jobs int, env []string, pkgName string) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	if pkgName == "m4" {
		env = append(env, "M4_MAINTAINER_MODE=no")
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", sourcePath)
	if err := mk.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return nil
}

func (r *ForgeRepository) makeInstall(sourcePath string, jobs int, env []string) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env
	mkInstall.Stdout = os.Stdout
	mkInstall.Stderr = os.Stderr

	fmt.Println("Running make install for", sourcePath)
	if err := mkInstall.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}
