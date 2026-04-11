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

func getOpenSSLConfigureTarget() string {
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "aarch64"
	}
	switch runtime.GOOS {
	case "linux":
		return "linux-" + goarch
	case "darwin":
		if goarch == "x86_64" {
			return "darwin64-x86_64-cc"
		} else if goarch == "aarch64" {
			return "darwin64-arm64-cc"
		}
		return "darwin-" + goarch + "-cc"
	default:
		return ""
	}
}

func (r *ForgeRepository) findConfigureInSubdir(basePath, name string) string {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			configPath := filepath.Join(basePath, entry.Name(), name)
			if _, err := os.Stat(configPath); err == nil {
				return configPath
			}
			if found := r.findConfigureInSubdir(filepath.Join(basePath, entry.Name()), name); found != "" {
				return found
			}
		}
	}
	return ""
}

func (r *ForgeRepository) buildConfigureMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	configurePath := filepath.Join(sourcePath, "configure")
	ConfigurePath := filepath.Join(sourcePath, "Configure")
	configPath := filepath.Join(sourcePath, "config")
	useConfigure := true
	usesPerl := false
	isOpensslConfig := false

	if _, err := os.Stat(configurePath); os.IsNotExist(err) {
		if config.Name == "openssl" || config.Name == "ossl" {
			if _, err := os.Stat(ConfigurePath); err == nil {
				configurePath = ConfigurePath
				useConfigure = false
				usesPerl = true
			} else if found := r.findConfigureInSubdir(sourcePath, "Configure"); found != "" {
				configurePath = found
				useConfigure = false
				usesPerl = true
			} else if _, err := os.Stat(configPath); err == nil {
				configurePath = configPath
				useConfigure = false
				isOpensslConfig = true
			} else {
				return domain.Forge{}, fmt.Errorf("configure script not found for %s (checked ./Configure, ./config, and subdirectories)", config.Name)
			}
		} else if _, err := os.Stat(ConfigurePath); os.IsNotExist(err) {
			if found := r.findConfigureInSubdir(sourcePath, "Configure"); found != "" {
				configurePath = found
				useConfigure = false
				usesPerl = true
			} else {
				return domain.Forge{}, fmt.Errorf("configure script not found at %s (or Configure)", configurePath)
			}
		} else {
			configurePath = ConfigurePath
			useConfigure = false
			usesPerl = true
		}
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
	}

	r.touchAutotools(sourcePath)

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	var filter *utils.ErrorWarningFilter
	if !config.Verbose {
		stdout = io.Discard
		filter = utils.NewErrorWarningFilter(os.Stderr)
		stderr = filter
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
			if filter != nil {
				filter.Flush()
			}
			return domain.Forge{}, fmt.Errorf("autoreconf failed: %w", err)
		}
	}

	args := []string{fmt.Sprintf("--prefix=%s", prefix)}
	args = append(args, config.ConfigureFlags...)

	var configure *exec.Cmd
	if useConfigure {
		configure = exec.Command("./configure", args...)
	} else if isOpensslConfig {
		configure = exec.Command("./config", args...)
	} else if usesPerl {
		target := getOpenSSLConfigureTarget()
		perlArgs := []string{configurePath}
		perlArgs = append(perlArgs, target)
		perlArgs = append(perlArgs, args...)
		configure = exec.Command("perl", perlArgs...)
	} else {
		configure = exec.Command(configurePath, args...)
	}
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

	if filter != nil {
		filter.Flush()
	}

	if err := r.makeWithName(sourcePath, config.Jobs, env, config.Name, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, config.Jobs, env, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}
