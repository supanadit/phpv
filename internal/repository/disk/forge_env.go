package disk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) buildEnv(config domain.ForgeConfig) []string {
	env := os.Environ()

	buildToolsPath := filepath.Join(r.siloRepo.silo.Root, "build-tools")
	buildToolsBinPath := r.buildToolsBinPath(buildToolsPath)

	for i, v := range env {
		if after, ok := strings.CutPrefix(v, "PATH="); ok {
			env[i] = "PATH=" + buildToolsBinPath + ":" + after
			break
		}
	}

	systemPkgConfigPaths := utils.GetSystemPkgConfigPaths()
	var allPkgConfigPaths []string
	if len(config.PkgConfigPaths) > 0 {
		allPkgConfigPaths = append(allPkgConfigPaths, config.PkgConfigPaths...)
	}
	allPkgConfigPaths = append(allPkgConfigPaths, systemPkgConfigPaths...)
	for i, v := range env {
		if after, ok := strings.CutPrefix(v, "PKG_CONFIG_PATH="); ok {
			allPkgConfigPaths = append(allPkgConfigPaths, strings.Split(after, ":")...)
			env[i] = "PKG_CONFIG_PATH=" + strings.Join(allPkgConfigPaths, ":")
			break
		}
	}
	if !hasEnvVar(env, "PKG_CONFIG_PATH") {
		env = append(env, "PKG_CONFIG_PATH="+strings.Join(allPkgConfigPaths, ":"))
	}

	if len(config.CPPFLAGS) > 0 {
		env = append(env, "CPPFLAGS="+strings.Join(config.CPPFLAGS, " "))
	}
	if len(config.LDFLAGS) > 0 {
		env = append(env, "LDFLAGS="+strings.Join(config.LDFLAGS, " "))
	}
	if len(config.LD_LIBRARY_PATH) > 0 {
		env = append(env, "LD_LIBRARY_PATH="+strings.Join(config.LD_LIBRARY_PATH, ":"))
	}
	if config.CC != "" {
		env = append(env, "CC="+config.CC)
	}
	if len(config.CFLAGS) > 0 {
		env = append(env, "CFLAGS="+strings.Join(config.CFLAGS, " "))
	}
	if config.CXX != "" {
		env = append(env, "CXX="+config.CXX)
	}
	if len(config.CXXFLAGS) > 0 {
		env = append(env, "CXXFLAGS="+strings.Join(config.CXXFLAGS, " "))
	}
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}

	return env
}

func hasEnvVar(env []string, prefix string) bool {
	for _, v := range env {
		if _, found := strings.CutPrefix(v, prefix+"="); found {
			return true
		}
	}
	return false
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
			versionDir := filepath.Join(pkgPath, vEntry.Name())

			binPath := filepath.Join(versionDir, "bin")
			if exists, _ := afero.DirExists(r.fs, binPath); exists {
				binPaths = append(binPaths, binPath)
				continue
			}

			if r.hasExecutable(versionDir, entry.Name()) {
				binPaths = append(binPaths, versionDir)
				continue
			}

			nestedBin := filepath.Join(versionDir, entry.Name(), "bin")
			if exists, _ := afero.DirExists(r.fs, nestedBin); exists {
				binPaths = append(binPaths, nestedBin)
			}
		}
	}

	return strings.Join(binPaths, ":")
}

func (r *ForgeRepository) hasExecutable(dir, name string) bool {
	exePath := filepath.Join(dir, name)
	if exists, _ := afero.Exists(r.fs, exePath); exists {
		return true
	}
	return false
}

func (r *ForgeRepository) chmodBuildScripts(sourcePath string) {
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "build")).Run()
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "ext")).Run()
}

func (r *ForgeRepository) touchAutotools(sourcePath string) {
	autotoolsFiles := []string{
		"aclocal.m4",
		"Makefile.in",
		"configure",
		"config.h.in",
	}
	for _, f := range autotoolsFiles {
		file := filepath.Join(sourcePath, f)
		if _, err := os.Stat(file); err == nil {
			os.Chtimes(file, time.Now(), time.Now())
		}
	}
}
