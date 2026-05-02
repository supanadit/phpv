package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func getOpenSSLConfigureTarget() string {
	return utils.GetOpenSSLConfigureTarget()
}

func getConfigureHostTriple() string {
	return utils.GetConfigureHostTriple()
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

func (r *ForgeRepository) isAutotoolsConfigure(sourcePath string) bool {
	files := []string{"configure.ac", "configure.in", "aclocal.m4"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(sourcePath, f)); err == nil {
			return true
		}
	}
	return false
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
		} else if found := r.findConfigureInSubdir(sourcePath, "configure"); found != "" {
			configurePath = found
			useConfigure = false
		} else {
			return domain.Forge{}, fmt.Errorf("configure script not found at %s (or Configure)", configurePath)
		}
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
	}

	r.touchAutotools(sourcePath)

	// For automake, touch all generated files to prevent make from trying
	// to regenerate them (which would require automake itself to be installed).
	if config.Name == "automake" {
		r.touchAllGeneratedFiles(sourcePath)
	}

	ctx := utils.NewExecContext(config.Verbose)
	jobs := utils.GetJobs(config.Jobs)

	if config.Name == "m4" {
		if _, err := os.Stat(configurePath); os.IsNotExist(err) {
			autoreconf := ctx.Command("autoreconf", "-fi")
			autoreconf.Dir = sourcePath
			autoreconf.Env = env

			if err := ctx.Run(autoreconf); err != nil {
				return domain.Forge{}, fmt.Errorf("autoreconf failed: %w", err)
			}
		}
	}

	args := []string{fmt.Sprintf("--prefix=%s", prefix)}

	args = append(args, config.ConfigureFlags...)

	var configure *exec.Cmd
	if useConfigure {
		if config.CC != "" && strings.Contains(config.CC, "zig") && r.isAutotoolsConfigure(sourcePath) {
			hostTriple := getConfigureHostTriple()
			args = append(args, "--build="+hostTriple, "--host="+hostTriple)
		}
		configure = ctx.Command("./configure", args...)
		configure.Dir = sourcePath
		configure.Env = env
	} else if isOpensslConfig {
		configure = ctx.Command("./config", args...)
		configure.Dir = sourcePath
		configure.Env = env
	} else if usesPerl {
		target := getOpenSSLConfigureTarget()
		perlArgs := []string{configurePath}
		perlArgs = append(perlArgs, target)
		perlArgs = append(perlArgs, args...)

		configure = ctx.Command("perl", perlArgs...)
		configure.Dir = sourcePath
		baseEnv := env
		if len(config.CFLAGS) > 0 {
			baseEnv = append(baseEnv, "CFLAGS="+strings.Join(config.CFLAGS, " "))
		}
		if len(config.CXXFLAGS) > 0 {
			baseEnv = append(baseEnv, "CXXFLAGS="+strings.Join(config.CXXFLAGS, " "))
		}
		if len(config.LDFLAGS) > 0 {
			baseEnv = append(baseEnv, "LDFLAGS="+strings.Join(config.LDFLAGS, " "))
		}
		configure.Env = baseEnv
	} else {
		configure = ctx.Command(configurePath, args...)
		configure.Dir = sourcePath
		configure.Env = env
	}

	if err := ctx.Run(configure); err != nil {
		return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
	}

	if err := r.makeWithName(sourcePath, jobs, env, config.Name, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, jobs, env, config.Verbose, config.Name); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}
