package disk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

	pkgConfigPaths := []string{
		"/usr/lib/pkgconfig",
		"/usr/lib/x86_64-linux-gnu/pkgconfig",
		"/usr/share/pkgconfig",
		"/usr/local/lib/pkgconfig",
		"/usr/local/share/pkgconfig",
	}
	for i, v := range env {
		if strings.HasPrefix(v, "PKG_CONFIG_PATH=") {
			existing := strings.TrimPrefix(v, "PKG_CONFIG_PATH=")
			pkgConfigPaths = append(pkgConfigPaths, strings.Split(existing, ":")...)
			env[i] = "PKG_CONFIG_PATH=" + strings.Join(pkgConfigPaths, ":")
			break
		}
	}
	if !hasEnvVar(env, "PKG_CONFIG_PATH") {
		env = append(env, "PKG_CONFIG_PATH="+strings.Join(pkgConfigPaths, ":"))
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
		if strings.HasPrefix(v, prefix+"=") {
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
