package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/flagresolver"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type bundlerRepository struct {
	assemblerSvc    *assembler.AssemblerService
	advisorSvc      *advisor.Service
	forgeSvc        *forge.Service
	downloadSvc     *download.Service
	unloadSvc       *unload.Service
	sourceSvc       *source.Service
	patternSvc      pattern.PatternRepository
	flagResolverSvc *flagresolver.Service
	silo            *domain.Silo
	siloRepo        *SiloRepository
	fs              afero.Fs
	jobs            int
	verbose         bool
	logger          utils.Logger
	extensions      []string
}

func NewBundlerRepository(cfg bundler.BundlerServiceConfig, patternSvc pattern.PatternRepository) bundler.BundlerRepository {

	jobs := cfg.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	assemblerSvc := assembler.NewAssemblerServiceWithRepo(cfg.Assembler)
	advisorSvc := advisor.NewAdvisorService(cfg.Advisor)
	flagResolverSvc := flagresolver.NewService(cfg.FlagResolverRepo)
	forgeSvc := forge.NewService(cfg.Forge, flagResolverSvc)
	downloadSvc := download.NewService(cfg.Download)
	unloadSvc := unload.NewService(cfg.Unload)
	sourceSvc := source.NewService(cfg.Source)

	var siloRepo *SiloRepository
	if cfg.SiloRepo != nil {
		siloRepo = cfg.SiloRepo.(*SiloRepository)
	}

	if cfg.Logger != nil {
		if diskForge, ok := cfg.Forge.(*ForgeRepository); ok {
			diskForge.SetLogger(cfg.Logger)
		}
	}

	return &bundlerRepository{
		assemblerSvc:    assemblerSvc,
		advisorSvc:      advisorSvc,
		forgeSvc:        forgeSvc,
		downloadSvc:     downloadSvc,
		unloadSvc:       unloadSvc,
		sourceSvc:       sourceSvc,
		patternSvc:      patternSvc,
		flagResolverSvc: flagResolverSvc,
		silo:            cfg.Silo,
		siloRepo:        siloRepo,
		fs:              afero.NewOsFs(),
		jobs:            jobs,
		verbose:         cfg.Verbose,
		logger:          cfg.Logger,
		extensions:      cfg.Extensions,
	}
}

func (s *bundlerRepository) Install(version string, compiler string, extensions []string, fresh bool) (domain.Forge, error) {
	exactVersion, err := s.resolvePHPVersion(version)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("[bundler] failed to resolve PHP version %q: %w", version, err)
	}
	return s.Orchestrate("php", exactVersion, compiler, extensions, fresh)
}

func (s *bundlerRepository) Rebuild(version string, compiler string, extensions []string) (domain.Forge, error) {
	exactVersion, err := s.resolvePHPVersion(version)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("[bundler] failed to resolve PHP version %q: %w", version, err)
	}

	outputPath := utils.PHPOutputPath(s.silo, exactVersion)
	phpBinary := filepath.Join(outputPath, "bin", "php")
	if exists, _ := afero.Exists(s.fs, phpBinary); !exists {
		return domain.Forge{}, fmt.Errorf("[bundler] PHP %s is not installed. Use 'phpv install %s' first", exactVersion, version)
	}

	// Same validation gates as Install
	if len(extensions) > 0 {
		if err := s.flagResolverSvc.ValidateExtensions(extensions, exactVersion); err != nil {
			return domain.Forge{}, fmt.Errorf("invalid extension: %w", err)
		}

		_, added := s.flagResolverSvc.ExpandImplied(extensions)
		if len(added) > 0 {
			s.logInfo("ℹ  Auto-added required dependencies: %s", strings.Join(added, ", "))
		}

		conflicts, conflictPairs, _ := s.flagResolverSvc.CheckExtensionConflicts(extensions)
		if len(conflicts) > 0 {
			s.logError("✗ Conflicting extensions: %s", strings.Join(conflicts, ", "))
			for _, pair := range conflictPairs {
				s.logError("  %s conflicts with %s", pair[0], pair[1])
			}
			return domain.Forge{}, fmt.Errorf("conflicting extensions: %s", strings.Join(conflicts, ", "))
		}
	}

	// Resolve all dependency levels
	levels, err := s.assemblerSvc.GetDependencyLevels("php", exactVersion)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	extDeps := s.resolveExtensionDependencies(extensions, exactVersion)
	if len(extDeps) > 0 {
		levels = append([][]domain.Dependency{extDeps}, levels...)
	}

	// Build only missing dependencies (existing ones skip via advisor cache)
	depLibraryPaths, depCppFlags, depLdFlags, depPkgConfigPaths := s.buildMissingDeps(levels, exactVersion, compiler)

	// Rebuild PHP with the full dependency context
	if err := s.buildPHP("php", exactVersion, extensions, depLibraryPaths, depCppFlags, depLdFlags, depPkgConfigPaths, compiler, true); err != nil {
		return domain.Forge{}, fmt.Errorf("[bundler] failed to rebuild PHP: %w", err)
	}

	outputPath = utils.PHPOutputPath(s.silo, exactVersion)
	depLibraryPaths = append(depLibraryPaths, filepath.Join(outputPath, "lib"))

	return domain.Forge{
		Prefix: outputPath,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(depLibraryPaths, ":"),
		},
	}, nil
}

// buildMissingDeps builds only dependencies that aren't already cached,
// and returns the accumulated library/include paths.
func (s *bundlerRepository) buildMissingDeps(levels [][]domain.Dependency, exactVersion, compiler string) ([]string, []string, []string, []string) {
	// Start by collecting paths from already-installed deps
	libPaths, cppFlags, ldFlags, pkgPath := s.collectInstalledDependencyPaths(exactVersion)

	for _, level := range levels {
		for _, dep := range level {
			depVersion := dep.Version
			if idx := strings.Index(dep.Version, "|"); idx != -1 {
				depVersion = dep.Version[:idx]
			}

			// buildPackage will "skip" if already cached
			depInfo, err := s.buildPackage(dep.Name, depVersion, exactVersion, libPaths, cppFlags, ldFlags, pkgPath, compiler)
			if err != nil {
				s.logError("✗ Failed to build %s@%s: %v", dep.Name, depVersion, err)
				continue
			}

			if depInfo != nil && !utils.BuildTools[dep.Name] {
				// Only add paths for source-built deps not already collected
				depPath := utils.DependencyPath(s.silo, exactVersion, dep.Name, depVersion)
				cppFlags = appendUnique(cppFlags, fmt.Sprintf("-I%s/include", depPath))
				ldFlags = appendUnique(ldFlags, fmt.Sprintf("-L%s/lib", depPath))
				ldFlags = appendUnique(ldFlags, fmt.Sprintf("-Wl,-rpath,%s/lib", depPath))
				libPaths = appendUnique(libPaths, filepath.Join(depPath, "lib"))
				if _, err := os.Stat(filepath.Join(depPath, "lib", "pkgconfig")); err == nil {
					pkgPath = appendUnique(pkgPath, filepath.Join(depPath, "lib", "pkgconfig"))
				}
			}
		}
	}
	return libPaths, cppFlags, ldFlags, pkgPath
}

func (s *bundlerRepository) Orchestrate(name, exactVersion string, forceCompiler string, extensions []string, fresh bool) (domain.Forge, error) {
	// Gate: validate extensions BEFORE anything else
	if len(extensions) > 0 {
		if err := s.flagResolverSvc.ValidateExtensions(extensions, exactVersion); err != nil {
			return domain.Forge{}, fmt.Errorf("invalid extension: %w", err)
		}

		_, added := s.flagResolverSvc.ExpandImplied(extensions)
		if len(added) > 0 {
			s.logInfo("ℹ  Auto-added required dependencies: %s", strings.Join(added, ", "))
		}

		conflicts, conflictPairs, _ := s.flagResolverSvc.CheckExtensionConflicts(extensions)
		if len(conflicts) > 0 {
			s.logError("✗ Conflicting extensions: %s", strings.Join(conflicts, ", "))
			for _, pair := range conflictPairs {
				s.logError("  %s conflicts with %s", pair[0], pair[1])
			}
			return domain.Forge{}, fmt.Errorf("conflicting extensions: %s", strings.Join(conflicts, ", "))
		}
	}

	if err := s.siloRepo.MarkInProgress(exactVersion); err != nil {
		return domain.Forge{}, fmt.Errorf("[bundler] failed to mark installation in progress: %w", err)
	}

	levels, err := s.assemblerSvc.GetDependencyLevels(name, exactVersion)
	if err != nil {
		s.siloRepo.MarkFailed(exactVersion)
		return domain.Forge{}, fmt.Errorf("[assembler] failed to resolve dependency levels: %w", err)
	}

	extDeps := s.resolveExtensionDependencies(extensions, exactVersion)
	if len(extDeps) > 0 {
		extLevels := [][]domain.Dependency{extDeps}
		levels = append(extLevels, levels...)
	}

	if fresh {
		if err := s.freshClean(name, exactVersion, levels); err != nil {
			return domain.Forge{}, fmt.Errorf("[bundler] failed to clean existing installation: %w", err)
		}
	}

	var firstErr error

	completed := make(map[string]bool)
	var depLibraryPaths []string
	var depCppFlags []string
	var depLdFlags []string
	var depPkgConfigPaths []string
	var builtDeps []domain.DependencyInfo

	for levelIdx, level := range levels {
		for _, dep := range level {
			if firstErr != nil {
				break
			}

			depVersion := dep.Version
			if idx := strings.Index(dep.Version, "|"); idx != -1 {
				depVersion = dep.Version[:idx]
			}

			depInfo, err := s.buildPackage(dep.Name, depVersion, exactVersion, depLibraryPaths, depCppFlags, depLdFlags, depPkgConfigPaths, forceCompiler)
			if err != nil {
				firstErr = fmt.Errorf("[forge] failed to build %s@%s: %w", dep.Name, depVersion, err)
				break
			}

			completed[dep.Name+"@"+depVersion] = true
			if !utils.BuildTools[dep.Name] && depInfo.BuiltFromSource {
				depPath := utils.DependencyPath(s.silo, exactVersion, dep.Name, depVersion)
				depLibraryPaths = append(depLibraryPaths, filepath.Join(depPath, "lib"))
				depCppFlags = append(depCppFlags, fmt.Sprintf("-I%s/include", depPath))
				depLdFlags = append(depLdFlags, fmt.Sprintf("-L%s/lib", depPath))
				depLdFlags = append(depLdFlags, fmt.Sprintf("-Wl,-rpath,%s/lib", depPath))
				lib64PcPath := filepath.Join(depPath, "lib64", "pkgconfig")
				libPcPath := filepath.Join(depPath, "lib", "pkgconfig")
				if _, err := os.Stat(libPcPath); err == nil {
					depPkgConfigPaths = append(depPkgConfigPaths, libPcPath)
				}
				if _, err := os.Stat(lib64PcPath); err == nil {
					depPkgConfigPaths = append(depPkgConfigPaths, lib64PcPath)
				}
			}
			if depInfo != nil {
				builtDeps = append(builtDeps, *depInfo)
			}
		}

		if firstErr != nil {
			s.siloRepo.MarkFailed(exactVersion)
			s.siloRepo.Rollback(exactVersion)
			return domain.Forge{}, firstErr
		}
		_ = levelIdx
	}

	wrapperMgr := NewVersionWrapperManager(s.fs, s.silo, exactVersion)
	if err := wrapperMgr.Ensure(); err != nil {
		s.logWarn("Warning: failed to ensure wrapper directory: %v", err)
	}

	for _, dep := range builtDeps {
		if !dep.BuiltFromSource || dep.Name == "php" {
			continue
		}
		if utils.BuildTools[dep.Name] {
			continue
		}

		depVersion := dep.Version
		if idx := strings.Index(dep.Version, "|"); idx != -1 {
			depVersion = dep.Version[:idx]
		}

		if err := wrapperMgr.CreateDepLibSymlink(dep.Name, depVersion); err != nil {
			s.logWarn("Warning: failed to create lib symlink for %s: %v", dep.Name, err)
		}
		if err := wrapperMgr.CreateDepIncludeSymlink(dep.Name, depVersion); err != nil {
			s.logWarn("Warning: failed to create include symlink for %s: %v", dep.Name, err)
		}
	}

	if err := wrapperMgr.CreatePgConfigWrapper(""); err != nil {
		s.logWarn("Warning: failed to create pg_config wrapper: %v", err)
	}
	if err := wrapperMgr.CreatePkgConfigWrapper(); err != nil {
		s.logWarn("Warning: failed to create pkg-config wrapper: %v", err)
	}

	if systemLibPath := GetSystemLibPqPath(); systemLibPath != "" {
		if err := wrapperMgr.CreateLibPqWrapper(systemLibPath); err != nil {
			s.logWarn("Warning: failed to create libpq wrapper: %v", err)
		}
	}

	if systemSSLPath := GetSystemOpenSSLLibPath(); systemSSLPath != "" {
		if err := wrapperMgr.AddSystemLib("ssl", systemSSLPath); err != nil {
			s.logWarn("Warning: failed to create ssl wrapper: %v", err)
		}
		if err := wrapperMgr.AddSystemLib("crypto", strings.Replace(systemSSLPath, "libssl.so", "libcrypto.so", 1)); err != nil {
			s.logWarn("Warning: failed to create crypto wrapper: %v", err)
		}
	}

	if !utils.BuildTools["openssl"] {
		depPath := utils.DependencyPath(s.silo, exactVersion, "openssl", "")
		if entries, err := os.ReadDir(depPath); err == nil && len(entries) > 0 {
			entry := entries[len(entries)-1]
			opensslPath := filepath.Join(depPath, entry.Name())
			opensslLib := filepath.Join(opensslPath, "lib")
			if _, err := os.Stat(opensslLib); err == nil {
				depLdFlags = append([]string{
					fmt.Sprintf("-Wl,-rpath,%s", opensslLib),
				}, depLdFlags...)
			}
		}
	}

	if err := s.buildPHP(name, exactVersion, extensions, depLibraryPaths, depCppFlags, depLdFlags, depPkgConfigPaths, forceCompiler, false); err != nil {
		s.siloRepo.MarkFailed(exactVersion)
		s.siloRepo.Rollback(exactVersion)
		return domain.Forge{}, fmt.Errorf("[forge] failed to build PHP: %w", err)
	}

	if err := s.siloRepo.SaveDependencyInfo(exactVersion, builtDeps); err != nil {
		s.logWarn("Warning: failed to save dependency info: %v", err)
	}

	if err := s.siloRepo.MarkComplete(exactVersion); err != nil {
		s.logWarn("Warning: failed to mark installation complete: %v", err)
	}

	outputPath := utils.PHPOutputPath(s.silo, exactVersion)
	depLibraryPaths = append(depLibraryPaths, filepath.Join(outputPath, "lib"))

	return domain.Forge{
		Prefix: outputPath,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(depLibraryPaths, ":"),
		},
	}, nil
}

func (s *bundlerRepository) resolvePHPVersion(constraint string) (string, error) {
	sources, err := s.sourceSvc.GetVersions()
	if err != nil {
		return "", err
	}

	var phpVersions []string
	for _, src := range sources {
		if src.Name == "php" {
			phpVersions = append(phpVersions, src.Version)
		}
	}

	return utils.ResolveVersionConstraint(phpVersions, constraint)
}

func (s *bundlerRepository) resolveExtensionDependencies(extensions []string, phpVersion string) []domain.Dependency {
	if len(extensions) == 0 {
		return nil
	}

	extensions = s.expandImpliedExtensions(extensions)

	depMap := make(map[string]domain.Dependency)
	seen := make(map[string]bool)

	for _, ext := range extensions {
		pkg, version, ok := s.flagResolverSvc.GetExtensionDependencyWithVersion(ext, phpVersion)
		if ok && !seen[pkg] {
			seen[pkg] = true
			depMap[pkg] = domain.Dependency{Name: pkg, Version: version}
		}
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(depMap))
	for key := range depMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var deps []domain.Dependency
	for _, key := range keys {
		deps = append(deps, depMap[key])
	}

	return deps
}

func (s *bundlerRepository) expandImpliedExtensions(extensions []string) []string {
	visited := make(map[string]bool)
	var result []string

	var add func(name string)
	add = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		result = append(result, name)
		if extDef, ok := s.flagResolverSvc.GetExtensionDef(name); ok {
			for _, implied := range extDef.Implied {
				add(implied)
			}
		}
	}

	for _, ext := range extensions {
		add(ext)
	}
	return result
}

func (s *bundlerRepository) freshClean(name, exactVersion string, levels [][]domain.Dependency) error {
	pathsToClean := []string{
		utils.PHPVersionPath(s.silo, exactVersion),
		utils.GetSourcePath(s.silo, name, exactVersion),
	}

	for _, level := range levels {
		for _, dep := range level {
			depVersion := dep.Version
			if idx := strings.Index(dep.Version, "|"); idx != -1 {
				depVersion = dep.Version[:idx]
			}
			sourcePath := utils.GetSourcePath(s.silo, dep.Name, depVersion)
			sourceDirPath := utils.GetSourceDirPath(s.silo, dep.Name, depVersion)
			pathsToClean = append(pathsToClean, sourcePath, sourceDirPath)
		}
	}

	for _, path := range pathsToClean {
		if exists, _ := afero.Exists(s.fs, path); exists {
			if err := s.fs.RemoveAll(path); err != nil {
				return fmt.Errorf("[bundler] failed to remove %s: %w", path, err)
			}
		}
	}

	return nil
}

// collectInstalledDependencyPaths scans the installed dependencies for a PHP version
// and returns the library/include paths needed for a rebuild.
func (s *bundlerRepository) collectInstalledDependencyPaths(exactVersion string) ([]string, []string, []string, []string) {
	depDir := filepath.Join(s.silo.Root, "versions", exactVersion, "dependency")
	entries, err := afero.ReadDir(s.fs, depDir)
	if err != nil {
		return nil, nil, nil, nil
	}

	var libPaths, cppFlags, ldFlags, pkgConfigPaths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip build tools
		if utils.BuildTools[name] {
			continue
		}
		versDir := filepath.Join(depDir, name)
		vers, err := afero.ReadDir(s.fs, versDir)
		if err != nil || len(vers) == 0 {
			continue
		}
		ver := vers[0].Name()
		depPath := filepath.Join(versDir, ver)
		libPaths = append(libPaths, filepath.Join(depPath, "lib"))
		cppFlags = append(cppFlags, fmt.Sprintf("-I%s/include", depPath))
		ldFlags = append(ldFlags, fmt.Sprintf("-L%s/lib", depPath))
		ldFlags = append(ldFlags, fmt.Sprintf("-Wl,-rpath,%s/lib", depPath))
		if _, err := os.Stat(filepath.Join(depPath, "lib", "pkgconfig")); err == nil {
			pkgConfigPaths = append(pkgConfigPaths, filepath.Join(depPath, "lib", "pkgconfig"))
		}
	}
	return libPaths, cppFlags, ldFlags, pkgConfigPaths
}
