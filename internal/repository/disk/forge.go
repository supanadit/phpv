package disk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/supanadit/phpv/internal/repository"

	"github.com/supanadit/phpv/domain"

	"github.com/supanadit/phpv/pkg/source"
)

// buildSpec is the driver's resolved build inputs; it keeps private build
// helpers on short signatures without changing the build steps themselves.
type buildSpec struct {
	name, version, srcPath string
	extraEnv, extraFlags   []string
	prefix                 string
	verbose                bool
	jobs                   int
}

type ForgeRepository struct{}

func NewForgeRepository() *ForgeRepository {
	return &ForgeRepository{}
}

func (f *ForgeRepository) Build(ctx context.Context, req domain.BuildRequest) (buildDir string, env map[string]string, err error) {
	srcPath := source.Find(req.SourceDir, req.Name)
	if srcPath == "" {
		return "", nil, fmt.Errorf("could not find source directory in %s", req.SourceDir)
	}

	repository.ChmodBuildScripts(srcPath)
	repository.TouchAutotools(srcPath)

	strategy := repository.DetectStrategy(req.Name)

	prefix := req.InstallPrefix
	if prefix == "" {
		prefix = filepath.Dir(srcPath)
	}
	env = map[string]string{
		"PKG_CONFIG_PATH": filepath.Join(prefix, "lib", "pkgconfig"),
	}
	env["CPPFLAGS"] = "-I" + filepath.Join(prefix, "include")
	env["LDFLAGS"] = "-L" + filepath.Join(prefix, "lib") + " -Wl,-rpath," + filepath.Join(prefix, "lib")

	spec := buildSpec{
		name: req.Name, version: req.Version, srcPath: srcPath,
		extraEnv: req.ExtraEnv, extraFlags: req.ExtraConfigureFlags,
		prefix: prefix, verbose: req.Verbose, jobs: req.Jobs,
	}
	switch strategy {
	case "cmake":
		return f.buildCMake(ctx, spec)
	case "makeonly":
		return f.buildMakeOnly(ctx, spec)
	case "configure":
		return f.buildConfigure(ctx, spec)
	case "autogen":
		return f.buildAutogen(ctx, spec)
	default:
		return "", nil, fmt.Errorf("unsupported build strategy: %s for %s", strategy, req.Name)
	}
}

func (f *ForgeRepository) Install(ctx context.Context, req domain.InstallRequest) error {
	name, buildDir, prefix, verbose := req.Name, req.BuildDir, req.Prefix, req.Verbose
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return fmt.Errorf("create prefix %s: %w", prefix, err)
	}

	installTarget := "install"
	if name == "openssl" || name == "ossl" {
		installTarget = "install_sw"
	}

	// Install targets are short but not parallel-safe: PHP <= 5.6 uses an
	// shtool mkdir -p wrapper whose check-then-mkdir races when install-cli
	// and install-cgi run concurrently, failing with "File exists".
	install := exec.CommandContext(ctx, "make", "-j1", installTarget)
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

func (f *ForgeRepository) buildCMake(ctx context.Context, spec buildSpec) (string, map[string]string, error) {
	name, srcPath, extraEnv, extraFlags, prefix, verbose, jobs := spec.name, spec.srcPath, spec.extraEnv, spec.extraFlags, spec.prefix, spec.verbose, spec.jobs

	buildDir := filepath.Join(srcPath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create build dir: %w", err)
	}

	// CMake caches the resolved compiler in CMakeCache.txt and ignores $CC
	// on re-runs. If that cached path no longer exists (e.g. the ccache
	// package was removed), every reconfigure fails with "is not a full
	// path to an existing compiler tool". A changed install prefix would
	// likewise silently keep the old one. Wipe the cache dir when it is
	// stale; cmake regenerates it in milliseconds.
	if err := repository.InvalidateStaleCMakeCache(buildDir, prefix); err != nil {
		return "", nil, fmt.Errorf("invalidate cmake cache %s: %w", name, err)
	}

	cmakeArgs := []string{
		srcPath,
		"-DCMAKE_INSTALL_PREFIX=" + prefix,
		"-DCMAKE_BUILD_TYPE=Release",
	}
	cmakeArgs = append(cmakeArgs, extraFlags...)

	cmake := exec.CommandContext(ctx, "cmake", cmakeArgs...)
	cmake.Dir = buildDir
	cmake.Env = repository.MergeEnv(extraEnv)
	if out, err := repository.RunCmd(ctx, cmake, verbose); err != nil {
		return "", nil, fmt.Errorf("cmake %s: %w\n%s", name, err, out)
	}

	make := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	make.Dir = buildDir
	make.Env = repository.MergeEnv(extraEnv)
	if out, err := repository.RunCmd(ctx, make, verbose); err != nil {
		return "", nil, fmt.Errorf("make %s: %w\n%s", name, err, out)
	}

	return buildDir, repository.ForgeEnv(prefix), nil
}

func (f *ForgeRepository) buildConfigure(ctx context.Context, spec buildSpec) (string, map[string]string, error) {
	name, srcPath, extraEnv, extraFlags, prefix, verbose := spec.name, spec.srcPath, spec.extraEnv, spec.extraFlags, spec.prefix, spec.verbose

	if name == "m4" {
		if _, err := os.Stat(filepath.Join(srcPath, "configure")); os.IsNotExist(err) {
			autoreconf := exec.CommandContext(ctx, "autoreconf", "-fi")
			autoreconf.Dir = srcPath
			autoreconf.Env = repository.MergeEnv(extraEnv)
			if out, err := repository.RunCmd(ctx, autoreconf, verbose); err != nil {
				return "", nil, fmt.Errorf("autoreconf %s: %w\n%s", name, err, out)
			}
		}
	}

	if name == "automake" {
		repository.TouchAllGeneratedFiles(srcPath)
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
		opensslTarget := repository.OpensslTarget()
		perlArgs := []string{configurePath, opensslTarget}
		for _, a := range args {
			perlArgs = append(perlArgs, a)
		}
		configure = exec.CommandContext(ctx, "perl", perlArgs...)
		configure.Dir = srcPath
		configure.Env = repository.MergeEnv(extraEnv)
	} else if useConfig {
		configure = exec.CommandContext(ctx, "./config", args...)
		configure.Dir = srcPath
		configure.Env = repository.MergeEnv(extraEnv)
	} else {
		configure = exec.CommandContext(ctx, "./configure", args...)
		configure.Dir = srcPath
		configure.Env = repository.MergeEnv(extraEnv)
	}

	if out, err := repository.RunCmd(ctx, configure, verbose); err != nil {
		return "", nil, fmt.Errorf("configure %s: %w\n%s", name, err, out)
	}

	if err := runMake(ctx, spec, srcPath); err != nil {
		return "", nil, err
	}

	return srcPath, repository.ForgeEnv(prefix), nil
}

func (f *ForgeRepository) buildAutogen(ctx context.Context, spec buildSpec) (string, map[string]string, error) {
	name, srcPath, extraEnv, extraFlags, prefix, verbose := spec.name, spec.srcPath, spec.extraEnv, spec.extraFlags, spec.prefix, spec.verbose

	autoreconf := exec.CommandContext(ctx, "autoreconf", "-fi")
	autoreconf.Dir = srcPath
	autoreconf.Env = repository.MergeEnv(extraEnv)
	if out, err := repository.RunCmd(ctx, autoreconf, verbose); err != nil {
		return "", nil, fmt.Errorf("autoreconf %s: %w\n%s", name, err, out)
	}

	args := []string{"--prefix=" + prefix}
	args = append(args, extraFlags...)

	configure := exec.CommandContext(ctx, "./configure", args...)
	configure.Dir = srcPath
	configure.Env = repository.MergeEnv(extraEnv)
	if out, err := repository.RunCmd(ctx, configure, verbose); err != nil {
		return "", nil, fmt.Errorf("configure %s: %w\n%s", name, err, out)
	}

	if err := runMake(ctx, spec, srcPath); err != nil {
		return "", nil, err
	}

	return srcPath, repository.ForgeEnv(prefix), nil
}

func (f *ForgeRepository) buildMakeOnly(ctx context.Context, spec buildSpec) (string, map[string]string, error) {
	srcPath := spec.srcPath

	prefix := filepath.Dir(srcPath)

	if err := runMake(ctx, spec, srcPath); err != nil {
		return "", nil, err
	}

	return srcPath, repository.ForgeEnv(prefix), nil
}

func runMake(ctx context.Context, spec buildSpec, srcPath string) error {
	name, version, extraEnv, verbose, jobs := spec.name, spec.version, spec.extraEnv, spec.verbose, spec.jobs
	args := repository.MakeArgs(name, version, jobs)

	make := exec.CommandContext(ctx, "make", args...)
	make.Dir = srcPath
	make.Env = repository.MergeEnv(extraEnv)
	if out, err := repository.RunCmd(ctx, make, verbose); err != nil {
		return fmt.Errorf("make %s: %w\n%s", name, err, out)
	}
	return nil
}
