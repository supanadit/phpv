package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ForgeRepository is a disk-backed implementation of forge.ForgeRepository.
// It builds packages from source using configure+make or cmake strategies
// and installs them into a prefix directory.
type ForgeRepository struct{}

func NewForgeRepository() *ForgeRepository {
	return &ForgeRepository{}
}

// Build compiles the package from sourceDir. It auto-detects the build
// strategy and returns the build directory and environment variables
// that downstream packages need to link against this dependency.
func (f *ForgeRepository) Build(name string, version string, sourceDir string, phpVersion string) (buildDir string, env map[string]string, err error) {
	srcPath := findSourceDir(sourceDir, name, version)
	if srcPath == "" {
		return "", nil, fmt.Errorf("could not find source directory in %s", sourceDir)
	}

	// Detect build strategy.
	switch {
	case fileExists(filepath.Join(srcPath, "CMakeLists.txt")):
		return f.buildCMake(name, version, srcPath)
	case fileExists(filepath.Join(srcPath, "configure")):
		return f.buildConfigure(name, version, srcPath)
	case fileExists(filepath.Join(srcPath, "autogen.sh")):
		return f.buildAutogen(name, version, srcPath)
	case fileExists(filepath.Join(srcPath, "Makefile")) || fileExists(filepath.Join(srcPath, "makefile")):
		return f.buildMakeOnly(name, version, srcPath)
	default:
		return "", nil, fmt.Errorf("no build system found in %s", srcPath)
	}
}

// Install installs a previously built package into prefix.
func (f *ForgeRepository) Install(name string, version string, buildDir string, prefix string) error {
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return fmt.Errorf("create prefix %s: %w", prefix, err)
	}

	install := exec.Command("make", "install")
	install.Dir = buildDir
	if out, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("make install %s: %w\n%s", name, err, out)
	}

	return nil
}

// buildCMake runs cmake + make. Returns the build directory.
func (f *ForgeRepository) buildCMake(name, version, srcPath string) (string, map[string]string, error) {
	buildDir := filepath.Join(srcPath, "build-phpv")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create build dir: %w", err)
	}

	cmake := exec.Command("cmake", srcPath,
		"-DCMAKE_INSTALL_PREFIX="+filepath.Dir(srcPath),
		"-DCMAKE_BUILD_TYPE=Release",
	)
	cmake.Dir = buildDir
	if out, err := cmake.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("cmake %s: %w\n%s", name, err, out)
	}

	make := exec.Command("make", "-j4")
	make.Dir = buildDir
	if out, err := make.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("make %s: %w\n%s", name, err, out)
	}

	return buildDir, forgeEnv(filepath.Dir(srcPath)), nil
}

// buildConfigure runs ./configure + make. Returns the source directory as build dir.
func (f *ForgeRepository) buildConfigure(name, version, srcPath string) (string, map[string]string, error) {
	configure := exec.Command("./configure", "--prefix="+filepath.Dir(srcPath))
	configure.Dir = srcPath
	if out, err := configure.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("configure %s: %w\n%s", name, err, out)
	}

	make := exec.Command("make", "-j4")
	make.Dir = srcPath
	if out, err := make.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("make %s: %w\n%s", name, err, out)
	}

	return srcPath, forgeEnv(filepath.Dir(srcPath)), nil
}

// buildAutogen runs ./autogen.sh then ./configure + make.
func (f *ForgeRepository) buildAutogen(name, version, srcPath string) (string, map[string]string, error) {
	autogen := exec.Command("./autogen.sh")
	autogen.Dir = srcPath
	if out, err := autogen.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("autogen %s: %w\n%s", name, err, out)
	}

	return f.buildConfigure(name, version, srcPath)
}

// buildMakeOnly runs make (no configure step). Returns the source directory.
func (f *ForgeRepository) buildMakeOnly(name, version, srcPath string) (string, map[string]string, error) {
	make := exec.Command("make", "-j4")
	make.Dir = srcPath
	if out, err := make.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("make %s: %w\n%s", name, err, out)
	}

	return srcPath, forgeEnv(filepath.Dir(srcPath)), nil
}

// findSourceDir locates the actual source directory inside the extracted dir.
// Archives typically extract to a single subdirectory (e.g., php-7.4.33/).
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
	return fileExists(filepath.Join(dir, "CMakeLists.txt")) ||
		fileExists(filepath.Join(dir, "configure")) ||
		fileExists(filepath.Join(dir, "autogen.sh")) ||
		fileExists(filepath.Join(dir, "Makefile")) ||
		fileExists(filepath.Join(dir, "makefile"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// forgeEnv builds the environment variables that downstream packages need
// to link against a dependency installed at prefix.
func forgeEnv(prefix string) map[string]string {
	env := map[string]string{
		"PKG_CONFIG_PATH": filepath.Join(prefix, "lib", "pkgconfig"),
	}
	env["CPPFLAGS"] = "-I" + filepath.Join(prefix, "include")
	env["LDFLAGS"] = "-L" + filepath.Join(prefix, "lib") + " -Wl,-rpath," + filepath.Join(prefix, "lib")
	return env
}
