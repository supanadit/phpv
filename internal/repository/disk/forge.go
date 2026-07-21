package disk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type ForgeRepository struct{}

func NewForgeRepository() *ForgeRepository {
	return &ForgeRepository{}
}

func (f *ForgeRepository) Build(ctx context.Context, name string, version string, sourceDir string, extraEnv []string, extraConfigureFlags []string, installPrefix string, verbose bool, jobs int) (buildDir string, env map[string]string, err error) {
	srcPath := findSourceDir(sourceDir, name, version)
	if srcPath == "" {
		return "", nil, fmt.Errorf("could not find source directory in %s", sourceDir)
	}

	chmodBuildScripts(srcPath)
	touchAutotools(srcPath)

	strategy := detectStrategy(name)

	prefix := installPrefix
	if prefix == "" {
		prefix = filepath.Dir(srcPath)
	}
	env = map[string]string{
		"PKG_CONFIG_PATH": filepath.Join(prefix, "lib", "pkgconfig"),
	}
	env["CPPFLAGS"] = "-I" + filepath.Join(prefix, "include")
	env["LDFLAGS"] = "-L" + filepath.Join(prefix, "lib") + " -Wl,-rpath," + filepath.Join(prefix, "lib")

	switch strategy {
	case "cmake":
		return f.buildCMake(ctx, name, version, srcPath, extraEnv, extraConfigureFlags, prefix, verbose, jobs)
	case "makeonly":
		return f.buildMakeOnly(ctx, name, version, srcPath, extraEnv, verbose, jobs)
	case "configure":
		return f.buildConfigure(ctx, name, version, srcPath, extraEnv, extraConfigureFlags, prefix, verbose, jobs)
	case "autogen":
		return f.buildAutogen(ctx, name, version, srcPath, extraEnv, extraConfigureFlags, prefix, verbose, jobs)
	default:
		return "", nil, fmt.Errorf("unsupported build strategy: %s for %s", strategy, name)
	}
}

func (f *ForgeRepository) Install(ctx context.Context, name string, version string, buildDir string, prefix string, verbose bool, jobs int) error {
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return fmt.Errorf("create prefix %s: %w", prefix, err)
	}

	installTarget := "install"
	if name == "openssl" || name == "ossl" {
		installTarget = "install_sw"
	}

	install := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs), installTarget)
	install.Dir = buildDir
	install.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if verbose {
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("make install %s: %w", name, err)
		}
	} else {
		if out, err := install.CombinedOutput(); err != nil {
			return fmt.Errorf("make install %s: %w\n%s", name, err, out)
		}
	}

	return nil
}

func runCmd(ctx context.Context, cmd *exec.Cmd, verbose bool) ([]byte, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	return cmd.CombinedOutput()
}

func (f *ForgeRepository) buildCMake(ctx context.Context, name, version, srcPath string, extraEnv []string, extraFlags []string, prefix string, verbose bool, jobs int) (string, map[string]string, error) {
	buildDir := filepath.Join(srcPath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create build dir: %w", err)
	}

	cmakeArgs := []string{
		srcPath,
		"-DCMAKE_INSTALL_PREFIX=" + prefix,
		"-DCMAKE_BUILD_TYPE=Release",
	}
	cmakeArgs = append(cmakeArgs, extraFlags...)

	cmake := exec.CommandContext(ctx, "cmake", cmakeArgs...)
	cmake.Dir = buildDir
	cmake.Env = mergeEnv(extraEnv)
	if out, err := runCmd(ctx, cmake, verbose); err != nil {
		return "", nil, fmt.Errorf("cmake %s: %w\n%s", name, err, out)
	}

	make := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	make.Dir = buildDir
	make.Env = mergeEnv(extraEnv)
	if out, err := runCmd(ctx, make, verbose); err != nil {
		return "", nil, fmt.Errorf("make %s: %w\n%s", name, err, out)
	}

	return buildDir, forgeEnv(prefix), nil
}

func (f *ForgeRepository) buildConfigure(ctx context.Context, name, version, srcPath string, extraEnv []string, extraFlags []string, prefix string, verbose bool, jobs int) (string, map[string]string, error) {
	if name == "m4" {
		if _, err := os.Stat(filepath.Join(srcPath, "configure")); os.IsNotExist(err) {
			autoreconf := exec.CommandContext(ctx, "autoreconf", "-fi")
			autoreconf.Dir = srcPath
			autoreconf.Env = mergeEnv(extraEnv)
			if out, err := runCmd(ctx, autoreconf, verbose); err != nil {
				return "", nil, fmt.Errorf("autoreconf %s: %w\n%s", name, err, out)
			}
		}
	}

	if name == "automake" {
		touchAllGeneratedFiles(srcPath)
	}

	var configurePath string
	var usePerl bool
	var useConfig bool

	if name == "openssl" || name == "ossl" {
		if _, err := os.Stat(filepath.Join(srcPath, "Configure")); err == nil {
			configurePath = filepath.Join(srcPath, "Configure")
			usePerl = true
		} else if _, err := os.Stat(filepath.Join(srcPath, "config")); err == nil {
			configurePath = filepath.Join(srcPath, "config")
			useConfig = true
		} else {
			return "", nil, fmt.Errorf("no configure script found for %s", name)
		}
	} else if _, err := os.Stat(filepath.Join(srcPath, "configure")); err == nil {
		configurePath = filepath.Join(srcPath, "configure")
	} else {
		return "", nil, fmt.Errorf("configure script not found for %s", name)
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return "", nil, fmt.Errorf("chmod configure: %w", err)
	}

	args := []string{"--prefix=" + prefix}
	args = append(args, extraFlags...)

	var configure *exec.Cmd
	if usePerl {
		opensslTarget := opensslTarget()
		perlArgs := []string{configurePath, opensslTarget}
		for _, a := range args {
			perlArgs = append(perlArgs, a)
		}
		configure = exec.CommandContext(ctx, "perl", perlArgs...)
		configure.Dir = srcPath
		configure.Env = mergeEnv(extraEnv)
	} else if useConfig {
		configure = exec.CommandContext(ctx, "./config", args...)
		configure.Dir = srcPath
		configure.Env = mergeEnv(extraEnv)
	} else {
		configure = exec.CommandContext(ctx, "./configure", args...)
		configure.Dir = srcPath
		configure.Env = mergeEnv(extraEnv)
	}

	if out, err := runCmd(ctx, configure, verbose); err != nil {
		return "", nil, fmt.Errorf("configure %s: %w\n%s", name, err, out)
	}

	if err := runMake(ctx, name, srcPath, extraEnv, verbose, jobs); err != nil {
		return "", nil, err
	}

	return srcPath, forgeEnv(prefix), nil
}

func (f *ForgeRepository) buildAutogen(ctx context.Context, name, version, srcPath string, extraEnv []string, extraFlags []string, prefix string, verbose bool, jobs int) (string, map[string]string, error) {
	autoreconf := exec.CommandContext(ctx, "autoreconf", "-fi")
	autoreconf.Dir = srcPath
	autoreconf.Env = mergeEnv(extraEnv)
	if out, err := runCmd(ctx, autoreconf, verbose); err != nil {
		return "", nil, fmt.Errorf("autoreconf %s: %w\n%s", name, err, out)
	}

	args := []string{"--prefix=" + prefix}
	args = append(args, extraFlags...)

	configure := exec.CommandContext(ctx, "./configure", args...)
	configure.Dir = srcPath
	configure.Env = mergeEnv(extraEnv)
	if out, err := runCmd(ctx, configure, verbose); err != nil {
		return "", nil, fmt.Errorf("configure %s: %w\n%s", name, err, out)
	}

	if err := runMake(ctx, name, srcPath, extraEnv, verbose, jobs); err != nil {
		return "", nil, err
	}

	return srcPath, forgeEnv(prefix), nil
}

func (f *ForgeRepository) buildMakeOnly(ctx context.Context, name, version, srcPath string, extraEnv []string, verbose bool, jobs int) (string, map[string]string, error) {
	prefix := filepath.Dir(srcPath)

	if err := runMake(ctx, name, srcPath, extraEnv, verbose, jobs); err != nil {
		return "", nil, err
	}

	return srcPath, forgeEnv(prefix), nil
}

func runMake(ctx context.Context, name, srcPath string, extraEnv []string, verbose bool, jobs int) error {
	if jobs < 1 {
		jobs = 4
	}
	args := []string{fmt.Sprintf("-j%d", jobs)}

	if name == "automake" || name == "autoconf" || name == "libtool" {
		args = []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}
	}

	make := exec.CommandContext(ctx, "make", args...)
	make.Dir = srcPath
	make.Env = mergeEnv(extraEnv)
	if out, err := runCmd(ctx, make, verbose); err != nil {
		return fmt.Errorf("make %s: %w\n%s", name, err, out)
	}
	return nil
}

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

func touchAutotools(sourcePath string) {
	for _, f := range []string{"aclocal.m4", "Makefile.in", "configure", "config.h.in"} {
		p := filepath.Join(sourcePath, f)
		if _, err := os.Stat(p); err == nil {
			os.Chtimes(p, time.Now(), time.Now())
		}
	}
}

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

func forgeEnv(prefix string) map[string]string {
	env := map[string]string{
		"PKG_CONFIG_PATH": filepath.Join(prefix, "lib", "pkgconfig"),
	}
	env["CPPFLAGS"] = "-I" + filepath.Join(prefix, "include")
	env["LDFLAGS"] = "-L" + filepath.Join(prefix, "lib") + " -Wl,-rpath," + filepath.Join(prefix, "lib")
	return env
}

func opensslTarget() string {
	if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			return "darwin64-arm64-cc"
		}
		return "darwin64-x86_64-cc"
	}
	return "linux-x86_64"
}

func findSourceDir(extractDir, name, version string) string {
	if hasBuildFile(extractDir) {
		return extractDir
	}
	// ICU tarballs extract to icu/source/ (nested one level deeper).
	if name == "icu" {
		icuSource := filepath.Join(extractDir, "icu", "source")
		if hasBuildFile(icuSource) {
			return icuSource
		}
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
