package assembler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/compiler"
	"github.com/supanadit/phpv/pkg/progress"
	"github.com/supanadit/phpv/pkg/source"
)

// AssemblerResult holds the outcome of assembling a package.
type AssemblerResult = domain.AssemblerResult

// DownloadResult holds the outcome of downloading + extracting a single package.
type DownloadResult = domain.DownloadResult

// ProgressFunc receives human-readable status updates during assembly.
// Pass nil to disable progress reporting.
type ProgressFunc = progress.Func

// AssemblerGraph is the build-plan and flag knowledge assembler consumes.
type AssemblerGraph interface {
	GetBuildPlan(name, version string, extensions []string, compilerName string) (*domain.BuildPlan, error)
	GetCompilerStdRule(phpVersion string) domain.CompilerRule
	CXXFlagsFromCFlagsWithStd(cflags []string, isPHPBuild bool, stdRule domain.CompilerRule) []string
	GetConfigureFlags(name, version string) []string
}

// AssemblerExtensions is the extension-catalog capability assembler consumes.
type AssemblerExtensions interface {
	ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo
	DefaultExtensions(phpVersion string) (included, skipped []string)
	ExpandImplied(extensions []string) (expanded, added []string)
	SharedOnlyExtensions(phpVersion string, requested []string) []string
	GetExtensionDef(name string) (domain.ExtensionDef, bool)
	GetExtensionConfigureFlags(name, phpVersion string) []string
}

// AssemblerSilo is the package-storage capability assembler consumes.
type AssemblerSilo interface {
	Download(name, version string) (bool, error)
	DownloadURL(url, checksumType, checksumValue string) (bool, error)
	Extract(archivePath, destDir string) (bool, error)
	GetState(name, version string) (domain.InstallState, error)
	MarkInProgress(name, version string) error
	MarkComplete(name, version string) error
	MarkFailed(name, version string) error
	MarkInterrupted(name, version string) error
	SourcePath(pkg, version string) string
	PackagePrefix(name, version string) string
	GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error)
	SaveExtensionManifest(phpVersion string, m *domain.ExtensionManifest) error
}

// AssemblerForge is the source-build capability assembler consumes.
type AssemblerForge interface {
	Build(ctx context.Context, req domain.BuildRequest) (string, map[string]string, error)
	Install(ctx context.Context, req domain.InstallRequest) error
}

// AssemblerPatcher is the source-patching capability assembler consumes.
type AssemblerPatcher interface {
	Prepare(name, version, sourceDir string) (*domain.PreparedPatches, error)
}

// AssemblerRegistry is the package-index capability assembler consumes.
type AssemblerRegistry interface {
	Get(name, version string) (domain.Registry, error)
	List(name string) ([]domain.Registry, error)
}

// AssemblerMirror is the patch-tree freshness capability assembler consumes.
type AssemblerMirror interface {
	EnsureFresh(ctx context.Context) (domain.PatchSyncResult, error)
	Update(ctx context.Context, scope *domain.PatchSyncScope) (domain.PatchSyncResult, error)
}

// ServiceDeps groups the assembler's injected capabilities.
type ServiceDeps struct {
	Graph      AssemblerGraph
	Extensions AssemblerExtensions
	Silo       AssemblerSilo
	Forge      AssemblerForge
	Patcher    AssemblerPatcher
	Registry   AssemblerRegistry
	Mirror     AssemblerMirror
}

// Service orchestrates the full build pipeline for any package:
// resolve → get build plan → download → build deps → build package → mark complete.
// It composes service wrappers that each add value over their underlying repositories.
type Service struct {
	graph      AssemblerGraph
	extensions AssemblerExtensions
	silo       AssemblerSilo
	forge      AssemblerForge
	patcher    AssemblerPatcher
	reg        AssemblerRegistry
	// patches, when non-nil, is consulted before every build so freshly
	// published patches reach installs regardless of binary age. A nil
	// service disables the self-sync (e.g. in tests).
	patches AssemblerMirror
}

// NewService creates a new assembler service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		graph:      deps.Graph,
		extensions: deps.Extensions,
		silo:       deps.Silo,
		forge:      deps.Forge,
		patcher:    deps.Patcher,
		reg:        deps.Registry,
		patches:    deps.Mirror,
	}
}

// Graph returns the graph service used by the assembler.
func (s *Service) Graph() AssemblerGraph {
	return s.graph
}

// ListExtensionsForPHP delegates to the graph service.
func (s *Service) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	return s.extensions.ListExtensionsForPHP(phpVersion)
}

// DefaultExtensions delegates to the graph service.
func (s *Service) DefaultExtensions(phpVersion string) (included []string, skipped []string) {
	return s.extensions.DefaultExtensions(phpVersion)
}

// ExpandImplied delegates to the graph service.
func (s *Service) ExpandImplied(extensions []string) (expanded []string, added []string) {
	return s.extensions.ExpandImplied(extensions)
}

// SharedOnlyExtensions delegates to the graph service.
func (s *Service) SharedOnlyExtensions(phpVersion string, requested []string) []string {
	return s.extensions.SharedOnlyExtensions(phpVersion, requested)
}

// GetExtensionDef delegates to the graph service.
func (s *Service) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return s.extensions.GetExtensionDef(name)
}

// syncPatches runs the pre-build patch-tree freshness check. Failures
// are emitted as warnings and never abort the build.
func (s *Service) syncPatches(ctx context.Context, emit func(stage, msg string)) {
	if s.patches == nil {
		return
	}
	res, err := s.patches.EnsureFresh(ctx)
	switch {
	case err != nil:
		emit("warn", fmt.Sprintf("Patch freshness could not be verified (%v); continuing with the locally available patch tree", err))
	case res.SkippedCheck:
		emit("patch-sync", "Patch tree checked recently; skipping freshness check")
	case len(res.Downloaded) > 0:
		emit("patch-sync", fmt.Sprintf("Synced %d patch file(s) from the publication source", len(res.Downloaded)))
	}
}

// AssembleSpec groups the inputs to a full PHP assembly.
type AssembleSpec = domain.AssembleSpec

// depBuildSpec groups the inputs to a dependency build pass.
type depBuildSpec struct {
	deps            []domain.Dependency
	mustBuild       map[string]bool
	systemPkgs      map[string]domain.SystemPackage
	static, verbose bool
	jobs            int
	tc              compiler.Toolchain
	emit            ProgressFunc
}

// downloadSpec groups the inputs to a multi-package download pass.
type downloadSpec struct {
	name, version string
	deps          []domain.Dependency
	mustBuild     map[string]bool
	systemPkgs    map[string]domain.SystemPackage
	static        bool
	emit          ProgressFunc
}

// Assemble runs the full pipeline for (name, version).
// systemPkgs optionally provides a map of available system packages for hybrid builds.
// jobs controls make parallelism (0 = auto).
// tc is the resolved compiler toolchain (see compiler.Resolve) used for both
// compiler flag selection and the CC/CXX of every build it spawns.
func (s *Service) Assemble(ctx context.Context, spec AssembleSpec) (*AssemblerResult, error) {
	name, version, static, extensions, verbose, progress, systemPkgs, jobs, force := spec.Name, spec.Version, spec.Static, spec.Extensions, spec.Verbose, spec.Progress, spec.SystemPkgs, spec.Jobs, spec.Force
	tc := compiler.Toolchain{Name: spec.TC.Name, CC: spec.TC.CC, CXX: spec.TC.CXX, Hint: spec.TC.Hint}
	emit := func(stage, msg string) {
		if progress != nil {
			progress(stage, msg)
		}
	}

	emit("resolve", fmt.Sprintf("Resolving %s version %q...", name, version))

	// Self-sync the managed patch tree before building: installs ride
	// the same network connection as the source downloads, so a stale
	// binary still receives newly published patches. Failure is a
	// warning, never a hard error (the build continues with the previous
	// synced tree, or unpatched on a cold machine).
	s.syncPatches(ctx, emit)

	exactVersion, err := s.ResolveVersion(name, version)
	if err != nil {
		return nil, fmt.Errorf("resolve version: %w", err)
	}
	emit("resolve", fmt.Sprintf("Resolved %s %s", name, exactVersion))

	prefix := s.silo.PackagePrefix(name, exactVersion)

	state, err := s.silo.GetState(name, exactVersion)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}
	if !force && state == domain.StateInstalled {
		return &AssemblerResult{Version: exactVersion, AlreadyInstalled: true}, nil
	}

	if err := s.silo.MarkInProgress(name, exactVersion); err != nil {
		return nil, fmt.Errorf("mark in-progress: %w", err)
	}

	var completed bool
	defer func() {
		if !completed {
			if ctx.Err() != nil {
				s.silo.MarkInterrupted(name, exactVersion)
			} else {
				s.silo.MarkFailed(name, exactVersion)
			}
		}
	}()

	emit("deps", "Resolving dependency graph...")
	plan, err := s.graph.GetBuildPlan(name, exactVersion, extensions, tc.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve build plan for %s@%s: %w", name, exactVersion, err)
	}
	emit("deps", fmt.Sprintf("Found %d dependencies", len(plan.Deps)))
	for _, w := range plan.Warnings {
		emit("deps", w)
	}

	// Determine which deps must be built from source because their system
	// version does not satisfy the required constraint. This decision only
	// needs the build plan and the system package map (already available
	// before any download), so it runs first and lets us download only the
	// packages that will actually be compiled. If a dep is built from source,
	// other deps that link against it must also be built from source to avoid
	// ABI conflicts. The most common case is curl linking against openssl:
	// if we build OpenSSL 1.1.1w locally, system curl (linked against OpenSSL
	// 3.x) would cause linker/runtime conflicts, so curl must also be local.
	mustBuild := s.computeMustBuild(plan.Deps, systemPkgs, static)
	// Propagate: if openssl is built locally, curl must be too.
	if mustBuild["openssl"] {
		mustBuild["curl"] = true
	}

	// Count how many archives will actually be fetched (skipping build tools
	// and deps that will use the system package) so the progress message is
	// honest about what is about to be downloaded.
	toDownload := 0
	for _, dep := range plan.Deps {
		if dep.Optional && dep.Version == "" {
			continue
		}
		if shouldDownloadDep(dep, mustBuild, systemPkgs, static) {
			toDownload++
		}
	}
	emit("download", fmt.Sprintf("Downloading and extracting %d packages...", toDownload+1))
	downloadResults, err := s.downloadAll(downloadSpec{name: name, version: exactVersion, deps: plan.Deps, mustBuild: mustBuild, systemPkgs: systemPkgs, static: static, emit: emit})
	if err != nil {
		return nil, err
	}
	var downloaded, skipped, extracted int
	for _, dr := range downloadResults {
		if dr.Err == nil {
			if dr.Downloaded {
				downloaded++
			} else {
				skipped++
			}
			if dr.Extracted {
				extracted++
			}
		}
	}
	emit("download", fmt.Sprintf("Downloaded: %d, Skipped: %d, Extracted: %d", downloaded, skipped, extracted))

	// Build all source dependencies (or resolve them against system
	// packages). Extracted so the PHP-FPM variant build can reuse it.
	depRes, err := s.buildDependencies(ctx, depBuildSpec{deps: plan.Deps, mustBuild: mustBuild, systemPkgs: systemPkgs, static: static, verbose: verbose, jobs: jobs, tc: tc, emit: emit})
	if err != nil {
		return nil, err
	}
	localCppFlags, localLdFlags, localPcPaths := depRes.LocalCppFlags, depRes.LocalLdFlags, depRes.LocalPcPaths
	systemCppFlags, systemLdFlags, systemPcPaths := depRes.SystemCppFlags, depRes.SystemLdFlags, depRes.SystemPcPaths
	localLibraryPaths, localBinPaths := depRes.LibraryPaths, depRes.BinPaths

	// Combine flags: local (isolated) flags first so they take precedence over
	// system headers/libraries when both are present.
	depCppFlags := append(localCppFlags, systemCppFlags...)
	depLdFlags := append(localLdFlags, systemLdFlags...)
	depPkgConfigPaths := append(localPcPaths, systemPcPaths...)

	sourceDir := s.silo.SourcePath(name, exactVersion)
	srcPath := source.Find(sourceDir, name)
	if srcPath == "" {
		return nil, fmt.Errorf("could not find source directory in %s", sourceDir)
	}

	env := os.Environ()
	// Pin the selected toolchain so the build never silently falls back to
	// whatever CC/CXX happens to be set in the caller's shell.
	env = setEnvVar(env, "CC", tc.CC)
	env = setEnvVar(env, "CXX", tc.CXX)
	if len(depCppFlags) > 0 {
		env = setEnvVar(env, "CPPFLAGS", strings.Join(depCppFlags, " "))
	}
	if len(depLdFlags) > 0 {
		env = setEnvVar(env, "LDFLAGS", strings.Join(depLdFlags, " "))
	}
	if len(depPkgConfigPaths) > 0 {
		env = setEnvVar(env, "PKG_CONFIG_PATH", strings.Join(depPkgConfigPaths, ":"))
	}
	// Prepend local dep bin directories to PATH so PHP's configure finds
	// local tools (icu-config, curl-config, xml2-config) before system ones.
	if len(localBinPaths) > 0 {
		existingPath := ""
		for _, e := range env {
			if strings.HasPrefix(e, "PATH=") {
				existingPath = strings.TrimPrefix(e, "PATH=")
				break
			}
		}
		if existingPath == "" {
			existingPath = os.Getenv("PATH")
		}
		env = setEnvVar(env, "PATH", strings.Join(append(localBinPaths, existingPath), ":"))
	}
	if len(plan.CFlags) > 0 || len(plan.CompilerFlags) > 0 || len(plan.CXXCompilerFlags) > 0 {
		allCFlags := plan.CFlags
		allCFlags = append(allCFlags, plan.CompilerFlags...)
		env = setEnvVar(env, "CFLAGS", strings.Join(allCFlags, " "))
		compilerRule := s.graph.GetCompilerStdRule(version)
		cxxflags := s.graph.CXXFlagsFromCFlagsWithStd(allCFlags, true, compilerRule)
		cxxflags = append(cxxflags, plan.CXXCompilerFlags...)
		env = setEnvVar(env, "CXXFLAGS", strings.Join(cxxflags, " "))
	}

	// Isolation: ensure local libs are found before system libs at both
	// link time and runtime. Without this, the system OpenSSL 3.x shadows
	// the locally-built OpenSSL 1.1.1w at link time, causing ABI mismatches
	// and "undefined reference" errors for symbols that differ between versions.
	//
	// 1. Prepend -Wl,-rpath,<local>/lib to LDFLAGS so the linker searches
	//    local lib directories first (before system /usr/lib64).
	// 2. Set LD_LIBRARY_PATH to local lib paths only for runtime isolation.
	// 3. Overwrite CPPFLAGS (not append) so system /usr/include is excluded.
	// 4. Add local include paths to CFLAGS so PHP's configure test programs
	//    find the local OpenSSL headers first (not the system 3.x headers).
	if len(depLdFlags) > 0 {
		var rpathFlags, libPaths []string
		for _, flag := range depLdFlags {
			if strings.HasPrefix(flag, "-L") {
				path := strings.TrimPrefix(flag, "-L")
				rpathFlags = append(rpathFlags, "-Wl,-rpath,"+path)
				libPaths = append(libPaths, path)
			}
		}
		if len(rpathFlags) > 0 {
			env = setEnvVar(env, "LDFLAGS", strings.Join(append(rpathFlags, depLdFlags...), " "))
		}
		if len(libPaths) > 0 {
			libPathEnv := "LD_LIBRARY_PATH"
			if runtime.GOOS == "darwin" {
				libPathEnv = "DYLD_LIBRARY_PATH"
			}
			env = setEnvVar(env, libPathEnv, strings.Join(libPaths, ":"))
		}
	}
	if len(depCppFlags) > 0 {
		// Also add local include paths to CFLAGS so PHP's configure test
		// programs find the local headers first. CPPFLAGS alone is not
		// enough because PHP's configure uses CFLAGS for its test compiles.
		existingCFlags := ""
		for _, e := range env {
			if strings.HasPrefix(e, "CFLAGS=") {
				existingCFlags = strings.TrimPrefix(e, "CFLAGS=")
				break
			}
		}
		if existingCFlags != "" {
			env = setEnvVar(env, "CFLAGS", existingCFlags+" "+strings.Join(depCppFlags, " "))
		} else {
			env = setEnvVar(env, "CFLAGS", strings.Join(depCppFlags, " "))
		}
	}

	configureFlags := plan.ConfigureFlags
	configureFlags = s.resolveDependencyFlags(name, exactVersion, configureFlags, plan.Deps)
	if static {
		env = setEnvVar(env, "LDFLAGS", "-static-libgcc -static")
		env = setEnvVar(env, "CFLAGS", "-Os -fdata-sections -ffunction-sections")
		env = setEnvVar(env, "CXXFLAGS", "-Os -fdata-sections -ffunction-sections")
		configureFlags = append([]string{"--enable-static", "--disable-shared"}, configureFlags...)
	}

	// PHP 5.3's generated link line omits -lstdc++ even though its intl
	// extension compiles C++ objects. Newer PHP includes it. Add it to LIBS
	// so it lands after the objects (LDFLAGS would be dropped by --as-needed).
	if runtime.GOOS == "linux" && name == "php" && vsn.CompareVersions(exactVersion, "5.4.0") < 0 {
		env = setEnvVar(env, "LIBS", strings.TrimSpace(os.Getenv("LIBS")+" -lstdc++"))
	}

	if runtime.GOOS == "darwin" && name == "php" {
		existingLdFlags := ""
		for _, e := range env {
			if strings.HasPrefix(e, "LDFLAGS=") {
				existingLdFlags = strings.TrimPrefix(e, "LDFLAGS=")
				break
			}
		}
		macosLdFlags := []string{"-lresolv"}
		brewIconv := "/opt/homebrew/opt/libiconv"
		if _, err := os.Stat(filepath.Join(brewIconv, "lib", "libiconv.dylib")); err == nil {
			macosLdFlags = append([]string{
				"-L" + filepath.Join(brewIconv, "lib"),
				"-Wl,-rpath," + filepath.Join(brewIconv, "lib"),
				"-liconv",
			}, macosLdFlags...)
		}
		env = setEnvVar(env, "LDFLAGS", existingLdFlags+" "+strings.Join(macosLdFlags, " "))
	}

	emit("configure", fmt.Sprintf("Configuring %s...", name))
	prepared, err := s.patcher.Prepare(name, exactVersion, srcPath)
	if err != nil {
		emit("error", fmt.Sprintf("Patch failed for %s: %v", name, err))
		return nil, err
	}
	if len(prepared.Applied) > 0 {
		emit("patch", fmt.Sprintf("Applied %d patch(es) to %s", len(prepared.Applied), name))
	}

	emit("make", fmt.Sprintf("Compiling %s (this may take a while)...", name))
	buildDir, _, err := s.forge.Build(ctx, domain.BuildRequest{Name: name, Version: exactVersion, SourceDir: srcPath, ExtraEnv: env, ExtraConfigureFlags: configureFlags, InstallPrefix: prefix, Verbose: verbose, Jobs: jobs})
	if err != nil {
		emit("error", fmt.Sprintf("Build failed for %s", name))
		return nil, fmt.Errorf("build %s@%s: %w", name, exactVersion, err)

	}
	emit("install", fmt.Sprintf("Installing %s → %s", name, prefix))
	if err := s.forge.Install(ctx, domain.InstallRequest{Name: name, Version: exactVersion, BuildDir: buildDir, Prefix: prefix, Verbose: verbose, Jobs: jobs}); err != nil {
		emit("error", fmt.Sprintf("Install failed for %s", name))
		return nil, fmt.Errorf("install %s@%s: %w", name, exactVersion, err)
	}
	if name == "php" {
		if err := ensureCLIBinary(srcPath, prefix); err != nil {
			emit("error", fmt.Sprintf("Install failed for %s", name))
			return nil, fmt.Errorf("install %s@%s: %w", name, exactVersion, err)
		}
		// Persist how this CLI was configured so SAPI variants (php-fpm,
		// mod_php) can be built with the same extensions and deps.
		if err := saveBuildProfile(prefix, BuildProfile{
			ConfigureArgs: configureFlags,
			Env:           profileEnv(env),
		}); err != nil {
			return nil, fmt.Errorf("save build profile: %w", err)
		}
	}

	emit("done", fmt.Sprintf("%s %s installed at %s", name, exactVersion, prefix))

	localLibraryPaths = appendUnique(localLibraryPaths, filepath.Join(prefix, "lib"))

	completed = true
	if err := s.silo.MarkComplete(name, exactVersion); err != nil {
		return nil, fmt.Errorf("mark complete: %w", err)
	}

	return &AssemblerResult{
		DownloadResults: downloadResults,
		Version:         exactVersion,
		Prefix:          prefix,
		Env: map[string]string{
			libPathEnvName(): strings.Join(localLibraryPaths, ":"),
		},
	}, nil
}

// DepBuildResult carries the compiler/linker flags and paths produced by
// building (or resolving) a build plan's dependency list.
type DepBuildResult struct {
	LocalCppFlags  []string
	LocalLdFlags   []string
	LocalPcPaths   []string
	SystemCppFlags []string
	SystemLdFlags  []string
	SystemPcPaths  []string
	LibraryPaths   []string
	BinPaths       []string
}

// buildDependencies builds every source dependency in the plan (or resolves
// it against a compatible system package) and returns the accumulated flags.
//
// It is the dependency half of Assemble, extracted so alternative PHP SAPIs
// (the FPM variant built for `phpv server`) share one implementation instead
// of duplicating the build loop.
func (s *Service) buildDependencies(ctx context.Context, spec depBuildSpec) (*DepBuildResult, error) {
	deps, mustBuild, systemPkgs, static, verbose, jobs, tc, emit := spec.deps, spec.mustBuild, spec.systemPkgs, spec.static, spec.verbose, spec.jobs, spec.tc, spec.emit
	res := &DepBuildResult{}
	for _, dep := range deps {
		if isBuildTool(dep.Name) {
			emit("skip", fmt.Sprintf("Skipping build tool %s", dep.Name))
			continue
		}
		if dep.Optional && dep.Version == "" {
			emit("skip", fmt.Sprintf("Skipping optional dep %s (no version specified)", dep.Name))
			continue
		}
		depVersion, constraint := vsn.SplitVersionSpec(dep.Version)
		depPrefix := s.silo.PackagePrefix(dep.Name, depVersion)
		sourceDir := s.silo.SourcePath(dep.Name, depVersion)

		if isDepInstalled(depPrefix) {
			// If this dep must be built from source (e.g. curl because openssl
			// is local), an existing install may be ABI-incompatible with the
			// current plan. Force a rebuild so it links against the correct
			// local dependency versions.
			if mustBuild[dep.Name] {
				emit("rebuild", fmt.Sprintf("Rebuilding %s@%s to ensure ABI compatibility", dep.Name, depVersion))
				if err := os.RemoveAll(depPrefix); err != nil {
					emit("error", fmt.Sprintf("Failed to remove stale %s@%s: %v", dep.Name, depVersion, err))
					return nil, err
				}
			} else {
				emit("skip", fmt.Sprintf("Already built %s@%s", dep.Name, depVersion))
				res.LocalCppFlags, res.LocalLdFlags, res.LocalPcPaths = s.collectDepFlags(depPrefix, res.LocalCppFlags, res.LocalLdFlags, res.LocalPcPaths)
				res.LibraryPaths = appendUnique(res.LibraryPaths, filepath.Join(depPrefix, "lib"))
				res.BinPaths = appendUnique(res.BinPaths, filepath.Join(depPrefix, "bin"))
				continue
			}
		}

		// Hybrid mode: check if a system package is available and compatible.
		if !static && systemPkgs != nil && !mustBuild[dep.Name] {
			if sysPkg, ok := systemPkgs[dep.Name]; ok && sysPkg.Installed && sysPkg.Version != "" {
				sysCompat := true
				if constraint != "" {
					sysCompat = vsn.MatchVersionRange(constraint, sysPkg.Version)
				}
				if sysCompat {
					emit("system-use", fmt.Sprintf("Using system %s@%s (satisfies %s)", dep.Name, sysPkg.Version, constraint))
					res.SystemCppFlags, res.SystemLdFlags, res.SystemPcPaths = collectSystemFlags(res.SystemCppFlags, res.SystemLdFlags, res.SystemPcPaths)
					continue
				}
				emit("system-incompatible", fmt.Sprintf("System %s@%s doesn't satisfy %s, building from source", dep.Name, sysPkg.Version, constraint))
			}
		}

		emit("build", fmt.Sprintf("Building %s@%s...", dep.Name, depVersion))
		prepared, err := s.patcher.Prepare(dep.Name, depVersion, sourceDir)
		if err != nil {
			emit("error", fmt.Sprintf("Patch failed for %s@%s: %v", dep.Name, depVersion, err))
			return nil, err
		}
		if len(prepared.Applied) > 0 {
			emit("patch", fmt.Sprintf("Applied %d patch(es) to %s@%s", len(prepared.Applied), dep.Name, depVersion))
		}
		buildEnv := tc.Env()
		// Dependencies built earlier in this run are shared libs installed only
		// under their own prefix (e.g. OpenSSL 1.0.1's libssl.so.1.0.0). A
		// dependent's ./configure links against them via -L but then RUNS its
		// test programs, and the loader only knows LD_LIBRARY_PATH. Without this,
		// curl 7.12's configure crashes with exit 77 ("cannot compute sizeof
		// (curl_off_t)"), surfacing as "OpenSSL libs not found where specified".
		if len(res.LibraryPaths) > 0 {
			sep := string(os.PathListSeparator)
			// Our prefixes come first so the loader finds the libs this run built
			// (e.g. libssl.so.1.0.0) rather than a system one. MergeEnv appends
			// extras after os.Environ() and the last entry wins, so the ambient
			// value must be carried over here explicitly.
			paths := strings.Join(res.LibraryPaths, sep)
			if cur := os.Getenv(libPathEnvName()); cur != "" {
				paths += sep + cur
			}
			buildEnv = append(buildEnv, libPathEnvName()+"="+paths)
		}
		if len(prepared.ExtraCFlags) > 0 {
			buildEnv = append(buildEnv, "CFLAGS="+strings.Join(prepared.ExtraCFlags, " "), "CXXFLAGS="+strings.Join(prepared.ExtraCFlags, " "))
		}
		// TODO: Hardcoded libiconv path
		// Description: This shouldn't be hardcoded, it should instead look like configureFlagRules in internal/repository/memory/graph.go
		if runtime.GOOS == "darwin" {
			brewIconv := "/opt/homebrew/opt/libiconv"
			if _, err := os.Stat(filepath.Join(brewIconv, "include", "iconv.h")); err == nil {
				buildEnv = append(buildEnv,
					"CPPFLAGS=-I"+filepath.Join(brewIconv, "include"),
					"LDFLAGS=-L"+filepath.Join(brewIconv, "lib")+" -Wl,-rpath,"+filepath.Join(brewIconv, "lib"),
				)
			}
		}
		depFlags := s.graph.GetConfigureFlags(dep.Name, depVersion)
		depFlags = append(depFlags, s.resolveDepPlaceholders(prepared.ConfigureFlags, deps)...)
		// Graph defaults and patch flags can both carry an SSL flag (curl has a
		// bare --with-ssl in graph defaults and --with-openssl=<prefix> in its
		// patch). Normalize them to one flag for this package before configure;
		// otherwise the bare flag wins and curl builds against system OpenSSL 3.
		depFlags = s.resolveDependencyFlags(dep.Name, depVersion, depFlags, deps)
		buildDir, _, err := s.forge.Build(ctx, domain.BuildRequest{Name: dep.Name, Version: depVersion, SourceDir: sourceDir, ExtraEnv: buildEnv, ExtraConfigureFlags: depFlags, InstallPrefix: depPrefix, Verbose: verbose, Jobs: jobs})
		if err != nil {
			emit("error", fmt.Sprintf("Build failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge build %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("install", fmt.Sprintf("Installing %s@%s → %s", dep.Name, depVersion, depPrefix))
		if err := s.forge.Install(ctx, domain.InstallRequest{Name: dep.Name, Version: depVersion, BuildDir: buildDir, Prefix: depPrefix, Verbose: verbose, Jobs: jobs}); err != nil {
			emit("error", fmt.Sprintf("Install failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge install %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("done", fmt.Sprintf("✓ %s@%s installed", dep.Name, depVersion))

		res.LocalCppFlags, res.LocalLdFlags, res.LocalPcPaths = s.collectDepFlags(depPrefix, res.LocalCppFlags, res.LocalLdFlags, res.LocalPcPaths)
		res.LibraryPaths = appendUnique(res.LibraryPaths, filepath.Join(depPrefix, "lib"))
		res.BinPaths = appendUnique(res.BinPaths, filepath.Join(depPrefix, "bin"))
	}
	return res, nil
}

// ensureCLIBinary makes sure <prefix>/bin/php is the CLI SAPI. PHP 4's
// `make install` only runs install-sapi (plus build/headers/programs) and never
// its install-cli target, so the CGI SAPI lands at bin/php and every command run
// through a shim would print HTTP headers instead of behaving like a command-line
// interpreter. Every other PHP release installs the CLI itself, in which case the
// two binaries are identical and this is a no-op. A build without a CLI SAPI
// (--disable-cli) has nothing to install and is left alone.
func ensureCLIBinary(sourceDir, prefix string) error {
	cli := filepath.Join(sourceDir, "sapi", "cli", "php")
	installed := filepath.Join(prefix, "bin", "php")

	cliData, err := os.ReadFile(cli)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read CLI binary: %w", err)
	}
	if installedData, err := os.ReadFile(installed); err == nil && bytes.Equal(cliData, installedData) {
		return nil
	}
	if err := os.WriteFile(installed, cliData, 0o755); err != nil {
		return fmt.Errorf("install CLI binary: %w", err)
	}
	return nil
}

// TODO: Hardcoded OS-specific library path
// Description: The if wrapper is only tied to MacOS, it's not allowed to be hardcoded, we must figure out how to handle dynamically across different OS
func libPathEnvName() string {
	if runtime.GOOS == "darwin" {
		return "DYLD_LIBRARY_PATH"
	}
	return "LD_LIBRARY_PATH"
}

// DownloadFailed returns true if any result has an error.
func DownloadFailed(results []DownloadResult) bool {
	for _, r := range results {
		if r.Err != nil {
			return true
		}
	}
	return false
}

func (s *Service) collectDepFlags(prefix string, cppFlags, ldFlags, pcPaths []string) ([]string, []string, []string) {
	cppFlags = appendUnique(cppFlags, "-I"+filepath.Join(prefix, "include"))
	ldFlags = appendUnique(ldFlags, "-L"+filepath.Join(prefix, "lib"))
	ldFlags = appendUnique(ldFlags, "-Wl,-rpath,"+filepath.Join(prefix, "lib"))
	pcPath := filepath.Join(prefix, "lib", "pkgconfig")
	if _, err := os.Stat(pcPath); err == nil {
		pcPaths = appendUnique(pcPaths, pcPath)
	}
	pc64Path := filepath.Join(prefix, "lib64", "pkgconfig")
	if _, err := os.Stat(pc64Path); err == nil {
		pcPaths = appendUnique(pcPaths, pc64Path)
	}
	return cppFlags, ldFlags, pcPaths
}

// TODO: Hardcoded OS-specific system flags
// Description: The if wrapper is only tied to MacOS, it's not allowed to be hardcoded, we must figure out how to handle dynamically across different OS
func collectSystemFlags(cppFlags, ldFlags, pcPaths []string) ([]string, []string, []string) {
	if runtime.GOOS == "darwin" {
		return cppFlags, ldFlags, pcPaths
	}
	cppFlags = appendUnique(cppFlags, "-I/usr/include")
	ldFlags = appendUnique(ldFlags, "-L/usr/lib64")
	ldFlags = appendUnique(ldFlags, "-Wl,-rpath,/usr/lib64")
	pcPaths = appendUnique(pcPaths, "/usr/lib/pkgconfig")
	pcPaths = appendUnique(pcPaths, "/usr/lib64/pkgconfig")
	pcPaths = appendUnique(pcPaths, "/usr/share/pkgconfig")
	return cppFlags, ldFlags, pcPaths
}

// getPHPIniDir returns the directory where PHP expects to find php.ini
// by querying php-config --ini-path. Falls back to prefix/etc if query fails.
func getPHPIniDir(phpPrefix string) string {
	phpConfig := filepath.Join(phpPrefix, "bin", "php-config")
	out, err := exec.Command(phpConfig, "--ini-path").Output()
	if err != nil {
		return filepath.Join(phpPrefix, "etc")
	}
	return strings.TrimSpace(string(out))
}

// InstallExtension builds a single PHP extension from the PHP source tree
// using phpize. The extension source must be at ext/<name>/ inside the
// PHP source tree. After building, it adds extension=<name>.so to php.ini
// and records the extension in the manifest.
func (s *Service) InstallExtension(ctx context.Context, req domain.ExtensionInstallRequest) error {
	phpVersion, extName, phpSourceDir, phpPrefix, jobs := req.PHPVersion, req.ExtName, req.PHPSourceDir, req.PHPPrefix, req.Jobs
	// Safety check: skip if the extension is already built into PHP.
	// This prevents "Module already loaded" warnings for extensions that
	// are compiled into the PHP binary (e.g. iconv in PHP 8.5+).
	phpBin := filepath.Join(phpPrefix, "bin", "php")
	if fi, err := os.Stat(phpBin); err == nil && !fi.IsDir() {
		cmd := exec.Command(phpBin, "-m")
		out, err := cmd.Output()
		if err == nil {
			modules := strings.Split(string(out), "\n")
			for _, mod := range modules {
				mod = strings.TrimSpace(mod)
				if strings.EqualFold(mod, extName) {
					return nil
				}
			}
		}
	}

	extDir := filepath.Join(phpSourceDir, "ext", extName)
	if _, err := os.Stat(extDir); os.IsNotExist(err) {
		return fmt.Errorf("extension %q not found in PHP source tree at %s", extName, extDir)
	}

	configM4 := filepath.Join(extDir, "config.m4")
	config0M4 := filepath.Join(extDir, "config0.m4")
	if _, err := os.Stat(configM4); os.IsNotExist(err) {
		if _, err := os.Stat(config0M4); err == nil {
			if err := os.Rename(config0M4, configM4); err != nil {
				return fmt.Errorf("restore config.m4 for %s: %w", extName, err)
			}
		}
	}

	phpize := filepath.Join(phpPrefix, "bin", "phpize")
	if _, err := os.Stat(phpize); os.IsNotExist(err) {
		return fmt.Errorf("phpize not found at %s (PHP not installed)", phpize)
	}

	phpConfig := filepath.Join(phpPrefix, "bin", "php-config")
	if _, err := os.Stat(phpConfig); os.IsNotExist(err) {
		return fmt.Errorf("php-config not found at %s", phpConfig)
	}

	cmd := exec.CommandContext(ctx, phpize)
	cmd.Dir = extDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("phpize %s: %w\n%s", extName, err, out)
	}

	args := []string{"--with-php-config=" + phpConfig}
	extFlags := s.extensions.GetExtensionConfigureFlags(extName, phpVersion)
	args = append(args, extFlags...)

	configure := exec.CommandContext(ctx, "./configure", args...)
	configure.Dir = extDir
	if out, err := configure.CombinedOutput(); err != nil {
		return fmt.Errorf("configure %s: %w\n%s", extName, err, out)
	}

	make := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	make.Dir = extDir
	if out, err := make.CombinedOutput(); err != nil {
		return fmt.Errorf("make %s: %w\n%s", extName, err, out)
	}

	install := exec.CommandContext(ctx, "make", "install")
	install.Dir = extDir
	if out, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("make install %s: %w\n%s", extName, err, out)
	}

	iniDir := getPHPIniDir(phpPrefix)
	if err := os.MkdirAll(iniDir, 0755); err != nil {
		return fmt.Errorf("create ini dir: %w", err)
	}
	iniPath := filepath.Join(iniDir, "php.ini")
	entry := "extension=" + extName + ".so"
	// Check if the extension line already exists in php.ini to avoid duplicates.
	if data, err := os.ReadFile(iniPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == entry {
				return nil
			}
		}
	}
	f, err := os.OpenFile(iniPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open php.ini: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry + "\n"); err != nil {
		return fmt.Errorf("write php.ini: %w", err)
	}

	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}
	// Deduplicate manifest: skip if this extension is already recorded.
	for _, e := range manifest.Extensions {
		if e.Name == extName {
			return nil
		}
	}
	manifest.Extensions = append(manifest.Extensions, domain.ExtensionState{
		Name:        extName,
		Type:        domain.ExtensionTypeBuiltin,
		InstalledAt: time.Now(),
	})
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return fmt.Errorf("save extension manifest: %w", err)
	}

	return nil
}

// RemoveExtension removes a PHP extension from an installed PHP version.
// It deletes the .so file, removes the extension=<name>.so line from php.ini,
// and updates the extension manifest.
func (s *Service) RemoveExtension(phpVersion, extName, phpPrefix string) error {
	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}

	found := false
	for _, e := range manifest.Extensions {
		if e.Name == extName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("extension %q is not installed for PHP %s", extName, phpVersion)
	}

	extDir := filepath.Join(phpPrefix, "lib", "php", "extensions")
	removed := false
	filepath.Walk(extDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == extName+".so" {
			os.Remove(path)
			removed = true
		}
		return nil
	})

	iniPath := filepath.Join(getPHPIniDir(phpPrefix), "php.ini")
	if data, err := os.ReadFile(iniPath); err == nil {
		lines := strings.Split(string(data), "\n")
		var kept []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "extension="+extName+".so" {
				kept = append(kept, line)
			}
		}
		os.WriteFile(iniPath, []byte(strings.Join(kept, "\n")), 0644)
	}

	var remaining []domain.ExtensionState
	for _, e := range manifest.Extensions {
		if e.Name != extName {
			remaining = append(remaining, e)
		}
	}
	manifest.Extensions = remaining
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return fmt.Errorf("save extension manifest: %w", err)
	}

	if !removed {
		return fmt.Errorf("extension %q .so file not found at %s", extName, extDir)
	}
	return nil
}

func (s *Service) downloadAll(spec downloadSpec) ([]DownloadResult, error) {
	name, version, deps, mustBuild, systemPkgs, static, emit := spec.name, spec.version, spec.deps, spec.mustBuild, spec.systemPkgs, spec.static, spec.emit
	type item struct {
		name    string
		version string
	}
	var items []item
	for _, dep := range deps {
		if dep.Optional && dep.Version == "" {
			continue
		}
		if !shouldDownloadDep(dep, mustBuild, systemPkgs, static) {
			continue
		}
		depVersion, _ := vsn.SplitVersionSpec(dep.Version)
		items = append(items, item{dep.Name, depVersion})
	}
	items = append(items, item{name, version})

	seen := make(map[string]bool)
	var unique []item
	for _, it := range items {
		key := it.name + "@" + it.version
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, it)
	}
	items = unique

	results := make([]DownloadResult, len(items))
	var wg sync.WaitGroup
	var doneCount atomic.Int32
	total := len(items)

	for i, it := range items {
		wg.Add(1)
		go func(idx int, n, v string) {
			defer wg.Done()
			results[idx] = DownloadResult{Name: n, Version: v}

			emit("download", fmt.Sprintf("Downloading %s %s...", n, v))
			regEntry, err := s.reg.Get(n, v)
			if err != nil {
				results[idx].Err = fmt.Errorf("registry resolve %s@%s: %w", n, v, err)
				return
			}
			downloaded, err := s.silo.DownloadURL(regEntry.URL, regEntry.ChecksumType, regEntry.ChecksumValue)
			if err != nil {
				results[idx].Err = fmt.Errorf("download %s@%s: %w", n, v, err)
				return
			}
			results[idx].Downloaded = downloaded

			emit("download", fmt.Sprintf("Extracting %s %s...", n, v))
			archivePath := filepath.Join(cacheDir(), filepath.Base(regEntry.URL))
			sourceDir := s.silo.SourcePath(n, v)
			extracted, err := s.silo.Extract(archivePath, sourceDir)
			if err != nil {
				results[idx].Err = fmt.Errorf("extract %s@%s: %w", n, v, err)
				return
			}
			results[idx].Extracted = extracted
			done := doneCount.Add(1)
			emit("download", fmt.Sprintf("Downloaded %s %s (%d/%d)", n, v, done, total))
		}(i, it.name, it.version)
	}

	wg.Wait()
	var failed []string
	for _, dr := range results {
		if dr.Err != nil {
			failed = append(failed, fmt.Sprintf("%s@%s: %v", dr.Name, dr.Version, dr.Err))
		}
	}
	if len(failed) > 0 {
		return results, fmt.Errorf("download failed:\n  - %s", strings.Join(failed, "\n  - "))
	}
	return results, nil
}

func (s *Service) ResolveVersion(name, constraint string) (string, error) {
	entries, err := s.reg.List(name)
	if err != nil {
		return "", err
	}
	var versions []string
	for _, e := range entries {
		versions = append(versions, e.Version)
	}
	return resolveVersionConstraint(versions, constraint)
}

func isBuildTool(name string) bool {
	switch name {
	case "m4", "autoconf", "automake", "libtool", "perl", "bison", "re2c", "flex", "zig":
		return true
	}
	return false
}

// computeMustBuild returns the set of dependency names that must be built
// from source because their installed system package version does not satisfy
// the constraint required by the build plan. Build tools and optional deps
// without a pinned version are excluded (they come from the system).
func (s *Service) computeMustBuild(deps []domain.Dependency, systemPkgs map[string]domain.SystemPackage, static bool) map[string]bool {
	mustBuild := make(map[string]bool)
	if static || systemPkgs == nil {
		return mustBuild
	}
	for _, dep := range deps {
		if isBuildTool(dep.Name) || (dep.Optional && dep.Version == "") {
			continue
		}
		_, constraint := vsn.SplitVersionSpec(dep.Version)
		if constraint == "" {
			continue
		}
		if sysPkg, ok := systemPkgs[dep.Name]; ok && sysPkg.Installed && sysPkg.Version != "" {
			if !vsn.MatchVersionRange(constraint, sysPkg.Version) {
				mustBuild[dep.Name] = true
			}
		}
	}
	return mustBuild
}

// shouldDownloadDep reports whether a dependency's archive needs to be
// downloaded and extracted at all. Packages that will use the system version
// (or that are never built, like build tools) are skipped, so a fresh install
// only fetches the archives it will actually compile.
func shouldDownloadDep(dep domain.Dependency, mustBuild map[string]bool, systemPkgs map[string]domain.SystemPackage, static bool) bool {
	if isBuildTool(dep.Name) || (dep.Optional && dep.Version == "") {
		return false
	}
	// Static builds and --no-system builds compile every dep from source.
	if static || systemPkgs == nil {
		return true
	}
	if mustBuild[dep.Name] {
		return true
	}
	_, constraint := vsn.SplitVersionSpec(dep.Version)
	if constraint == "" {
		return true
	}
	if sysPkg, ok := systemPkgs[dep.Name]; ok && sysPkg.Installed && sysPkg.Version != "" {
		if vsn.MatchVersionRange(constraint, sysPkg.Version) {
			return false
		}
	}
	return true
}

func isDepInstalled(prefix string) bool {
	info, err := os.Stat(filepath.Join(prefix, "include"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func cacheDir() string {
	return filepath.Join(resolvePHPVRoot(), "caches")
}

func resolvePHPVRoot(parts ...string) string {
	root := os.Getenv("PHPV_ROOT")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".phpv")
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, v := range env {
		if strings.HasPrefix(v, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func resolveVersionConstraint(versions []string, constraint string) (string, error) {
	for _, v := range versions {
		if v == constraint {
			return v, nil
		}
	}
	// Empty constraint: newest available version wins (registry lists are
	// ordered newest-first, and this keeps `phpv server install caddy`
	// working without a version argument).
	if constraint == "" && len(versions) > 0 {
		best := versions[0]
		for _, v := range versions[1:] {
			if vsn.CompareVersions(v, best) > 0 {
				best = v
			}
		}
		return best, nil
	}
	if strings.Count(constraint, ".") == 1 || strings.Count(constraint, ".") == 0 {
		prefix := constraint + "."
		if best := vsn.LatestMatching(versions, prefix); best != "" {
			return best, nil
		}
	}
	return "", fmt.Errorf("no version matching %q found", constraint)
}

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

// ApacheModuleSpec groups the inputs to building a mod_php module.
type ApacheModuleSpec = domain.ApacheModuleSpec

// FPMSpec groups the inputs to ensuring a php-fpm runtime.
type FPMSpec = domain.FPMSpec

// ServerAssembleSpec groups the inputs to assembling a webserver package.
type ServerAssembleSpec = domain.ServerAssembleSpec

// EnsureApacheModule builds libphp.so against the installed httpd (apxs) and
// returns its path. httpdPrefix must point at an installed Apache (httpd)
// package.
func (a *ApacheService) EnsureApacheModule(ctx context.Context, spec ApacheModuleSpec) (string, error) {
	phpVersion, httpdPrefix, verbose, jobs, progress := spec.PHPVersion, spec.HTTPDPrefix, spec.Verbose, spec.Jobs, spec.Progress
	tc := compiler.Toolchain{Name: spec.TC.Name, CC: spec.TC.CC, CXX: spec.TC.CXX, Hint: spec.TC.Hint}
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
			return a.build(ctx, apacheBuildSpec{phpVersion: phpVersion, args: args, env: env, verbose: verbose, jobs: jobs, hash: hash, progress: progress})
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

	if err := a.build(ctx, apacheBuildSpec{phpVersion: phpVersion, args: args, env: env, verbose: verbose, jobs: jobs, hash: hash, progress: progress}); err != nil {
		return "", err
	}
	module := a.ModulePath(phpVersion)
	if module == "" {
		return "", fmt.Errorf("apache module for PHP %s was not produced by the build", phpVersion)
	}
	return module, nil
}

// apacheBuildSpec groups the inputs to an Apache build step.
type apacheBuildSpec struct {
	phpVersion, hash string
	args, env        []string
	verbose          bool
	jobs             int
	progress         ProgressFunc
}

// fpmBuildSpec groups the inputs to an FPM build step.
type fpmBuildSpec struct {
	phpVersion, hash, stagedFrom string
	args, env                    []string
	verbose                      bool
	jobs                         int
	progress                     ProgressFunc
}

// configureSpec groups the inputs to the FPM configure phase.
type configureSpec struct {
	phpVersion string
	args, env  []string
	verbose    bool
}

// serverPkgSpec groups the inputs to building a webserver package.
type serverPkgSpec struct {
	name, version, prefix string
	deps                  []domain.Dependency
	verbose               bool
	jobs                  int
	tc                    compiler.Toolchain
	emit                  ProgressFunc
}

func (a *ApacheService) build(ctx context.Context, spec apacheBuildSpec) error {
	phpVersion, args, env, verbose, jobs, hash, progress := spec.phpVersion, spec.args, spec.env, spec.verbose, spec.jobs, spec.hash, spec.progress
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

// copyTree recursively copies src to dst, preserving file modes and
// modification times. Symlinks are recreated. Used to clone the patched php
// source tree for the FPM variant build without disturbing the CLI build's
// configure state.
//
// Preserving mtimes is load-bearing: make triggers bison regeneration when a
// shipped .y parser looks newer than its pre-generated .c (fresh mtimes in
// directory-iteration order do exactly that), which breaks PHP < 7.1 builds
// against bison 3. With original timestamps the shipped generated parsers
// stay up-to-date and bison is never needed.
func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch info.Mode() & os.ModeType {
	case os.ModeSymlink:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case os.ModeDir:
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return os.Chtimes(dst, mtime(info), mtime(info))
	default:
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return fmt.Errorf("copy %s: %w", filepath.Base(src), err)
		}
		if err := out.Close(); err != nil {
			return err
		}
		return os.Chtimes(dst, mtime(info), mtime(info))
	}
}

func mtime(info os.FileInfo) time.Time {
	return info.ModTime()
}

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
func (f *FPMService) EnsureFPM(ctx context.Context, spec FPMSpec) (string, error) {
	phpVersion, verbose, jobs, stagedFrom, progress := spec.PHPVersion, spec.Verbose, spec.Jobs, spec.StagedFrom, spec.Progress
	tc := compiler.Toolchain{Name: spec.TC.Name, CC: spec.TC.CC, CXX: spec.TC.CXX, Hint: spec.TC.Hint}
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
			return f.build(ctx, fpmBuildSpec{phpVersion: phpVersion, args: args, env: env, verbose: verbose, jobs: jobs, hash: hash, stagedFrom: stagedFrom, progress: progress})
		}); err != nil {
			return "", err
		}
		return f.FPMBinary(phpVersion), nil
	}

	if err := f.build(ctx, fpmBuildSpec{phpVersion: phpVersion, args: args, env: env, verbose: verbose, jobs: jobs, hash: hash, stagedFrom: stagedFrom, progress: progress}); err != nil {
		return "", err
	}
	return f.FPMBinary(phpVersion), nil
}

func (f *FPMService) build(ctx context.Context, spec fpmBuildSpec) error {
	phpVersion, args, env, verbose, jobs, hash, stagedFrom, progress := spec.phpVersion, spec.args, spec.env, spec.verbose, spec.jobs, spec.hash, spec.stagedFrom, spec.progress
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
		if err := f.phaseConfigure(ctx, configureSpec{phpVersion: phpVersion, args: args, env: env, verbose: verbose}); err != nil {
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

func (f *FPMService) phaseConfigure(ctx context.Context, spec configureSpec) error {
	phpVersion, args, env, verbose := spec.phpVersion, spec.args, spec.env, spec.verbose
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

// preparePHPSource materializes a pristine PHP source tree for a SAPI variant
// build: extract the cached tarball into work/src and re-apply published
// patches. Reusing the CLI build's configured tree would carry stale
// .deps/.lo state that breaks make.
func preparePHPSource(deps *Service, phpVersion, work string) error {
	if source.Find(filepath.Join(work, "src"), "php") != "" {
		return nil // already extracted; resume
	}
	// A partial/interrupted extraction leaves a non-empty src dir that
	// Extract would skip (it only checks for content), so clean it first.
	_ = os.RemoveAll(work)
	reg, err := deps.reg.Get("php", phpVersion)
	if err != nil {
		return fmt.Errorf("resolve php %s archive: %w", phpVersion, err)
	}
	archivePath := filepath.Join(cacheDir(), filepath.Base(reg.URL))
	if _, err := os.Stat(archivePath); err != nil {
		if _, err := deps.silo.DownloadURL(reg.URL, reg.ChecksumType, reg.ChecksumValue); err != nil {
			return fmt.Errorf("download php %s for variant build: %w", phpVersion, err)
		}
	}
	if err := os.RemoveAll(work); err != nil {
		return fmt.Errorf("clean variant work dir: %w", err)
	}
	if _, err := deps.silo.Extract(archivePath, filepath.Join(work, "src")); err != nil {
		return fmt.Errorf("extract php source for variant build: %w", err)
	}
	srcPath := source.Find(filepath.Join(work, "src"), "php")
	if srcPath == "" {
		return fmt.Errorf("php %s source not found after extract in %s", phpVersion, work)
	}
	if _, err := deps.patcher.Prepare("php", phpVersion, srcPath); err != nil {
		return fmt.Errorf("patch php source for variant build: %w", err)
	}
	return nil
}

// variantSourcePath finds the extracted PHP source dir inside a work dir.
func variantSourcePath(work string) (string, error) {
	p := source.Find(filepath.Join(work, "src"), "php")
	if p == "" {
		return "", fmt.Errorf("php source not found under %s", work)
	}
	return p, nil
}

// BuildProfile captures how the CLI PHP was configured so SAPI variants
// (php-fpm, the Apache module) can be built from the exact same extension and
// dependency set. Without it, variants built with --disable-all diverge from
// the CLI install (missing pdo_mysql, curl, mbstring, ...).
type BuildProfile struct {
	ConfigureArgs []string          `json:"configure_args"`
	Env           map[string]string `json:"env,omitempty"`
}

const buildProfileFile = ".build.json"

func buildProfilePath(prefix string) string {
	return filepath.Join(prefix, buildProfileFile)
}

// saveBuildProfile writes the profile next to the CLI install.
func saveBuildProfile(prefix string, profile BuildProfile) error {
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	tmp := buildProfilePath(prefix) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, buildProfilePath(prefix))
}

func readBuildProfile(prefix string) (BuildProfile, error) {
	var profile BuildProfile
	data, err := os.ReadFile(buildProfilePath(prefix))
	if err != nil {
		return profile, err
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, err
	}
	return profile, nil
}

// loadCLIProfile returns the persisted CLI build profile, or derives one from
// php-config for installs created before profiles existed.
func loadCLIProfile(phpVersion, cliPrefix, root string) (BuildProfile, error) {
	if profile, err := readBuildProfile(cliPrefix); err == nil && len(profile.ConfigureArgs) > 0 {
		return profile, nil
	}

	phpConfig := filepath.Join(cliPrefix, "bin", "php-config")
	out, err := exec.Command(phpConfig, "--configure-options").Output()
	if err != nil {
		return BuildProfile{}, fmt.Errorf("read CLI build profile for PHP %s: %w", phpVersion, err)
	}
	args, env := parseConfigureOptions(string(out))
	return BuildProfile{ConfigureArgs: args, Env: env}, nil
}

// parseConfigureOptions splits `php-config --configure-options` into configure
// arguments and trailing KEY=VALUE environment tokens.
func parseConfigureOptions(raw string) (args []string, env map[string]string) {
	env = map[string]string{}
	for _, token := range strings.Fields(raw) {
		if token == "" {
			continue
		}
		if !strings.HasPrefix(token, "-") {
			if key, value, ok := strings.Cut(token, "="); ok && key != "" {
				env[key] = value
				continue
			}
		}
		args = append(args, token)
	}
	return args, env
}

// Hash fingerprints the profile for a SAPI so stale variant builds can be
// detected and rebuilt automatically.
func (p BuildProfile) Hash(sapi string) string {
	h := sha256.New()
	fmt.Fprintf(h, "sapi=%s\n", sapi)
	for _, arg := range p.ConfigureArgs {
		fmt.Fprintln(h, arg)
	}
	keys := make([]string, 0, len(p.Env))
	for key := range p.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "PATH" {
			// PATH only affects tool lookup, not the built configuration;
			// excluding it avoids spurious rebuilds when the shell changes.
			continue
		}
		fmt.Fprintf(h, "%s=%s\n", key, p.Env[key])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// VariantArgs rewrites a CLI profile for another SAPI: only the prefix and the
// SAPI flags change, every extension/dependency flag is preserved.
func (p BuildProfile) VariantArgs(sapi, prefix, httpdPrefix string) ([]string, error) {
	out := make([]string, 0, len(p.ConfigureArgs)+4)
	for _, arg := range p.ConfigureArgs {
		switch {
		case strings.HasPrefix(arg, "--prefix="),
			arg == "--enable-cli", arg == "--disable-cli",
			arg == "--enable-cgi", arg == "--disable-cgi",
			arg == "--enable-fpm", arg == "--disable-fpm",
			arg == "--enable-phpdbg", arg == "--disable-phpdbg",
			arg == "--enable-embed", arg == "--disable-embed",
			strings.HasPrefix(arg, "--with-apxs2="):
			continue
		}
		out = append(out, arg)
	}
	out = append(out, "--prefix="+prefix)

	switch sapi {
	case "fpm":
		out = append(out, "--enable-fpm", "--disable-cli", "--disable-cgi", "--disable-phpdbg", "--disable-embed")
	case "modphp":
		if httpdPrefix == "" {
			return nil, fmt.Errorf("apache prefix is required for the mod_php SAPI")
		}
		apxs := filepath.Join(httpdPrefix, "bin", "apxs")
		if _, err := os.Stat(apxs); err != nil {
			return nil, fmt.Errorf("apxs not found at %s (install apache first: phpv server install apache)", apxs)
		}
		out = append(out,
			"--with-apxs2="+apxs,
			"--disable-cli", "--disable-cgi", "--disable-fpm", "--disable-phpdbg", "--disable-embed",
		)
	default:
		return nil, fmt.Errorf("unsupported PHP SAPI %q (want fpm or modphp)", sapi)
	}
	return out, nil
}

// cliExtensionDir resolves the CLI install's dynamically loadable extension
// directory. The Apache module bakes this in via EXTENSION_DIR so php.ini's
// `extension=` lines load at startup (php_admin_value is applied too late).
func cliExtensionDir(cliPrefix string) string {
	phpConfig := filepath.Join(cliPrefix, "bin", "php-config")
	if out, err := exec.Command(phpConfig, "--extension-dir").Output(); err == nil {
		if value := strings.TrimSpace(string(out)); value != "" {
			return value
		}
	}
	matches, _ := filepath.Glob(filepath.Join(cliPrefix, "lib", "php", "extensions", "*"))
	for _, match := range matches {
		if st, err := os.Stat(match); err == nil && st.IsDir() {
			return match
		}
	}
	return ""
}

// ApplyEnv layers the profile environment over a base environment.
func (p BuildProfile) ApplyEnv(base []string) []string {
	env := base
	keys := make([]string, 0, len(p.Env))
	for key := range p.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = setEnvVar(env, key, p.Env[key])
	}
	return env
}

// profileEnv extracts the build-relevant variables from an os.Environ slice.
func profileEnv(env []string) map[string]string {
	out := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		switch key {
		case "CC", "CXX", "CFLAGS", "CXXFLAGS", "CPPFLAGS", "LDFLAGS",
			"PKG_CONFIG_PATH", "PATH", "LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH":
			out[key] = value
		}
	}
	return out
}

// resolveDepPlaceholders replaces {{dep:NAME}} placeholders in configure flags
// with the install prefix of the named dependency, and {{source:NAME}} with
// the extracted source directory of the named dependency. This is used by the
// patcher to inject dependency paths (e.g., curl's --with-openssl={{dep:openssl}})
// and by packages that compile a library from its source tree rather than an
// install prefix (e.g., nginx's --with-pcre=<pcre2 source dir>).
//
// The placeholder is resolved for every dependency that appears in the build
// plan, but only if the dependency was actually built locally (not when
// satisfied by a system package). If the local install directory doesn't exist,
// the placeholder is removed entirely, letting the package's configure script
// find the system version via pkg-config or default search paths.
func (s *Service) resolveDepPlaceholders(flags []string, deps []domain.Dependency) []string {
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		resolved := flag
		skip := false
		for _, dep := range deps {
			depVersion := dep.Version
			if idx := strings.Index(depVersion, "|"); idx != -1 {
				depVersion = depVersion[:idx]
			}
			if depVersion == "" {
				continue
			}
			prefix := s.silo.PackagePrefix(dep.Name, depVersion)
			// Check if the dependency was actually built locally
			if fi, err := os.Stat(prefix); err != nil || !fi.IsDir() {
				continue
			}
			resolved = strings.ReplaceAll(resolved, "{{dep:"+dep.Name+"}}", prefix)
			// {{source:NAME}} must point at the actual buildable source dir
			// (packages that compile a library from its source tree, like
			// nginx's --with-pcre=<pcre2 source>, need a directory containing
			// configure — the extract root only holds the nested archive dir).
			srcDir := s.silo.SourcePath(dep.Name, depVersion)
			if located := source.Find(srcDir, dep.Name); located != "" {
				srcDir = located
			}
			resolved = strings.ReplaceAll(resolved, "{{source:"+dep.Name+"}}", srcDir)
		}
		if strings.Contains(resolved, "{{dep:") || strings.Contains(resolved, "{{source:") {
			// A placeholder survived without a matching locally-built dep:
			// drop the flag and let configure find the system version.
			skip = true
		}
		if !skip {
			result = append(result, resolved)
		}
	}
	return result
}

// ResolveDepPlaceholders exposes {{dep:NAME}}/{{source:NAME}} resolution
// (see resolveDepPlaceholders) for FPM variant builds, which reuse the
// placeholder syntax in their configure flags.
func (s *Service) ResolveDepPlaceholders(flags []string, deps []domain.Dependency) []string {
	return s.resolveDepPlaceholders(flags, deps)
}

// resolveDependencyFlags resolves package-relative configure flags
// (e.g. --with-openssl, --with-zlib, --with-libxml) to absolute paths
// pointing at the locally-built dependency install prefix. Without this
// resolution, PHP's ./configure would use the system OpenSSL/zlib/libxml2
// (which is incompatible with PHP 7.4 for OpenSSL 3.x) instead of the
// version that phpv just built from source.
//
// For each recognized flag, the function looks up the matching dep in the
// build plan, resolves its install prefix via silo.PackagePrefix, and
// replaces the flag with the absolute-path variant. Falls back to
// scanning the dep dir for any installed version if the exact pin is
// not found. Falls back to keeping the original flag if no local build
// is available (letting PHP use the system library).
func (s *Service) resolveDependencyFlags(name, phpVersion string, flags []string, deps []domain.Dependency) []string {
	result := make([]string, 0, len(flags))
	var sslFlag string
	for _, flag := range flags {
		switch {
		case flag == "--with-openssl" || flag == "--with-ssl" ||
			strings.HasPrefix(flag, "--with-openssl=") || strings.HasPrefix(flag, "--with-ssl="):
			path := ""
			if idx := strings.Index(flag, "="); idx >= 0 {
				path = flag[idx+1:]
			}
			if path == "" {
				path = s.findDepPrefix(deps, "openssl", "include/openssl")
			}
			if path != "" {
				opensslFlag := "--with-openssl"
				if name != "php" {
					opensslFlag = "--with-ssl"
				}
				sslFlag = opensslFlag + "=" + path
			} else {
				sslFlag = flag
			}

		case flag == "--with-zlib" && (name == "libxml2" || name == "curl"):
			if path := s.findDepPrefix(deps, "zlib", "include/zlib.h"); path != "" {
				result = append(result, "--with-zlib="+path)
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-libxml") || strings.HasPrefix(flag, "--enable-libxml"):
			if path := s.findDepPrefix(deps, "libxml2", "lib/pkgconfig/libxml-2.0.pc"); path != "" {
				if strings.HasPrefix(flag, "--enable-libxml") {
					result = append(result, flag)
					result = append(result, "--with-libxml-dir="+path)
				} else {
					result = append(result, flag+"="+path)
				}
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-curl"):
			if path := s.findDepPrefix(deps, "curl", "lib/libcurl.so"); path != "" {
				result = append(result, "--with-curl="+path)
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-pdo-pgsql"):
			if flag == "--with-pdo-pgsql" || flag == "--with-pdo-pgsql=yes" {
				wrapperPath := filepath.Join(s.silo.PackagePrefix("php", phpVersion), "wrapper")
				pgConfigPath := filepath.Join(wrapperPath, "bin", "pg_config")
				if fi, err := os.Stat(pgConfigPath); err == nil && !fi.IsDir() {
					result = append(result, "--with-pdo-pgsql="+wrapperPath)
				} else {
					result = append(result, flag)
				}
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-pgsql"):
			if flag == "--with-pgsql" || flag == "--with-pgsql=yes" {
				wrapperPath := filepath.Join(s.silo.PackagePrefix("php", phpVersion), "wrapper")
				pgConfigPath := filepath.Join(wrapperPath, "bin", "pg_config")
				if fi, err := os.Stat(pgConfigPath); err == nil && !fi.IsDir() {
					result = append(result, "--with-pgsql="+wrapperPath)
				} else {
					result = append(result, flag)
				}
			} else {
				result = append(result, flag)
			}

		case flag == "--enable-intl":
			result = append(result, flag)
			if path := s.resolveICUPath(deps); path != "" {
				result = append(result, "--with-icu-dir="+path)
			}

		case flag == "--with-iconv":
			if runtime.GOOS == "darwin" {
				brewIconv := "/opt/homebrew/opt/libiconv"
				if fi, err := os.Stat(filepath.Join(brewIconv, "include", "iconv.h")); err == nil && !fi.IsDir() {
					result = append(result, "--with-iconv="+brewIconv)
				} else {
					result = append(result, flag)
				}
			} else {
				result = append(result, flag)
			}

		default:
			result = append(result, flag)
		}
	}
	if sslFlag != "" {
		result = append(result, sslFlag)
	}
	return result
}

// findDepPrefix locates the install prefix for a dependency by looking up
// the exact version in the build plan's dep list, then falling back to
// scanning the dep dir for installed versions that satisfy the dep's version
// constraint. Returns "" if no local build is available, signaling the caller
// to keep the original flag.
//
// The fallback is constrained because an installed version that is newer than
// the one required by the current build may have been compiled against a
// different ABI (e.g. curl linked against system OpenSSL 3.x while the
// current PHP needs local OpenSSL 1.1.1w). Using only versions that match the
// plan's constraint preserves ABI consistency.
func (s *Service) findDepPrefix(deps []domain.Dependency, depName, sentinelPath string) string {
	var constraint string
	for _, dep := range deps {
		if dep.Name != depName {
			continue
		}
		ver := dep.Version
		if idx := strings.Index(ver, "|"); idx != -1 {
			ver = ver[:idx]
			constraint = dep.Version[idx+1:]
		}
		if ver == "" {
			continue
		}
		prefix := s.silo.PackagePrefix(depName, ver)
		if fi, err := os.Stat(prefix); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(prefix, sentinelPath)); err == nil {
				return prefix
			}
		}
	}
	depDir := filepath.Join(resolvePHPVRoot(), "packages", depName)
	if entries, err := os.ReadDir(depDir); err == nil {
		var candidates []string
		for _, entry := range entries {
			candidate := filepath.Join(depDir, entry.Name())
			if _, err := os.Stat(filepath.Join(candidate, sentinelPath)); err == nil {
				candidates = append(candidates, entry.Name())
			}
		}
		if constraint != "" {
			filtered := candidates[:0]
			for _, c := range candidates {
				if vsn.MatchVersionRange(constraint, c) {
					filtered = append(filtered, c)
				}
			}
			candidates = filtered
		}
		if len(candidates) > 0 {
			sort.Slice(candidates, func(i, j int) bool {
				return vsn.CompareVersions(candidates[i], candidates[j]) > 0
			})
			return filepath.Join(depDir, candidates[0])
		}
	}
	return ""
}

// resolveICUPath finds the bundled ICU installation directory. ICU is
// required by the intl extension and must be passed via --with-icu-dir
// to PHP's configure. Searches the dependency directory for any ICU
// version that contains include/unicode/urename.h.
func (s *Service) resolveICUPath(deps []domain.Dependency) string {
	return s.findDepPrefix(deps, "icu", "include/unicode/urename.h")
}

// ServerPackageName maps the user-facing server name (domain.ServerCaddy,
// domain.ServerNginx, domain.ServerApache) to its registry package name.
// Apache is known to users as "apache" but its upstream project and archive
// are "httpd".
func ServerPackageName(server string) string {
	if server == domain.ServerApache {
		return "httpd"
	}
	return server
}

// AssembleServer installs a webserver package (caddy, nginx, httpd) into its
// silo prefix. It reuses the regular pipeline: resolve → download → build
// deps → configure/make/install, with server-specific handling:
//
//   - caddy ships a prebuilt static binary: no forge build, just copy the
//     binary out of the extracted archive into <prefix>/bin;
//   - nginx always builds pcre2 and zlib from source so the generated
//     configuration never depends on system library variance;
//   - httpd vendors apr/apr-util into srclib/ at configure time
//     (--with-included-apr), so those deps are downloaded and copied but not
//     built standalone.
//
// Returns the install prefix. Already-installed servers short-circuit.
func (s *Service) AssembleServer(ctx context.Context, spec ServerAssembleSpec) (string, error) {
	name, version, verbose, jobs, progress := spec.Name, spec.Version, spec.Verbose, spec.Jobs, spec.Progress
	tc := compiler.Toolchain{Name: spec.TC.Name, CC: spec.TC.CC, CXX: spec.TC.CXX, Hint: spec.TC.Hint}
	emit := func(stage, msg string) {
		if verbose {
			fmt.Printf("[%s] %s\n", stage, msg)
		}
		if progress != nil {
			progress(stage, msg)
		}
	}

	pkgName := ServerPackageName(name)
	emit("resolve", fmt.Sprintf("Resolving %s version...", name))
	exact, err := s.ResolveVersion(pkgName, version)
	if err != nil {
		return "", fmt.Errorf("resolve %s version: %w", name, err)
	}
	prefix := s.silo.PackagePrefix(name, exact)
	if s.serverInstalled(name, pkgName, exact, prefix) {
		emit("skip", fmt.Sprintf("Already installed %s@%s", name, exact))
		return prefix, nil
	}
	if jobs < 1 {
		jobs = 4
	}

	if err := s.silo.MarkInProgress(name, exact); err != nil {
		return "", fmt.Errorf("mark in-progress: %w", err)
	}
	var completed bool
	defer func() {
		if !completed {
			if ctx.Err() != nil {
				s.silo.MarkInterrupted(name, exact)
			} else {
				s.silo.MarkFailed(name, exact)
			}
		}
	}()

	plan, err := s.graph.GetBuildPlan(pkgName, exact, nil, tc.Name)
	if err != nil {
		return "", fmt.Errorf("resolve build plan for %s@%s: %w", name, exact, err)
	}

	mustBuild := map[string]bool{}
	if name == "nginx" {
		// Deterministic nginx builds: never mix system pcre2/zlib.
		mustBuild["pcre2"] = true
		mustBuild["zlib"] = true
	}

	emit("download", fmt.Sprintf("Downloading %s %s...", pkgName, exact))
	if _, err := s.downloadAll(downloadSpec{name: pkgName, version: exact, deps: plan.Deps, mustBuild: mustBuild, systemPkgs: nil, static: false, emit: emit}); err != nil {
		return "", err
	}

	switch pkgName {
	case "caddy":
		emit("install", fmt.Sprintf("Installing %s %s...", name, exact))
		if err := s.installCaddyBinary(pkgName, exact, prefix); err != nil {
			return "", err
		}
	case "nginx":
		// nginx compiles pcre2 and zlib from their source trees itself
		// (--with-pcre=<src>, --with-zlib=<src>), so no standalone dep
		// builds — only download/extract, then configure nginx.
		if err := s.buildServerPackage(ctx, serverPkgSpec{name: pkgName, version: exact, deps: plan.Deps, prefix: prefix, verbose: verbose, jobs: jobs, tc: tc, emit: emit}); err != nil {
			return "", err
		}
	case "httpd":
		// apr/apr-util are vendored into the httpd source tree, not built
		// standalone. pcre2 IS built standalone (httpd invokes pcre2-config
		// from the install prefix). Filter apr/apr-util out of the dep
		// build list, build the rest, vendor the apr sources, then build.
		var httpdDeps []domain.Dependency
		for _, dep := range plan.Deps {
			if dep.Name == "apr" || dep.Name == "apr-util" {
				continue
			}
			httpdDeps = append(httpdDeps, dep)
		}
		if _, err := s.buildDependencies(ctx, depBuildSpec{deps: httpdDeps, mustBuild: mustBuild, systemPkgs: nil, static: false, verbose: verbose, jobs: jobs, tc: tc, emit: emit}); err != nil {
			return "", err
		}
		aprVers := planDepVersions(plan.Deps, "apr", "apr-util")
		if err := s.vendorAPR(exact, aprVers[0], aprVers[1], emit); err != nil {
			return "", err
		}
		if err := s.buildServerPackage(ctx, serverPkgSpec{name: pkgName, version: exact, deps: httpdDeps, prefix: prefix, verbose: verbose, jobs: jobs, tc: tc, emit: emit}); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported server package: %s", name)
	}

	completed = true
	if err := s.silo.MarkComplete(name, exact); err != nil {
		return "", fmt.Errorf("mark complete: %w", err)
	}
	emit("done", fmt.Sprintf("%s %s installed at %s", name, exact, prefix))
	return prefix, nil
}

// serverInstalled reports whether a webserver is already installed. Servers
// do not ship an include/ tree, so the shared dependency check does not apply;
// prefer the recorded install state and fall back to the binary for legacy
// installs that predate state markers.
func (s *Service) serverInstalled(name, pkgName, version, prefix string) bool {
	if state, err := s.silo.GetState(name, version); err == nil && state == domain.StateInstalled {
		return true
	}
	return FindServerBinary(prefix, pkgName) != ""
}

// buildServerPackage runs configure/make/install for a source-built server.
func (s *Service) buildServerPackage(ctx context.Context, spec serverPkgSpec) error {
	name, version, deps, prefix, verbose, jobs, tc, emit := spec.name, spec.version, spec.deps, spec.prefix, spec.verbose, spec.jobs, spec.tc, spec.emit
	sourceDir := s.silo.SourcePath(name, version)
	srcPath := source.Find(sourceDir, name)
	if srcPath == "" {
		return fmt.Errorf("could not find source directory in %s", sourceDir)
	}

	flags := s.graph.GetConfigureFlags(name, version)
	flags = s.resolveDepPlaceholders(flags, deps)

	emit("build", fmt.Sprintf("Building %s %s (configure + make)...", name, version))
	buildDir, _, err := s.forge.Build(ctx, domain.BuildRequest{Name: name, Version: version, SourceDir: srcPath, ExtraEnv: tc.Env(), ExtraConfigureFlags: flags, InstallPrefix: prefix, Verbose: verbose, Jobs: jobs})
	if err != nil {
		return fmt.Errorf("build %s@%s: %w", name, version, err)
	}
	emit("install", fmt.Sprintf("Installing %s@%s → %s", name, version, prefix))
	if err := s.forge.Install(ctx, domain.InstallRequest{Name: name, Version: version, BuildDir: buildDir, Prefix: prefix, Verbose: verbose, Jobs: jobs}); err != nil {
		return fmt.Errorf("install %s@%s: %w", name, version, err)
	}
	return nil
}

// installCaddyBinary extracts the prebuilt caddy binary from the downloaded
// archive into <prefix>/bin/caddy. Caddy ships no build files, so the plain
// extract dir is scanned instead of source.Find (which requires configure/
// Makefile markers).
func (s *Service) installCaddyBinary(name, version, prefix string) error {
	srcPath := s.silo.SourcePath(name, version)
	if fi, err := os.Stat(srcPath); err != nil || !fi.IsDir() {
		return fmt.Errorf("caddy archive not extracted at %s", srcPath)
	}
	bin := FindServerBinary(srcPath, "caddy")
	if bin == "" {
		return fmt.Errorf("caddy binary not found in extracted archive at %s", srcPath)
	}
	if err := os.MkdirAll(filepath.Join(prefix, "bin"), 0o755); err != nil {
		return fmt.Errorf("create prefix: %w", err)
	}
	dst := filepath.Join(prefix, "bin", "caddy")
	if err := copyFileMode(bin, dst, 0o755); err != nil {
		return fmt.Errorf("install caddy binary: %w", err)
	}
	return nil
}

// vendorAPR copies the downloaded apr/apr-util source trees into the httpd
// source tree as srclib/apr and srclib/apr-util, as required by
// --with-included-apr.
func (s *Service) vendorAPR(httpdVersion string, aprVersion, aprUtilVersion string, emit func(stage, msg string)) error {
	httpdSrc := source.Find(s.silo.SourcePath("httpd", httpdVersion), "httpd")
	if httpdSrc == "" {
		return fmt.Errorf("httpd source not found under %s", s.silo.SourcePath("httpd", httpdVersion))
	}
	pairs := []struct{ pkg, ver, dest string }{
		{"apr", aprVersion, "srclib/apr"},
		{"apr-util", aprUtilVersion, "srclib/apr-util"},
	}
	for _, p := range pairs {
		src := source.Find(s.silo.SourcePath(p.pkg, p.ver), p.pkg)
		if src == "" {
			return fmt.Errorf("%s %s source not found (httpd build needs it vendored)", p.pkg, p.ver)
		}
		dst := filepath.Join(httpdSrc, p.dest)
		if _, err := os.Stat(filepath.Join(dst, "configure")); err == nil {
			continue // already vendored
		}
		emit("prepare", fmt.Sprintf("Vendoring %s → %s", p.pkg, p.dest))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := copyTree(src, dst); err != nil {
			return fmt.Errorf("vendor %s: %w", p.pkg, err)
		}
	}
	return nil
}

// planDepVersions returns the exact pinned versions of the named deps from
// a build plan's dependency list ("" for names not present).
func planDepVersions(deps []domain.Dependency, names ...string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		for _, d := range deps {
			if d.Name != n {
				continue
			}
			v, _ := vsn.SplitVersionSpec(d.Version)
			out[i] = v
		}
	}
	return out
}

// FindServerBinary locates an executable inside an extracted archive,
// checking the root, bin/ and a single nested top-level directory.
func FindServerBinary(dir, name string) string {
	candidates := []string{
		filepath.Join(dir, name),
		filepath.Join(dir, "bin", name),
		filepath.Join(dir, "sbin", name),
	}
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				candidates = append(candidates,
					filepath.Join(dir, e.Name(), name),
					filepath.Join(dir, e.Name(), "bin", name),
				)
			}
		}
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return c
		}
	}
	return ""
}

func copyFileMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// latestLocalVersion returns the newest installed version directory name for
// a package under packages/<name>/, or "" when none exist.
func latestLocalVersion(pkgDir string) string {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return ""
	}
	var best string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if best == "" || vsn.CompareVersions(e.Name(), best) > 0 {
			best = e.Name()
		}
	}
	return best
}

const variantMarkerFile = ".phpv-variant.json"

// VariantMarker records which SAPI/profile a cached variant binary was built
// with, so changing the CLI build or the SAPI triggers an automatic rebuild.
type VariantMarker struct {
	SAPI        string `json:"sapi"`
	PHPVersion  string `json:"php_version"`
	ProfileHash string `json:"profile_hash"`
}

func variantMarkerPath(prefix string) string {
	return filepath.Join(prefix, variantMarkerFile)
}

func readVariantMarker(prefix string) (VariantMarker, bool) {
	var marker VariantMarker
	data, err := os.ReadFile(variantMarkerPath(prefix))
	if err != nil {
		return marker, false
	}
	if err := json.Unmarshal(data, &marker); err != nil {
		return marker, false
	}
	return marker, true
}

func writeVariantMarker(prefix string, marker VariantMarker) error {
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	tmp := variantMarkerPath(prefix) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, variantMarkerPath(prefix))
}

func variantMatches(prefix, sapi, phpVersion, profileHash string) bool {
	marker, ok := readVariantMarker(prefix)
	return ok &&
		marker.SAPI == sapi &&
		marker.PHPVersion == phpVersion &&
		marker.ProfileHash == profileHash
}

// safeRebuild replaces a cached prefix without losing a working binary when
// the new build fails: the old prefix is renamed to .bak, restored on error,
// and removed on success.
func safeRebuild(prefix string, build func() error) error {
	if err := os.MkdirAll(filepath.Dir(prefix), 0o755); err != nil {
		return err
	}
	bak := prefix + ".bak"
	_ = os.RemoveAll(bak)
	hadOld := false
	if _, err := os.Stat(prefix); err == nil {
		if err := os.Rename(prefix, bak); err != nil {
			return fmt.Errorf("move old build aside: %w", err)
		}
		hadOld = true
	}
	if err := build(); err != nil {
		_ = os.RemoveAll(prefix)
		if hadOld {
			_ = os.Rename(bak, prefix)
		}
		return err
	}
	_ = os.RemoveAll(bak)
	return nil
}
