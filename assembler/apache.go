package assembler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/supanadit/phpv/compiler"
)

// ApacheService builds the on-demand Apache module variant (libphp.so) used
// by `phpv server run --server apache --sapi modphp`. Like the FPM variant it
// is built from a pristine source copy using the CLI build profile, so it has
// the same extensions as the CLI and FPM.
type ApacheService struct {
	deps *Service
	root string
}

// ApacheStages are the resumable build phases, in order.
var ApacheStages = []string{"prepare", "configure", "make", "install"}

// NewApacheService builds an Apache module service on top of the assembler.
func NewApacheService(deps *Service, rootDir string) *ApacheService {
	return &ApacheService{deps: deps, root: rootDir}
}

// ModulePrefix returns the install prefix of the Apache module variant.
func (a *ApacheService) ModulePrefix(phpVersion string) string {
	return filepath.Join(a.root, "packages", "php-apache", phpVersion)
}

// ModuleWorkDir returns the build directory for the module variant.
func (a *ApacheService) ModuleWorkDir(phpVersion string) string {
	return filepath.Join(a.root, "build", "php-apache", phpVersion)
}

// ModulePath returns the built libphp.so path, or "" when not installed.
func (a *ApacheService) ModulePath(phpVersion string) string {
	p := filepath.Join(a.ModulePrefix(phpVersion), "modules", "libphp.so")
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		return p
	}
	return ""
}

// NeedsRebuild reports whether the cached Apache module is missing or was
// built with a different CLI profile/httpd/compiler.
func (a *ApacheService) NeedsRebuild(phpVersion, httpdPrefix string, tc compiler.Toolchain) bool {
	if a.ModulePath(phpVersion) == "" {
		return true
	}
	cliPrefix := filepath.Join(a.root, "packages", "php", phpVersion)
	profile, err := loadCLIProfile(phpVersion, cliPrefix, a.root)
	if err != nil {
		return true
	}
	hash := profile.Hash("modphp") + ":" + filepath.Base(httpdPrefix) + ":" + cliExtensionDir(cliPrefix)
	return !variantMatches(a.ModulePrefix(phpVersion), "modphp", phpVersion, hash)
}

// EnsureApacheModule builds libphp.so against the installed httpd (apxs) and
// returns its path. httpdPrefix must point at an installed Apache (httpd)
// package.
func (a *ApacheService) EnsureApacheModule(ctx context.Context, phpVersion, httpdPrefix string, verbose bool, jobs int, tc compiler.Toolchain, progress ProgressFunc) (string, error) {
	cliPrefix := filepath.Join(a.root, "packages", "php", phpVersion)
	profile, err := loadCLIProfile(phpVersion, cliPrefix, a.root)
	if err != nil {
		return "", err
	}
	extDir := cliExtensionDir(cliPrefix)
	prefix := a.ModulePrefix(phpVersion)
	hash := profile.Hash("modphp") + ":" + filepath.Base(httpdPrefix) + ":" + extDir

	if module := a.ModulePath(phpVersion); module != "" {
		if variantMatches(prefix, "modphp", phpVersion, hash) {
			return module, nil
		}
		args, err := profile.VariantArgs("modphp", prefix, httpdPrefix)
		if err != nil {
			return "", err
		}
		env := profile.ApplyEnv(os.Environ())
		env = setEnvVar(env, "CC", tc.CC)
		env = setEnvVar(env, "CXX", tc.CXX)
		if extDir != "" {
			env = setEnvVar(env, "EXTENSION_DIR", extDir)
		}
		_ = os.RemoveAll(a.ModuleWorkDir(phpVersion))
		if err := safeRebuild(prefix, func() error {
			return a.build(ctx, phpVersion, args, env, verbose, jobs, hash, progress)
		}); err != nil {
			return "", err
		}
		return a.ModulePath(phpVersion), nil
	}

	args, err := profile.VariantArgs("modphp", prefix, httpdPrefix)
	if err != nil {
		return "", err
	}
	env := profile.ApplyEnv(os.Environ())
	env = setEnvVar(env, "CC", tc.CC)
	env = setEnvVar(env, "CXX", tc.CXX)
	if extDir != "" {
		env = setEnvVar(env, "EXTENSION_DIR", extDir)
	}

	if err := a.build(ctx, phpVersion, args, env, verbose, jobs, hash, progress); err != nil {
		return "", err
	}
	module := a.ModulePath(phpVersion)
	if module == "" {
		return "", fmt.Errorf("apache module for PHP %s was not produced by the build", phpVersion)
	}
	return module, nil
}

func (a *ApacheService) build(ctx context.Context, phpVersion string, args, env []string, verbose bool, jobs int, hash string, progress ProgressFunc) error {
	if jobs < 1 {
		jobs = 4
	}
	emit := func(stage, msg string) {
		if verbose {
			fmt.Printf("  %s\n", msg)
		}
		if progress != nil {
			progress(stage, msg)
		}
	}

	workDir := a.ModuleWorkDir(phpVersion)
	prefix := a.ModulePrefix(phpVersion)
	// Apache module builds are quick enough to always start clean; this
	// avoids reusing objects configured for a different MPM/ZTS profile.
	_ = os.RemoveAll(workDir)

	emit("prepare", fmt.Sprintf("php-apache %s not built yet — one-time setup", phpVersion))
	if err := preparePHPSource(a.deps, phpVersion, workDir); err != nil {
		return err
	}
	emit("configure", fmt.Sprintf("configuring php-apache %s…", phpVersion))
	srcPath, err := variantSourcePath(workDir)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "./configure", args...)
	cmd.Dir = srcPath
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			return fmt.Errorf("configure php-apache %s: %w\n%s", phpVersion, err, out)
		}
		return fmt.Errorf("configure php-apache %s: %w (re-run with --verbose for details)", phpVersion, err)
	}

	emit("make", fmt.Sprintf("compiling php-apache %s…", phpVersion))
	makeCmd := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	makeCmd.Dir = srcPath
	out, err = makeCmd.CombinedOutput()
	if err != nil {
		if verbose {
			return fmt.Errorf("make php-apache %s: %w\n%s", phpVersion, err, out)
		}
		return fmt.Errorf("make php-apache %s: %w (re-run with --verbose for details)", phpVersion, err)
	}

	emit("install", fmt.Sprintf("installing libphp.so for PHP %s…", phpVersion))
	built, err := findApacheModule(srcPath)
	if err != nil {
		return err
	}
	dstDir := filepath.Join(prefix, "modules")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	if err := copyFileMode(built, filepath.Join(dstDir, "libphp.so"), 0o755); err != nil {
		return fmt.Errorf("install libphp.so: %w", err)
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "modphp",
		PHPVersion:  phpVersion,
		ProfileHash: hash,
	}); err != nil {
		return fmt.Errorf("write apache module build marker: %w", err)
	}
	emit("done", fmt.Sprintf("php-apache %s ready", phpVersion))
	return nil
}

// findApacheModule locates libphp.so in a built PHP source tree.
func findApacheModule(srcPath string) (string, error) {
	candidates := []string{
		filepath.Join(srcPath, "libs", "libphp.so"),
		filepath.Join(srcPath, "modules", "libphp.so"),
		filepath.Join(srcPath, "sapi", "apache2handler", ".libs", "libphp.so"),
	}
	for _, candidate := range candidates {
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate, nil
		}
	}
	for _, pattern := range []string{
		filepath.Join(srcPath, "libs", "libphp*.so"),
		filepath.Join(srcPath, "modules", "libphp*.so"),
	} {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return matches[0], nil
		}
	}
	return "", fmt.Errorf("libphp.so not found after make in %s", srcPath)
}
