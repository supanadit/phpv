package assembler

import (
	"fmt"
	"maps"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

type AssemblerRepository interface {
	GetGraph(packageName string, version string) (domain.DependencyGraph, error)
	GetDependencies(packageName string, version string) ([]domain.Dependency, error)
	GetOrderedDependencies(packageName string, version string) ([]domain.Dependency, error)
	GetDependencyLevels(packageName string, version string) ([][]domain.Dependency, error)
}

type AssemblerService struct {
	packages map[string]domain.Package
	repo     AssemblerRepository
}

func NewAssemblerService() *AssemblerService {
	return &AssemblerService{
		packages: make(map[string]domain.Package),
	}
}

func NewAssemblerServiceWithRepo(repo AssemblerRepository) *AssemblerService {
	return &AssemblerService{
		packages: make(map[string]domain.Package),
		repo:     repo,
	}
}

func (s *AssemblerService) RegisterPackage(pkg domain.Package) {
	s.packages[pkg.Package] = pkg
}

func (s *AssemblerService) GetPackage(name string) (domain.Package, error) {
	pkg, ok := s.packages[name]
	if !ok {
		return domain.Package{}, fmt.Errorf("package not found: %s", name)
	}
	return pkg, nil
}

func (s *AssemblerService) GetDependencies(packageName string, version string) ([]domain.Dependency, error) {
	pkg, err := s.GetPackage(packageName)
	if err != nil {
		return nil, err
	}

	for _, constraint := range pkg.Constraints {
		if utils.MatchVersionRange(constraint.VersionRange, version) {
			return constraint.Dependencies, nil
		}
	}

	return pkg.Default, nil
}

func (s *AssemblerService) GetOrderedDependencies(packageName string, version string) ([]domain.Dependency, error) {
	if s.repo != nil {
		return s.repo.GetOrderedDependencies(packageName, version)
	}

	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var result []domain.Dependency
	seen := make(map[string]bool)

	var resolve func(name string, ver string) error
	resolve = func(name string, ver string) error {
		if visited[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving %s", name)
		}

		visiting[name] = true

		deps, err := s.GetDependencies(name, ver)
		if err != nil {
			visiting[name] = false
			return fmt.Errorf("failed to get dependencies for %s@%s: %w", name, ver, err)
		}

		for _, dep := range deps {
			depVersion := dep.Version
			if idx := strings.Index(dep.Version, "|"); idx != -1 {
				depVersion = dep.Version[:idx]
			}

			if err := resolve(dep.Name, depVersion); err != nil {
				if dep.Optional {
					continue
				}
				visiting[name] = false
				return err
			}
		}

		visiting[name] = false
		visited[name] = true

		if name != packageName {
			key := name + "@" + ver
			if !seen[key] {
				seen[key] = true
				result = append(result, domain.Dependency{
					Name:     name,
					Version:  ver,
					Optional: false,
				})
			}
		}

		return nil
	}

	if err := resolve(packageName, version); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *AssemblerService) GetDependencyLevels(packageName string, version string) ([][]domain.Dependency, error) {
	if s.repo != nil {
		return s.repo.GetDependencyLevels(packageName, version)
	}

	allDeps, err := s.GetOrderedDependencies(packageName, version)
	if err != nil {
		return nil, err
	}

	depNamesInResult := make(map[string]bool)
	for _, dep := range allDeps {
		depNamesInResult[dep.Name] = true
	}

	type levelItem struct {
		name    string
		version string
	}

	var levels [][]levelItem
	placed := make(map[string]bool)
	depCache := make(map[string][]domain.Dependency)

	for len(placed) < len(allDeps) {
		var currentLevel []levelItem
		for _, dep := range allDeps {
			if placed[dep.Name] {
				continue
			}

			var deps []domain.Dependency
			if cached, ok := depCache[dep.Name]; ok {
				deps = cached
			} else {
				var err error
				deps, err = s.GetDependencies(dep.Name, dep.Version)
				if err != nil {
					deps = []domain.Dependency{}
				}
				depCache[dep.Name] = deps
			}

			canBuild := true
			for _, d := range deps {
				if depNamesInResult[d.Name] && !placed[d.Name] {
					canBuild = false
					break
				}
			}

			if canBuild {
				currentLevel = append(currentLevel, levelItem{
					name:    dep.Name,
					version: dep.Version,
				})
			}
		}

		if len(currentLevel) == 0 && len(placed) < len(allDeps) {
			return nil, fmt.Errorf("circular dependency detected")
		}

		for _, item := range currentLevel {
			placed[item.name] = true
		}
		levels = append(levels, currentLevel)
	}

	var result [][]domain.Dependency
	for _, level := range levels {
		var levelDeps []domain.Dependency
		for _, item := range level {
			levelDeps = append(levelDeps, domain.Dependency{
				Name:     item.name,
				Version:  item.version,
				Optional: false,
			})
		}
		result = append(result, levelDeps)
	}

	return result, nil
}

func (s *AssemblerService) GetGraph(packageName string, version string) (domain.DependencyGraph, error) {
	if s.repo != nil {
		return s.repo.GetGraph(packageName, version)
	}

	visiting := make(map[string]bool)
	visited := make(map[string]bool)

	var resolve func(name string, ver string) (domain.DependencyGraph, error)

	resolve = func(name string, ver string) (domain.DependencyGraph, error) {
		if visited[name] {
			return domain.DependencyGraph{}, nil
		}

		if visiting[name] {
			return domain.DependencyGraph{}, fmt.Errorf("circular dependency detected involving %s", name)
		}

		visiting[name] = true
		defer func() { visiting[name] = false }()

		deps, err := s.GetDependencies(name, ver)
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for %s@%s: %w", name, ver, err)
		}

		graph := domain.DependencyGraph{
			name: deps,
		}

		for _, dep := range deps {
			depVersion := dep.Version
			if idx := strings.Index(dep.Version, "|"); idx != -1 {
				depVersion = dep.Version[:idx]
			}

			depGraph, err := resolve(dep.Name, depVersion)
			if err != nil {
				if dep.Optional {
					continue
				}
				return nil, fmt.Errorf("failed to resolve dependency %s@%s: %w", dep.Name, depVersion, err)
			}

			visited[dep.Name] = true

			for k, v := range depGraph {
				if _, exists := graph[k]; !exists {
					graph[k] = v
				}
			}
		}

		return graph, nil
	}

	graph, err := resolve(packageName, version)
	if err != nil {
		return nil, err
	}

	result := make(domain.DependencyGraph)
	maps.Copy(result, graph)

	return result, nil
}

type assemblerRepository struct {
	*AssemblerService
}

func NewAssemblerRepository() AssemblerRepository {
	return &assemblerRepository{
		AssemblerService: NewAssemblerService(),
	}
}

func (r *assemblerRepository) GetGraph(packageName string, version string) (domain.DependencyGraph, error) {
	return r.AssemblerService.GetGraph(packageName, version)
}

func (r *assemblerRepository) GetDependencies(packageName string, version string) ([]domain.Dependency, error) {
	return r.AssemblerService.GetDependencies(packageName, version)
}

func (r *assemblerRepository) GetOrderedDependencies(packageName string, version string) ([]domain.Dependency, error) {
	return r.AssemblerService.GetOrderedDependencies(packageName, version)
}

func (r *assemblerRepository) GetDependencyLevels(packageName string, version string) ([][]domain.Dependency, error) {
	return r.AssemblerService.GetDependencyLevels(packageName, version)
}
