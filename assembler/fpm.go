package assembler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/compiler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/source"
)

// FPMService builds and manages the on-demand PHP-FPM variant used by
// `phpv server`.
//
// Why a separate variant instead of adding --enable-fpm to normal installs:
//   - the CLI install (--disable-all --enable-cli) has no FPM SAPI, and
//     nginx/caddy reach PHP only over FastCGI;
//   - rebuilding a prefix with a different SAPI set would silently change an
//     install the user already validated.
//
// So the FPM variant lives at $PHPV_ROOT/packages/php-fpm/<ver>/ and is built
// from a COPY of the php source tree (the CLI build's configure state must not
// be disturbed). It compiles no extensions: the generated pool config points
// extension_dir at the CLI install's extension dir and includes its INI, so
// FPM loads the very same shared .so files as the CLI.
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

// EnsureFPM installs the FPM variant when missing and returns the php-fpm
// binary path. stagedFrom resumes from a previously completed phase
// ("" starts from the beginning); stages before it are skipped. progress
// receives stage updates for the caller's spinner; verbose prints raw lines.
func (f *FPMService) EnsureFPM(ctx context.Context, phpVersion string, verbose bool, jobs int, tc compiler.Toolchain, stagedFrom string, progress ProgressFunc) (string, error) {
	if bin := f.FPMBinary(phpVersion); bin != "" {
		return bin, nil
	}
	if jobs < 1 {
		jobs = 4
	}

	start := 0
	for i, s := range FPMStages {
		if s == stagedFrom {
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

	// Every stage reports unconditionally: a cold FPM build takes minutes,
	// and silence reads as a hang.
	if start <= 0 {
		emit("prepare", fmt.Sprintf("php-fpm %s not built yet — one-time setup", phpVersion))
		if err := f.phasePrepare(phpVersion); err != nil {
			return "", err
		}
		emit("configure", fmt.Sprintf("configuring php-fpm %s…", phpVersion))
		if err := f.phaseConfigure(ctx, phpVersion, verbose, tc); err != nil {
			return "", err
		}
		start = 2
	}
	if start <= 2 {
		emit("make", fmt.Sprintf("compiling php-fpm %s (a few minutes on first run)…", phpVersion))
		if err := f.phaseMake(ctx, phpVersion, verbose, jobs); err != nil {
			return "", err
		}
	}
	if start <= 3 {
		emit("install", fmt.Sprintf("installing php-fpm %s…", phpVersion))
		if err := f.phaseInstall(ctx, phpVersion, jobs); err != nil {
			return "", err
		}
	}
	emit("done", fmt.Sprintf("php-fpm %s ready", phpVersion))
	return f.FPMBinary(phpVersion), nil
}

// phasePrepare materializes a pristine php source tree for the FPM build:
// extract the cached tarball into the work dir and re-apply the published
// patches. Copying the CLI build's configured tree instead would carry its
// stale .deps/.lo state (absolute paths into the original source dir) and
// break make.
func (f *FPMService) phasePrepare(phpVersion string) error {
	work := f.FPMWorkDir(phpVersion)
	marker := filepath.Join(work, "src", "configure")
	if _, err := os.Stat(marker); err == nil {
		return nil // already prepared; resume
	}
	reg, err := f.deps.reg.Get("php", phpVersion)
	if err != nil {
		return fmt.Errorf("resolve php %s archive: %w", phpVersion, err)
	}
	archivePath := filepath.Join(cacheDir(), filepath.Base(reg.URL))
	if _, err := os.Stat(archivePath); err != nil {
		if _, err := f.deps.silo.DownloadURL(reg.URL, reg.ChecksumType, reg.ChecksumValue); err != nil {
			return fmt.Errorf("download php %s for fpm build: %w", phpVersion, err)
		}
	}
	if err := os.RemoveAll(work); err != nil {
		return fmt.Errorf("clean fpm work dir: %w", err)
	}
	if _, err := f.deps.silo.Extract(archivePath, filepath.Join(work, "src")); err != nil {
		return fmt.Errorf("extract php source for fpm build: %w", err)
	}
	srcPath := source.Find(filepath.Join(work, "src"), "php")
	if srcPath == "" {
		return fmt.Errorf("php %s source not found after extract in %s", phpVersion, work)
	}
	if _, err := f.deps.patcher.Prepare("php", phpVersion, srcPath); err != nil {
		return fmt.Errorf("patch php source for fpm build: %w", err)
	}
	return nil
}

// fpmConfigureFlags are the configure flags for the minimal FPM build.
func (f *FPMService) fpmConfigureFlags(prefix string) []string {
	return []string{
		"--prefix=" + prefix,
		"--disable-all",
		"--disable-cli",
		"--disable-cgi",
		"--enable-fpm",
		"--disable-maintainer-mode",
		// ext/standard's image.c compiles its SWC handler when zlib.h is
		// found (header check, not the --with-zlib flag), so a --disable-all
		// build still references uncompress() and the link fails without
		// -lz. Pointing --with-zlib at the local zlib install keeps the
		// link self-consistent (same zlib the CLI build uses).
		"--with-zlib={{dep:zlib}}",
	}
}

// fpmSrcPath returns the actual build dir inside the work dir (the tarball
// extracts into a top-level php-<version>/ directory).
func (f *FPMService) fpmSrcPath(phpVersion string) (string, error) {
	p := source.Find(filepath.Join(f.FPMWorkDir(phpVersion), "src"), "php")
	if p == "" {
		return "", fmt.Errorf("php-fpm work dir for %s is not prepared (run again)", phpVersion)
	}
	return p, nil
}

func (f *FPMService) phaseConfigure(ctx context.Context, phpVersion string, verbose bool, tc compiler.Toolchain) error {
	work, err := f.fpmSrcPath(phpVersion)
	if err != nil {
		return err
	}
	prefix := f.FPMPrefix(phpVersion)

	// Era compiler flags, identical to the regular php build pipeline:
	// legacy PHP sources need -Wno-implicit-function-declaration and
	// friends on GCC 14+, and configure bakes CFLAGS into the generated
	// Makefile, so they must be set here (not only at make time).
	plan, err := f.deps.graph.GetBuildPlan("php", phpVersion, nil, tc.Name)
	if err != nil {
		return fmt.Errorf("resolve fpm compiler flags: %w", err)
	}
	cflags := append([]string(nil), plan.CFlags...)
	cflags = append(cflags, plan.CompilerFlags...)
	if plan.CompilerFlags == nil {
		// CompilerFlags already includes the -std=gnu… suffix when set;
		// fall back to the raw std rule for safety.
		rule := f.deps.graph.GetCompilerStdRule(phpVersion)
		if rule.CStd != "" {
			cflags = append(cflags, rule.CStd)
		}
	}

	env := append(os.Environ(), "CC="+tc.CC, "CXX="+tc.CXX)
	if len(cflags) > 0 {
		env = append(env, "CFLAGS="+strings.Join(cflags, " "))
	}

	// Resume support: an existing Makefile means configure already ran
	// (the flag set is deterministic per version+prefix), so an interrupted
	// make can resume without reconfiguring.
	if _, err := os.Stat(filepath.Join(work, "Makefile")); err == nil {
		return nil
	}

	args := f.fpmConfigureFlags(prefix)
	args = f.resolveFPMFlags(args, phpVersion)
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

// resolveFPMFlags resolves package paths in FPM configure flags. The
// minimal FPM build plan has no dependency entries, so {{dep:NAME}} is
// resolved by scanning installed packages directly; unresolved flags fall
// back to their bare form (system library).
func (f *FPMService) resolveFPMFlags(flags []string, phpVersion string) []string {
	out := make([]string, 0, len(flags))
	for _, flag := range flags {
		if strings.Contains(flag, "{{dep:zlib}}") {
			if prefix := f.latestPackagePrefix("zlib", "include/zlib.h"); prefix != "" {
				out = append(out, strings.ReplaceAll(flag, "{{dep:zlib}}", prefix))
			} else {
				out = append(out, "--with-zlib") // system zlib, links -lz
			}
			continue
		}
		if strings.Contains(flag, "{{dep:") || strings.Contains(flag, "{{source:") {
			continue // unresolved placeholders: let configure find the system lib
		}
		out = append(out, flag)
	}
	return out
}

// latestPackagePrefix returns the newest installed prefix of a package whose
// prefix contains the sentinel file ("" when the package is not installed).
func (f *FPMService) latestPackagePrefix(name, sentinel string) string {
	pkgDir := filepath.Join(f.root, "packages", name)
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return ""
	}
	var best string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(pkgDir, e.Name(), sentinel)); err != nil {
			continue
		}
		if best == "" || domain.CompareVersions(e.Name(), best) > 0 {
			best = e.Name()
		}
	}
	if best == "" {
		return ""
	}
	return filepath.Join(pkgDir, best)
}

func (f *FPMService) phaseMake(ctx context.Context, phpVersion string, verbose bool, jobs int) error {
	work, err := f.fpmSrcPath(phpVersion)
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
	work, err := f.fpmSrcPath(phpVersion)
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
