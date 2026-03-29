package disk

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func (r *ForgeRepository) make(sourcePath string, jobs int, env []string, verbose bool) error {
	return r.makeWithName(sourcePath, jobs, env, "", verbose)
}

func (r *ForgeRepository) makeWithName(sourcePath string, jobs int, env []string, pkgName string, verbose bool) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	if pkgName == "m4" {
		env = append(env, "M4_MAINTAINER_MODE=no")
	}

	var stdout, stderr io.Writer = os.Stdout, os.Stderr
	if !verbose {
		stdout = io.Discard
		stderr = io.Discard
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
		return fmt.Errorf("make failed: %w", err)
	}

	return nil
}

func (r *ForgeRepository) makeInstall(sourcePath string, jobs int, env []string, verbose bool) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	var stdout, stderr io.Writer = os.Stdout, os.Stderr
	if !verbose {
		stdout = io.Discard
		stderr = io.Discard
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env
	mkInstall.Stdout = stdout
	mkInstall.Stderr = stderr

	if verbose {
		fmt.Println("Running make install for", sourcePath)
	}
	if err := mkInstall.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}
