package assembler

import (
	"fmt"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

type AssemblerRepository interface {
	GetGraph(packageName string, version string) (domain.DependencyGraph, error)
	GetDependencies(packageName string, version string) ([]domain.Dependency, error)
}

type AssemblerService struct {
	packages map[string]domain.Package
}

func NewAssemblerService() *AssemblerService {
	return &AssemblerService{
		packages: make(map[string]domain.Package),
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

func (s *AssemblerService) GetGraph(packageName string, version string) (domain.DependencyGraph, error) {
	visited := make(map[string]bool)

	var resolve func(name string, ver string) (domain.DependencyGraph, error)

	resolve = func(name string, ver string) (domain.DependencyGraph, error) {
		if visited[name] {
			return domain.DependencyGraph{}, nil
		}

		visited[name] = true
		defer func() { visited[name] = false }()

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

	result := domain.DependencyGraph{}
	for k, v := range graph {
		result[k] = v
	}

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
