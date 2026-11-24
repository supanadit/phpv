package memory

import (
	"fmt"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/supanadit/phpv/domain"
)

// GraphRepository is an in-memory implementation of graph.GraphRepository.
// It holds hardcoded build knowledge: dependency graph, extension definitions,
// flag rules, and compiler rules.
type GraphRepository struct {
	packages      map[string]domain.Package
	extensions    map[string]domain.ExtensionDef
	flagRules     []domain.FlagRule
	compilerRules []domain.CompilerRule
	conflicts     map[string][]string
	implied       map[string][]string
}

// NewGraphRepository creates a new in-memory graph repository with
// the built-in knowledge pre-registered.
func NewGraphRepository() *GraphRepository {
	r := &GraphRepository{
		packages:   make(map[string]domain.Package),
		extensions: make(map[string]domain.ExtensionDef),
		conflicts:  make(map[string][]string),
		implied:    make(map[string][]string),
	}
	r.registerPackages()
	r.registerExtensions()
	return r
}

// GetOrderedDependencies returns all transitive dependencies for
// (name, version) in dependency order — dependencies before dependents.
// The root package itself is excluded from the result.
// Circular dependencies are detected and return an error.
func (r *GraphRepository) GetOrderedDependencies(name string, version string) ([]domain.Dependency, error) {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	seen := make(map[string]bool)
	optional := make(map[string]bool)
	var result []domain.Dependency

	var resolve func(pkgName, pkgVersion string) error
	resolve = func(pkgName, pkgVersion string) error {
		if visited[pkgName] {
			return nil
		}
		if visiting[pkgName] {
			return fmt.Errorf("circular dependency detected involving %s", pkgName)
		}

		visiting[pkgName] = true

		deps, err := r.getDependencies(pkgName, pkgVersion)
		if err != nil {
			visiting[pkgName] = false
			return fmt.Errorf("failed to get dependencies for %s@%s: %w", pkgName, pkgVersion, err)
		}

		for _, dep := range deps {
			depVersion, _ := vsn.SplitVersionSpec(dep.Version)
			if dep.Optional {
				optional[dep.Name] = true
			}
			if err := resolve(dep.Name, depVersion); err != nil {
				if dep.Optional {
					continue
				}
				visiting[pkgName] = false
				return err
			}
		}

		visiting[pkgName] = false
		visited[pkgName] = true

		if pkgName != name {
			key := pkgName + "@" + pkgVersion
			if !seen[key] {
				seen[key] = true
				result = append(result, domain.Dependency{
					Name:     pkgName,
					Version:  pkgVersion,
					Optional: optional[pkgName],
				})
			}
		}

		return nil
	}

	if err := resolve(name, version); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *GraphRepository) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	def, ok := r.extensions[name]
	return def, ok
}

func (r *GraphRepository) IsExtensionValidForPHPVersion(name, phpVersion string) bool {
	def, ok := r.extensions[name]
	if !ok {
		return false
	}
	if def.MinPHPVersion != "" && vsn.CompareVersions(phpVersion, def.MinPHPVersion) < 0 {
		return false
	}
	if def.MaxPHPVersion != "" && vsn.CompareVersions(phpVersion, def.MaxPHPVersion) > 0 {
		return false
	}
	return true
}

func (r *GraphRepository) GetConflictingExtensions(name string) []string {
	return r.conflicts[name]
}

func (r *GraphRepository) GetExtensionDependency(name string) (string, bool) {
	def, ok := r.extensions[name]
	if !ok {
		return "", false
	}
	if def.RequiresPackage == "" {
		return "", false
	}
	return def.RequiresPackage, true
}

func (r *GraphRepository) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	def, ok := r.extensions[extName]
	if !ok || def.RequiresPackage == "" {
		return "", "", false
	}
	for _, v := range def.Versions {
		if vsn.MatchVersionRange(v.VersionRange, phpVersion) {
			return def.RequiresPackage, v.Version, true
		}
	}
	return def.RequiresPackage, "", true
}

func (r *GraphRepository) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	var unknown []string
	for _, ext := range extensions {
		if _, ok := r.extensions[ext]; !ok {
			unknown = append(unknown, ext)
		}
	}
	return unknown, nil
}

func (r *GraphRepository) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	var conflicts []string
	var groups [][]string
	for _, ext := range extensions {
		if c, ok := r.conflicts[ext]; ok {
			for _, other := range extensions {
				for _, conflict := range c {
					if other == conflict {
						conflicts = append(conflicts, ext)
						groups = append(groups, []string{ext, other})
					}
				}
			}
		}
	}
	return conflicts, groups
}

func (r *GraphRepository) ListExtensions() []domain.ExtensionInfo {
	var list []domain.ExtensionInfo
	for name, def := range r.extensions {
		list = append(list, domain.ExtensionInfo{
			Name:        name,
			Description: def.Description,
		})
	}
	return list
}

func (r *GraphRepository) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	var list []domain.ExtensionInfo
	for name, def := range r.extensions {
		if r.IsExtensionValidForPHPVersion(name, phpVersion) {
			list = append(list, domain.ExtensionInfo{
				Name:        name,
				Description: def.Description,
			})
		}
	}
	return list
}

func (r *GraphRepository) ExpandImplied(extensions []string) ([]string, []string) {
	seen := make(map[string]bool)
	var expanded []string
	var added []string
	var expand func(name string)
	expand = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		expanded = append(expanded, name)
		if implied, ok := r.implied[name]; ok {
			for _, imp := range implied {
				if !seen[imp] {
					added = append(added, imp)
					expand(imp)
				}
			}
		}
	}
	for _, ext := range extensions {
		expand(ext)
	}
	return expanded, added
}

func (r *GraphRepository) GetConfigureFlags(name, version string) []string {
	rules, ok := ConfigureFlagRules[name]
	if !ok {
		return nil
	}
	for _, rule := range rules {
		minOK := rule.MinVer == "" || vsn.CompareVersions(version, rule.MinVer) >= 0
		maxOK := rule.MaxVer == "" || vsn.CompareVersions(version, rule.MaxVer) <= 0
		if minOK && maxOK {
			return rule.Flags
		}
	}
	return nil
}
