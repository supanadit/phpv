package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

	var depLibraryPaths []string
	var depCppFlags []string
	var depLdFlags []string
	var depPkgConfigPaths []string

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

func (s *bundlerRepository) Orchestrate(name, exactVersion string, forceCompiler string, extensions []string, fresh bool) (domain.Forge, error) {
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

	var deps []domain.Dependency
	for _, dep := range depMap {
		deps = append(deps, dep)
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
