package disk

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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

var buildTools = map[string]bool{
	"m4":       true,
	"autoconf": true,
	"automake": true,
	"libtool":  true,
	"perl":     true,
	"bison":    true,
	"flex":     true,
	"re2c":     true,
	"zig":      true,
}

type bundlerRepository struct {
	assemblerSvc    *assembler.AssemblerService
	advisorSvc      *advisor.Service
	forgeSvc        *forge.Service
	downloadSvc     *download.Service
	unloadSvc       *unload.Service
	sourceSvc       *source.Service
	patternRegistry *pattern.PatternRegistry
	flagResolverSvc *flagresolver.Service
	silo            *domain.Silo
	siloRepo        *SiloRepository
	fs              afero.Fs
	jobs            int
	verbose         bool
	logger          utils.Logger
}

func NewBundlerRepository(cfg bundler.BundlerServiceConfig, flagResolverRepo domain.FlagResolverRepository) bundler.BundlerRepository {
	registry := pattern.NewPatternRegistry()
	registry.RegisterPatterns(pattern.DefaultURLPatterns)

	jobs := cfg.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	assemblerSvc := assembler.NewAssemblerServiceWithRepo(cfg.Assembler)
	advisorSvc := advisor.NewAdvisorService(cfg.Advisor)
	flagResolverSvc := flagresolver.NewService(flagResolverRepo)
	forgeSvc := forge.NewService(cfg.Forge, flagResolverSvc)
	downloadSvc := download.NewService(cfg.Download)
	unloadSvc := unload.NewService(cfg.Unload)
	sourceSvc := source.NewService(cfg.Source)

	var siloRepo *SiloRepository
	if cfg.SiloRepo != nil {
		siloRepo = cfg.SiloRepo.(*SiloRepository)
	}

	return &bundlerRepository{
		assemblerSvc:    assemblerSvc,
		advisorSvc:      advisorSvc,
		forgeSvc:        forgeSvc,
		downloadSvc:     downloadSvc,
		unloadSvc:       unloadSvc,
		sourceSvc:       sourceSvc,
		patternRegistry: registry,
		flagResolverSvc: flagResolverSvc,
		silo:            cfg.Silo,
		siloRepo:        siloRepo,
		fs:              afero.NewOsFs(),
		jobs:            jobs,
		verbose:         cfg.Verbose,
		logger:          cfg.Logger,
	}
}

func (s *bundlerRepository) Install(version string, compiler string, fresh bool) (domain.Forge, error) {
	exactVersion, err := s.resolvePHPVersion(version)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("failed to resolve version %q: %w", version, err)
	}
	return s.Orchestrate("php", exactVersion, compiler, fresh)
}

func (s *bundlerRepository) Orchestrate(name, exactVersion string, forceCompiler string, fresh bool) (domain.Forge, error) {
	if fresh {
		if err := s.freshClean(name, exactVersion); err != nil {
			return domain.Forge{}, fmt.Errorf("failed to clean existing installation: %w", err)
		}
	}

	if err := s.siloRepo.MarkInProgress(exactVersion); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to mark installation in progress: %w", err)
	}

	levels, err := s.assemblerSvc.GetDependencyLevels(name, exactVersion)
	if err != nil {
		s.siloRepo.MarkFailed(exactVersion)
		return domain.Forge{}, fmt.Errorf("failed to resolve dependency levels: %w", err)
	}

	sem := make(chan struct{}, s.jobs)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	completed := make(map[string]bool)
	var depLibraryPaths []string
	var depCppFlags []string
	var depLdFlags []string
	var builtDeps []domain.DependencyInfo

	for levelIdx, level := range levels {
		wg.Add(len(level))
		for _, dep := range level {
			go func(dep domain.Dependency) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				mu.Lock()
				if firstErr != nil {
					mu.Unlock()
					return
				}
				mu.Unlock()

				depVersion := dep.Version
				if idx := strings.Index(dep.Version, "|"); idx != -1 {
					depVersion = dep.Version[:idx]
				}

				contextMsg := ""
				depInfo, err := s.buildPackageWithInfo(dep.Name, depVersion, exactVersion, depLibraryPaths, depCppFlags, depLdFlags, contextMsg, buildTools[dep.Name], forceCompiler)
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("failed to build %s@%s: %w", dep.Name, depVersion, err)
					}
					mu.Unlock()
					return
				}

				mu.Lock()
				completed[dep.Name+"@"+depVersion] = true
				if !buildTools[dep.Name] {
					depPath := utils.DependencyPath(s.silo, exactVersion, dep.Name, depVersion)
					depLibraryPaths = append(depLibraryPaths, filepath.Join(depPath, "lib"))
					depCppFlags = append(depCppFlags, fmt.Sprintf("-I%s/include", depPath))
					depLdFlags = append(depLdFlags, fmt.Sprintf("-L%s/lib", depPath))
				}
				if depInfo != nil {
					builtDeps = append(builtDeps, *depInfo)
				}
				mu.Unlock()
			}(dep)
		}
		wg.Wait()

		if firstErr != nil {
			s.siloRepo.MarkFailed(exactVersion)
			s.siloRepo.Rollback(exactVersion)
			return domain.Forge{}, firstErr
		}
		_ = levelIdx
	}

	if err := s.buildPHP(name, exactVersion, depLibraryPaths, depCppFlags, depLdFlags, forceCompiler); err != nil {
		s.siloRepo.MarkFailed(exactVersion)
		s.siloRepo.Rollback(exactVersion)
		return domain.Forge{}, fmt.Errorf("failed to build PHP: %w", err)
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

func (s *bundlerRepository) freshClean(name, exactVersion string) error {
	pathsToClean := []string{
		utils.PHPVersionPath(s.silo, exactVersion),
		utils.GetSourcePath(s.silo, name, exactVersion),
	}

	for _, path := range pathsToClean {
		if exists, _ := afero.Exists(s.fs, path); exists {
			if err := s.fs.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
			}
		}
	}

	return nil
}
