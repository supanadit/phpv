package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) buildAutogen(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	ctx := utils.NewExecContext(config.Verbose)
	jobs := utils.GetJobs(config.Jobs)

	buildToolsBinPath := r.getBuildToolsBinPath(config)

	bundledAutoreconf := r.findToolInPath("autoreconf", buildToolsBinPath)
	if bundledAutoreconf != "" {
		autoreconfCmd := ctx.Command(bundledAutoreconf, "-fi")
		autoreconfCmd.Dir = sourcePath
		autoreconfCmd.Env = env

		if err := ctx.Run(autoreconfCmd); err != nil {
			return domain.Forge{}, fmt.Errorf("autoreconf failed: %w", err)
		}
	}

	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); err == nil {
		if err := os.Chmod(configurePath, 0o755); err != nil {
			return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
		}

		args := []string{fmt.Sprintf("--prefix=%s", prefix)}
		args = append(args, config.ConfigureFlags...)

		configure := ctx.Command("./configure", args...)
		configure.Dir = sourcePath
		configure.Env = env

		if err := ctx.Run(configure); err != nil {
			return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
		}
	}

	if err := r.makeWithName(sourcePath, jobs, env, config.Name, config.Verbose); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, jobs, env, config.Verbose, config.Name); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}

func (r *ForgeRepository) getBuildToolsBinPath(config domain.ForgeConfig) string {
	buildToolsPath := filepath.Join(r.siloRepo.silo.Root, "build-tools")
	binPath := r.buildToolsBinPath(buildToolsPath)

	if config.PHPVersion != "" {
		depRootPath := filepath.Join(r.siloRepo.silo.Root, "versions", config.PHPVersion, "dependency")
		depBinPaths := r.discoverDependencyBinPaths(depRootPath)
		if depBinPaths != "" {
			binPath = depBinPaths + ":" + binPath
		}
	}

	return binPath
}

func (r *ForgeRepository) discoverDependencyBinPaths(depRoot string) string {
	var binPaths []string

	entries, err := afero.ReadDir(r.fs, depRoot)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgDir := filepath.Join(depRoot, entry.Name())
		versionEntries, err := afero.ReadDir(r.fs, pkgDir)
		if err != nil {
			continue
		}
		for _, vEntry := range versionEntries {
			if !vEntry.IsDir() {
				continue
			}
			verDir := filepath.Join(pkgDir, vEntry.Name())
			binPath := filepath.Join(verDir, "bin")
			if exists, _ := afero.DirExists(r.fs, binPath); exists {
				binPaths = append(binPaths, binPath)
			}
		}
	}

	return strings.Join(binPaths, ":")
}
