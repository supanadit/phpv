package disk

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
)

func (r *ForgeRepository) buildConfigureMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); os.IsNotExist(err) {
		return domain.Forge{}, fmt.Errorf("configure script not found at %s", configurePath)
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
	}

	r.touchAutotools(sourcePath)

	var stdout, stderr io.Writer = os.Stdout, os.Stderr
	if !config.Verbose {
		stdout = io.Discard
		stderr = io.Discard
	}

	if config.Name == "m4" {
		autoreconf := exec.Command("autoreconf", "-fi")
		autoreconf.Dir = sourcePath
		autoreconf.Env = env
		autoreconf.Stdout = stdout
		autoreconf.Stderr = stderr
		if config.Verbose {
			fmt.Println("Running autoreconf for m4")
		}
		if err := autoreconf.Run(); err != nil {
			return domain.Forge{}, fmt.Errorf("autoreconf failed: %w", err)
		}
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
		return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
	}

	if err := r.makeWithName(sourcePath, config.Jobs, env, config.Name, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, config.Jobs, env, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}
