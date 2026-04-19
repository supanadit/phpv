package disk

import (
	"fmt"
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

	if config.PHPVersion != "" {
		depRootPath := filepath.Join(r.siloRepo.silo.Root, "versions", config.PHPVersion, "dependency")
		depBinPaths := r.discoverDependencyBinPaths(depRootPath)
		if depBinPaths != "" {
			buildToolsBinPath = depBinPaths + ":" + buildToolsBinPath
		}
	}

// When using Zig as CC, create wrapper scripts for ar, ranlib, nm, and ld
	// so that configure scripts can discover them in PATH.
	if strings.Contains(config.CC, "zig") {
		zigBinary := strings.Split(config.CC, " ")[0] // extract zig binary path from "zig cc -target ..."
		wrapperDir := r.ensureZigToolWrappers(zigBinary)
		if wrapperDir != "" {
			buildToolsBinPath = wrapperDir + ":" + buildToolsBinPath
			env = append(env, "AR="+filepath.Join(wrapperDir, "ar"))
			env = append(env, "RANLIB="+filepath.Join(wrapperDir, "ranlib"))
			env = append(env, "NM="+filepath.Join(wrapperDir, "nm"))
		}
	}

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

	// Set autotools environment variables to use bundled versions
	r.setAutotoolsEnv(config, buildToolsBinPath, &env)

	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}

	return env
}

func (r *ForgeRepository) setAutotoolsEnv(config domain.ForgeConfig, buildToolsBinPath string, env *[]string) {
	tools := map[string]string{
		"AUTOCONF":        "autoconf",
		"AUTOMAKE":        "automake",
		"ACLOCAL":         "aclocal",
		"ACLOCAL_AMFLAGS": "",
		"LIBTOOLIZE":      "libtoolize",
		"AUTOHEADER":      "autoheader",
		"AUTOM4TE":        "autom4te",
		"AUTORECONF":      "autoreconf",
	}

	for envVar, toolName := range tools {
		if toolName == "" {
			*env = append(*env, envVar+"=")
			continue
		}
		if path := r.findToolInPath(toolName, buildToolsBinPath); path != "" {
			*env = append(*env, envVar+"="+path)
		}
	}
}

func (r *ForgeRepository) findToolInPath(toolName, path string) string {
	parts := strings.Split(path, ":")
	for _, dir := range parts {
		if dir == "" {
			continue
		}
		toolPath := filepath.Join(dir, toolName)
		if _, err := os.Stat(toolPath); err == nil {
			return toolPath
		}
	}
	if fullPath, err := exec.LookPath(toolName); err == nil {
		return fullPath
	}
	return ""
}

// ensureZigToolWrappers creates wrapper scripts for ar, ranlib, nm, and ld
// in a directory next to the zig binary, so that GNU autotools configure scripts
// can find them in PATH. Zig provides these as subcommands (e.g. "zig ar")
// but configure expects standalone executables.
// Returns the directory containing the wrapper scripts.
func (r *ForgeRepository) ensureZigToolWrappers(zigBinary string) string {
	wrapperDir := filepath.Join(filepath.Dir(zigBinary), "wrappers")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		return ""
	}

	// Tools that zig provides as subcommands
	zigTools := map[string]string{
		"ar":     "ar",
		"ranlib": "ranlib",
	}

	for toolName, zigSubcmd := range zigTools {
		wrapperPath := filepath.Join(wrapperDir, toolName)
		if _, err := os.Stat(wrapperPath); err == nil {
			continue // already exists
		}
		script := fmt.Sprintf("#!/bin/sh\nexec \"%s\" %s \"$@\"\n", zigBinary, zigSubcmd)
		if err := os.WriteFile(wrapperPath, []byte(script), 0o755); err != nil {
			continue
		}
	}

	// For 'nm', zig doesn't provide a subcommand so fall back to system nm
	nmWrapper := filepath.Join(wrapperDir, "nm")
	if _, err := os.Stat(nmWrapper); os.IsNotExist(err) {
		if systemNm, err := exec.LookPath("nm"); err == nil {
			script := fmt.Sprintf("#!/bin/sh\nexec \"%s\" \"$@\"\n", systemNm)
			os.WriteFile(nmWrapper, []byte(script), 0o755)
		}
	}

	// For 'ld', prefer the system linker since zig's internal linker isn't
	// directly exposed as a standalone 'ld' command. The system 'ld' is
	// typically provided by binutils and is needed by configure scripts.
	ldWrapper := filepath.Join(wrapperDir, "ld")
	if _, err := os.Stat(ldWrapper); os.IsNotExist(err) {
		// Try to find system ld
		if systemLd, err := exec.LookPath("ld"); err == nil {
			script := fmt.Sprintf("#!/bin/sh\nexec \"%s\" \"$@\"\n", systemLd)
			os.WriteFile(ldWrapper, []byte(script), 0o755)
		} else {
			// Fallback: use zig cc as linker (zig can link via cc)
			script := fmt.Sprintf("#!/bin/sh\nexec \"%s\" cc \"$@\"\n", zigBinary)
			os.WriteFile(ldWrapper, []byte(script), 0o755)
		}
	}

	return wrapperDir
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
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "build-aux")).Run()

	autotoolsScripts := []string{
		"install-sh",
		"depcomp",
		"ylwrap",
		"compile",
		"config.guess",
		"config.sub",
		"configure",
		"missing",
		"mkinstalldirs",
	}
	for _, script := range autotoolsScripts {
		scriptPath := filepath.Join(sourcePath, script)
		if _, err := os.Stat(scriptPath); err == nil {
			os.Chmod(scriptPath, 0o755)
		}
	}
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
