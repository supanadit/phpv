package assembler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/compiler"
)

// FPMService builds and manages the on-demand PHP-FPM variant used by
// `phpv server`.
//
// The variant is built from a COPY of the php source tree so the CLI build's
// configure state is not disturbed. It reuses the CLI build profile: the same
// static extensions and dependency paths are passed to configure, and shared
// extensions are loaded at runtime from the CLI install's extension dir. That
// keeps `php-fpm` extension-compatible with `php -S`.
type FPMService struct {
	deps *Service
	root string
}

// FPMStages are the resumable build phases, in order.
var FPMStages = []string{"prepare", "configure", "make", "install"}

// NewFPMService builds an FPM service on top of the main assembler.
func NewFPMService(deps *Service, rootDir string) *FPMService {
	return &FPMService{deps: deps, root: rootDir}
}

// FPMPrefix returns the install prefix of the FPM variant for a PHP version.
func (f *FPMService) FPMPrefix(phpVersion string) string {
	return filepath.Join(f.root, "packages", "php-fpm", phpVersion)
}

// FPMWorkDir returns the build directory holding the copied source tree.
func (f *FPMService) FPMWorkDir(phpVersion string) string {
	return filepath.Join(f.root, "build", "php-fpm", phpVersion)
}

// FPMBinary returns the php-fpm binary path, or "" when not installed.
func (f *FPMService) FPMBinary(phpVersion string) string {
	p := filepath.Join(f.FPMPrefix(phpVersion), "sbin", "php-fpm")
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		return p
	}
	return ""
}

// NeedsRebuild reports whether the cached FPM binary is missing or was built
// with a different CLI profile/SAPI/compiler.
func (f *FPMService) NeedsRebuild(phpVersion string, tc compiler.Toolchain) bool {
	if f.FPMBinary(phpVersion) == "" {
		return true
	}
	cliPrefix := filepath.Join(f.root, "packages", "php", phpVersion)
	profile, err := loadCLIProfile(phpVersion, cliPrefix, f.root)
	if err != nil {
		return true
	}
	hash := profile.Hash("fpm")
	return !variantMatches(f.FPMPrefix(phpVersion), "fpm", phpVersion, hash)
}

// EnsureFPM installs the FPM variant when missing (or when the CLI build
// profile changed) and returns the php-fpm binary path. stagedFrom resumes
// from a previously completed phase.
func (f *FPMService) EnsureFPM(ctx context.Context, phpVersion string, verbose bool, jobs int, tc compiler.Toolchain, stagedFrom string, progress ProgressFunc) (string, error) {
	cliPrefix := filepath.Join(f.root, "packages", "php", phpVersion)
	profile, err := loadCLIProfile(phpVersion, cliPrefix, f.root)
	if err != nil {
		return "", err
	}
	prefix := f.FPMPrefix(phpVersion)
	args, err := profile.VariantArgs("fpm", prefix, "")
	if err != nil {
		return "", err
	}
	env := profile.ApplyEnv(os.Environ())
	env = setEnvVar(env, "CC", tc.CC)
	env = setEnvVar(env, "CXX", tc.CXX)
	hash := profile.Hash("fpm")

	if bin := f.FPMBinary(phpVersion); bin != "" {
		if variantMatches(prefix, "fpm", phpVersion, hash) {
			return bin, nil
		}
		// The CLI install or the variant flags changed: rebuild from
		// scratch, keeping the old binary only if the new build fails.
		stagedFrom = ""
		_ = os.RemoveAll(f.FPMWorkDir(phpVersion))
		if err := safeRebuild(prefix, func() error {
			return f.build(ctx, phpVersion, args, env, verbose, jobs, hash, stagedFrom, progress)
		}); err != nil {
			return "", err
		}
		return f.FPMBinary(phpVersion), nil
	}

	if err := f.build(ctx, phpVersion, args, env, verbose, jobs, hash, stagedFrom, progress); err != nil {
		return "", err
	}
	return f.FPMBinary(phpVersion), nil
}

func (f *FPMService) build(ctx context.Context, phpVersion string, args, env []string, verbose bool, jobs int, hash, stagedFrom string, progress ProgressFunc) error {
	if jobs < 1 {
		jobs = 4
	}

	start := 0
	for i, stage := range FPMStages {
		if stage == stagedFrom {
			start = i
			break
		}
	}

	emit := func(stage, msg string) {
		if verbose {
			fmt.Printf("  %s\n", msg)
		}
		if progress != nil {
			progress(stage, msg)
		}
	}

	prefix := f.FPMPrefix(phpVersion)
	// Every stage reports unconditionally: a cold FPM build takes minutes,
	// and silence reads as a hang.
	if start <= 0 {
		emit("prepare", fmt.Sprintf("php-fpm %s not built yet — one-time setup", phpVersion))
		if err := preparePHPSource(f.deps, phpVersion, f.FPMWorkDir(phpVersion)); err != nil {
			return err
		}
		emit("configure", fmt.Sprintf("configuring php-fpm %s…", phpVersion))
		if err := f.phaseConfigure(ctx, phpVersion, args, env, verbose); err != nil {
			return err
		}
		start = 2
	}
	if start <= 2 {
		emit("make", fmt.Sprintf("compiling php-fpm %s (a few minutes on first run)…", phpVersion))
		if err := f.phaseMake(ctx, phpVersion, verbose, jobs); err != nil {
			return err
		}
	}
	if start <= 3 {
		emit("install", fmt.Sprintf("installing php-fpm %s…", phpVersion))
		if err := f.phaseInstall(ctx, phpVersion, jobs); err != nil {
			return err
		}
	}
	if err := writeVariantMarker(prefix, VariantMarker{
		SAPI:        "fpm",
		PHPVersion:  phpVersion,
		ProfileHash: hash,
	}); err != nil {
		return fmt.Errorf("write fpm build marker: %w", err)
	}
	emit("done", fmt.Sprintf("php-fpm %s ready", phpVersion))
	return nil
}

func (f *FPMService) phaseConfigure(ctx context.Context, phpVersion string, args, env []string, verbose bool) error {
	work, err := variantSourcePath(f.FPMWorkDir(phpVersion))
	if err != nil {
		return err
	}

	// Resume support: an existing Makefile means configure already ran
	// with this profile, so an interrupted make can resume.
	if _, err := os.Stat(filepath.Join(work, "Makefile")); err == nil {
		return nil
	}

	cmd := exec.CommandContext(ctx, "./configure", args...)
	cmd.Dir = work
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			return fmt.Errorf("configure php-fpm %s: %w\n%s", phpVersion, err, out)
		}
		return fmt.Errorf("configure php-fpm %s: %w (re-run with --verbose for details)", phpVersion, err)
	}
	return nil
}

func (f *FPMService) phaseMake(ctx context.Context, phpVersion string, verbose bool, jobs int) error {
	work, err := variantSourcePath(f.FPMWorkDir(phpVersion))
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	cmd.Dir = work
	out, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			return fmt.Errorf("make php-fpm %s: %w\n%s", phpVersion, err, out)
		}
		return fmt.Errorf("make php-fpm %s: %w (re-run with --verbose for details)", phpVersion, err)
	}
	return nil
}

func (f *FPMService) phaseInstall(ctx context.Context, phpVersion string, jobs int) error {
	work, err := variantSourcePath(f.FPMWorkDir(phpVersion))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(f.FPMPrefix(phpVersion), 0o755); err != nil {
		return fmt.Errorf("create fpm prefix: %w", err)
	}
	cmd := exec.CommandContext(ctx, "make", "install")
	cmd.Dir = work
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install php-fpm %s: %w\n%s", phpVersion, err, strings.TrimSpace(string(out)))
	}
	return nil
}
