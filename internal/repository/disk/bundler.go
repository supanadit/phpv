package disk

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/flagresolver"
	"github.com/supanadit/phpv/forge"
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
	patternRegistry *pattern.PatternRegistry
	flagResolverSvc *flagresolver.Service
	silo            *domain.Silo
	fs              afero.Fs
	jobs            int
	verbose         bool
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
		fs:              afero.NewOsFs(),
		jobs:            jobs,
		verbose:         cfg.Verbose,
	}
}

func (s *bundlerRepository) Install(version string) (domain.Forge, error) {
	exactVersion, err := s.resolvePHPVersion(version)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("failed to resolve version %q: %w", version, err)
	}
	return s.Orchestrate("php", exactVersion)
}

func (s *bundlerRepository) Orchestrate(name, exactVersion string) (domain.Forge, error) {
	if err := s.ensureBuildTools(); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to ensure build tools: %w", err)
	}

	graph, err := s.assemblerSvc.GetGraph(name, exactVersion)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("failed to resolve dependency graph: %w", err)
	}

	var depOrder []domain.VersionResolved
	processed := make(map[string]bool)

	for _, deps := range graph {
		for _, dep := range deps {
			depVer := extractVersion(dep.Version)
			key := dep.Name + "@" + depVer
			if !processed[key] {
				processed[key] = true
				depOrder = append(depOrder, domain.VersionResolved{
					Package: dep.Name,
					Version: depVer,
				})
			}
		}
	}

	ldLibraryPath := make([]string, 0)
	cppFlags := make([]string, 0)
	ldFlags := make([]string, 0)

	for _, dep := range depOrder {
		if err := s.buildPackage(dep.Package, dep.Version, exactVersion, ldLibraryPath, cppFlags, ldFlags); err != nil {
			return domain.Forge{}, fmt.Errorf("failed to build %s@%s: %w", dep.Package, dep.Version, err)
		}
		depPath := s.silo.DependencyPath(exactVersion, dep.Package, dep.Version)
		ldLibraryPath = append(ldLibraryPath, filepath.Join(depPath, "lib"))
		cppFlags = append(cppFlags, fmt.Sprintf("-I%s/include", depPath))
		ldFlags = append(ldFlags, fmt.Sprintf("-L%s/lib", depPath))
	}

	if err := s.buildPHP(name, exactVersion, ldLibraryPath, cppFlags, ldFlags); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to build PHP: %w", err)
	}

	outputPath := s.silo.PHPOutputPath(exactVersion)
	ldLibraryPath = append(ldLibraryPath, filepath.Join(outputPath, "lib"))

	return domain.Forge{
		Prefix: outputPath,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(ldLibraryPath, ":"),
		},
	}, nil
}

func (s *bundlerRepository) resolvePHPVersion(constraint string) (string, error) {
	sources, err := s.sourceSvc.GetVersions()
	if err != nil {
		return "", err
	}

	var phpSources []domain.Source
	for _, src := range sources {
		if src.Name == "php" {
			phpSources = append(phpSources, src)
		}
	}

	parts := strings.Split(constraint, ".")
	major := 0
	minor := 0
	patch := -1

	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}

	var candidates []domain.Source
	for _, src := range phpSources {
		v := pattern.ParseVersion(src.Version)
		if v.Major != major {
			continue
		}
		if minor > 0 && v.Minor != minor {
			continue
		}
		if patch >= 0 && v.Patch != patch {
			continue
		}
		candidates = append(candidates, src)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no PHP version found matching %q", constraint)
	}

	sort.Slice(candidates, func(i, j int) bool {
		vi := pattern.ParseVersion(candidates[i].Version)
		vj := pattern.ParseVersion(candidates[j].Version)
		if vi.Major != vj.Major {
			return vi.Major > vj.Major
		}
		if vi.Minor != vj.Minor {
			return vi.Minor > vj.Minor
		}
		return vi.Patch > vj.Patch
	})

	return candidates[0].Version, nil
}
