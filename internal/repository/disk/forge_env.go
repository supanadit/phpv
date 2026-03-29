package disk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
)

func (r *ForgeRepository) buildEnv(config domain.ForgeConfig) []string {
	env := os.Environ()

	buildToolsPath := filepath.Join(r.siloRepo.silo.Root, "build-tools")
	buildToolsBinPath := r.buildToolsBinPath(buildToolsPath)

	for i, v := range env {
		if strings.HasPrefix(v, "PATH=") {
			env[i] = "PATH=" + buildToolsBinPath + ":" + strings.TrimPrefix(v, "PATH=")
			break
		}
	}

	for _, v := range config.CPPFLAGS {
		env = append(env, "CPPFLAGS="+v)
	}
	for _, v := range config.LDFLAGS {
		env = append(env, "LDFLAGS="+v)
	}
	if len(config.LD_LIBRARY_PATH) > 0 {
		env = append(env, "LD_LIBRARY_PATH="+strings.Join(config.LD_LIBRARY_PATH, ":"))
	}
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}

	return env
}

func (r *ForgeRepository) buildToolsBinPath(buildToolsPath string) string {
	var binPaths []string

	entries, err := afero.ReadDir(r.fs, buildToolsPath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgPath := filepath.Join(buildToolsPath, entry.Name())
		versionEntries, err := afero.ReadDir(r.fs, pkgPath)
		if err != nil {
			continue
		}
		for _, vEntry := range versionEntries {
			if !vEntry.IsDir() {
				continue
			}
			binPath := filepath.Join(pkgPath, vEntry.Name(), "bin")
			if exists, _ := afero.DirExists(r.fs, binPath); exists {
				binPaths = append(binPaths, binPath)
			}
		}
	}

	return strings.Join(binPaths, ":")
}

func (r *ForgeRepository) chmodBuildScripts(sourcePath string) {
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "build")).Run()
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "ext")).Run()
}
