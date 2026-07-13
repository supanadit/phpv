package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ForgeRepository is a disk-backed implementation of forge.ForgeRepository.
// It builds packages from source using per-package strategy detection.
type ForgeRepository struct{}

func NewForgeRepository() *ForgeRepository {
	return &ForgeRepository{}
}

// Build compiles the package from sourceDir. extraEnv provides additional
// environment variables (e.g., PATH with build tool bins).
func (f *ForgeRepository) Build(name string, version string, sourceDir string, extraEnv []string) (buildDir string, env map[string]string, err error) {
	srcPath := findSourceDir(sourceDir, name, version)
	if srcPath == "" {
		return "", nil, fmt.Errorf("could not find source directory in %s", sourceDir)
	}

	chmodBuildScripts(srcPath)
	touchAutotools(srcPath)

	// Per-package strategy detection (mirrors legacy).
	strategy := detectStrategy(name)

	// Build env vars that downstream packages need.
	env = map[string]string{
		"PKG_CONFIG_PATH": filepath.Join(filepath.Dir(srcPath), "lib", "pkgconfig"),
	}
	env["CPPFLAGS"] = "-I" + filepath.Join(filepath.Dir(srcPath), "include")
	env["LDFLAGS"] = "-L" + filepath.Join(filepath.Dir(srcPath), "lib") + " -Wl,-rpath," + filepath.Join(filepath.Dir(srcPath), "lib")

	switch strategy {
	case "cmake":
		return f.buildCMake(name, version, srcPath, extraEnv)
	case "makeonly":
		return f.buildMakeOnly(name, version, srcPath, extraEnv)
	case "configure":
		return f.buildConfigure(name, version, srcPath, extraEnv)
	case "autogen":
		return f.buildAutogen(name, version, srcPath, extraEnv)
	default:
		return "", nil, fmt.Errorf("unsupported build strategy: %s for %s", strategy, name)
	}
}

// Install installs a previously built package into prefix.
func (f *ForgeRepository) Install(name string, version string, buildDir string, prefix string) error {
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return fmt.Errorf("create prefix %s: %w", prefix, err)
	}

	installTarget := "install"
	if name == "openssl" || name == "ossl" {
		installTarget = "install_sw"
	}

	install := exec.Command("make", "-j4", installTarget)
	install.Dir = buildDir
	if out, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("make install %s: %w\n%s", name, err, out)
	}

	return nil
}

// buildCMake runs cmake + make.
func (f *ForgeRepository) buildCMake(name, version, srcPath string, extraEnv []string) (string, map[string]string, error) {
	buildDir := filepath.Join(srcPath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create build dir: %w", err)
	}

	cmake := exec.Command("cmake", srcPath,
		"-DCMAKE_INSTALL_PREFIX="+filepath.Dir(srcPath),
		"-DCMAKE_BUILD_TYPE=Release",
	)
	cmake.Dir = buildDir
	cmake.Env = mergeEnv(extraEnv)
	if out, err := cmake.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("cmake %s: %w\n%s", name, err, out)
	}

	make := exec.Command("make", "-j4")
	make.Dir = buildDir
	make.Env = mergeEnv(extraEnv)
	if out, err := make.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("make %s: %w\n%s", name, err, out)
	}

	return buildDir, forgeEnv(filepath.Dir(srcPath)), nil
}

// buildConfigure runs ./configure + make. Handles OpenSSL's Configure and config.
func (f *ForgeRepository) buildConfigure(name, version, srcPath string, extraEnv []string) (string, map[string]string, error) {
	prefix := filepath.Dir(srcPath)

	// For m4, run autoreconf first.
	if name == "m4" {
		if _, err := os.Stat(filepath.Join(srcPath, "configure")); os.IsNotExist(err) {
			autoreconf := exec.Command("autoreconf", "-fi")
			autoreconf.Dir = srcPath
			autoreconf.Env = mergeEnv(extraEnv)
			if out, err := autoreconf.CombinedOutput(); err != nil {
				return "", nil, fmt.Errorf("autoreconf %s: %w\n%s", name, err, out)
			}
		}
	}

	// For automake, touch all generated files to prevent regeneration.
	if name == "automake" {
		touchAllGeneratedFiles(srcPath)
	}

	// Determine which configure script to use.
	var configurePath string
	var usePerl bool
	var useConfig bool

	if _, err := os.Stat(filepath.Join(srcPath, "configure")); err == nil {
		configurePath = filepath.Join(srcPath, "configure")
	} else if name == "openssl" || name == "ossl" {
		if _, err := os.Stat(filepath.Join(srcPath, "config")); err == nil {
			configurePath = filepath.Join(srcPath, "config")
			useConfig = true
		} else if _, err := os.Stat(filepath.Join(srcPath, "Configure")); err == nil {
			configurePath = filepath.Join(srcPath, "Configure")
			usePerl = true
		} else {
			return "", nil, fmt.Errorf("no configure script found for %s", name)
		}
	} else {
		return "", nil, fmt.Errorf("configure script not found for %s", name)
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return "", nil, fmt.Errorf("chmod configure: %w", err)
	}

	args := []string{"--prefix=" + prefix}

	var configure *exec.Cmd
	if usePerl {
		perlArgs := []string{configurePath, "linux-x86_64"}
		for _, a := range args {
			perlArgs = append(perlArgs, a)
		}
		configure = exec.Command("perl", perlArgs...)
		configure.Dir = srcPath
		configure.Env = mergeEnv(extraEnv)
	} else if useConfig {
		configure = exec.Command("./config", args...)
		configure.Dir = srcPath
		configure.Env = mergeEnv(extraEnv)
	} else {
		configure = exec.Command("./configure", args...)
		configure.Dir = srcPath
		configure.Env = mergeEnv(extraEnv)
	}

	if out, err := configure.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("configure %s: %w\n%s", name, err, out)
	}

	if err := runMake(name, srcPath, extraEnv); err != nil {
		return "", nil, err
	}

	return srcPath, forgeEnv(prefix), nil
}

// buildAutogen runs autoreconf then configure + make.
func (f *ForgeRepository) buildAutogen(name, version, srcPath string, extraEnv []string) (string, map[string]string, error) {
	prefix := filepath.Dir(srcPath)

	autoreconf := exec.Command("autoreconf", "-fi")
	autoreconf.Dir = srcPath
	autoreconf.Env = mergeEnv(extraEnv)
	if out, err := autoreconf.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("autoreconf %s: %w\n%s", name, err, out)
	}

	configure := exec.Command("./configure", "--prefix="+prefix)
	configure.Dir = srcPath
	configure.Env = mergeEnv(extraEnv)
	if out, err := configure.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("configure %s: %w\n%s", name, err, out)
	}

	if err := runMake(name, srcPath, extraEnv); err != nil {
		return "", nil, err
	}

	return srcPath, forgeEnv(prefix), nil
}

// buildMakeOnly runs make (no configure step).
func (f *ForgeRepository) buildMakeOnly(name, version, srcPath string, extraEnv []string) (string, map[string]string, error) {
	prefix := filepath.Dir(srcPath)

	if err := runMake(name, srcPath, extraEnv); err != nil {
		return "", nil, err
	}

	return srcPath, forgeEnv(prefix), nil
}

// runMake runs make with per-package settings.
func runMake(name, srcPath string, extraEnv []string) error {
	jobs := 4
	args := []string{fmt.Sprintf("-j%d", jobs)}

	if name == "automake" || name == "autoconf" || name == "libtool" {
		args = []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}
	}

	make := exec.Command("make", args...)
	make.Dir = srcPath
	make.Env = mergeEnv(extraEnv)
	if out, err := make.CombinedOutput(); err != nil {
		return fmt.Errorf("make %s: %w\n%s", name, err, out)
	}
	return nil
}

// mergeEnv merges extraEnv into the current process environment.
func mergeEnv(extraEnv []string) []string {
	if len(extraEnv) == 0 {
		return nil
	}
	env := os.Environ()
	for _, e := range extraEnv {
		env = append(env, e)
	}
	return env
}

// detectStrategy returns the build strategy for a package name.
func detectStrategy(name string) string {
	switch name {
	case "zlib", "openssl", "ossl", "libtool", "libxml2", "php", "m4", "automake", "autoconf":
		return "configure"
	case "cmake":
		return "cmake"
	case "bison", "flex", "perl":
		return "makeonly"
	default:
		return "configure"
	}
}

// chmodBuildScripts makes build scripts executable.
func chmodBuildScripts(sourcePath string) {
	for _, dir := range []string{"build", "ext", "build-aux", "lib", "scripts"} {
		exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, dir)).Run()
	}
	for _, script := range []string{"install-sh", "depcomp", "ylwrap", "compile",
		"config.guess", "config.sub", "configure", "missing", "mkinstalldirs"} {
		p := filepath.Join(sourcePath, script)
		if _, err := os.Stat(p); err == nil {
			os.Chmod(p, 0o755)
		}
	}
}

// touchAutotools touches autotools files to prevent regeneration.
func touchAutotools(sourcePath string) {
	for _, f := range []string{"aclocal.m4", "Makefile.in", "configure", "config.h.in"} {
		p := filepath.Join(sourcePath, f)
		if _, err := os.Stat(p); err == nil {
			os.Chtimes(p, time.Now(), time.Now())
		}
	}
}

// touchAllGeneratedFiles touches .am then .in files to prevent make regeneration.
func touchAllGeneratedFiles(sourcePath string) {
	now := time.Now()
	filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".am") {
			os.Chtimes(path, now, now)
		}
		return nil
	})
	time.Sleep(time.Second)
	now = time.Now()
	filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".in") {
			os.Chtimes(path, now, now)
		}
		return nil
	})
}

// forgeEnv builds the environment variables for downstream packages.
func forgeEnv(prefix string) map[string]string {
	env := map[string]string{
		"PKG_CONFIG_PATH": filepath.Join(prefix, "lib", "pkgconfig"),
	}
	env["CPPFLAGS"] = "-I" + filepath.Join(prefix, "include")
	env["LDFLAGS"] = "-L" + filepath.Join(prefix, "lib") + " -Wl,-rpath," + filepath.Join(prefix, "lib")
	return env
}

// findSourceDir locates the actual source directory inside the extracted dir.
func findSourceDir(extractDir, name, version string) string {
	if hasBuildFile(extractDir) {
		return extractDir
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return ""
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 1 {
		candidate := filepath.Join(extractDir, dirs[0])
		if hasBuildFile(candidate) {
			return candidate
		}
	}
	for _, d := range dirs {
		candidate := filepath.Join(extractDir, d)
		if hasBuildFile(candidate) {
			return candidate
		}
	}
	return ""
}

func hasBuildFile(dir string) bool {
	return fileExists(filepath.Join(dir, "configure")) ||
		fileExists(filepath.Join(dir, "Configure")) ||
		fileExists(filepath.Join(dir, "CMakeLists.txt")) ||
		fileExists(filepath.Join(dir, "Makefile")) ||
		fileExists(filepath.Join(dir, "makefile"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
