package graph

import (
	"fmt"
	"sort"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/compiler"
)

// GraphRepository is the data-access contract for PHP build knowledge.
// Implementations can be in-memory (hardcoded), REST API, PostgreSQL, etc.
type GraphRepository interface {
	// Dependency graph
	GetOrderedDependencies(name string, version string) ([]domain.Dependency, error)

	// Extension knowledge
	GetExtensionDef(name string) (domain.ExtensionDef, bool)
	IsExtensionValidForPHPVersion(name string, phpVersion string) bool
	GetConflictingExtensions(name string) []string
	GetExtensionDependency(name string) (string, bool)
	GetExtensionDependencyWithVersion(extName string, phpVersion string) (string, string, bool)
	ValidateExtensions(extensions []string, phpVersion string) ([]string, error)
	CheckExtensionConflicts(extensions []string) ([]string, [][]string)
	ListExtensions() []domain.ExtensionInfo
	ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo
	ExpandImplied(extensions []string) (expanded []string, added []string)

	// Default extension set
	DefaultExtensions(phpVersion string) (included []string, skipped []string)

	// Shared-only extensions (built as shared via phpize, not in main binary)
	SharedOnlyExtensions(phpVersion string, requested []string) []string

	// Flag knowledge
	GetConfigureFlags(name string, version string) []string
	GetPHPConfigureFlags(phpVersion string, extensions []string) []string
	GetExtensionConfigureFlags(name string, phpVersion string) []string

	// Compiler knowledge
	GetCompilerStdRule(phpVersion string) domain.CompilerRule
	GetCompilerFlags(compiler string, phpVersion string) []string
}

// BuildPlan consolidates everything needed to build a package.
type BuildPlan = domain.BuildPlan

// ExtensionSource is the extension-catalog capability graph consumes.
type ExtensionSource interface {
	ExpandImplied(extensions []string) (expanded []string, added []string)
	GetPHPConfigureFlags(phpVersion string, extensions []string) []string
	GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool)
}

// FlagSource is the compiler-flag capability graph consumes from patcher.
type FlagSource interface {
	CompilerFlags(compiler, phpVersion string) []string
	CompilerStdRule(phpVersion string) domain.CompilerRule
}

// Service wraps GraphRepository and adds value by consolidating
// multiple queries into a single GetBuildPlan call.
type Service struct {
	repo GraphRepository
	FlagSource
	ExtensionSource
}

// NewService creates a graph service backed by the given repository.
func NewService(repo GraphRepository, flags FlagSource, ext ExtensionSource) *Service {
	return &Service{repo: repo, FlagSource: flags, ExtensionSource: ext}
}

// GetBuildPlan returns everything needed to build (name, version) in one call.
// compilerName selects the compiler flag rules; an empty name uses the platform
// default, an unknown name is an error.
// For PHP builds, the dependency list is derived from the requested extensions'
// RequiresPackage + Versions (resolved per PHP version), merged with any
// package-level deps from GetOrderedDependencies. This ensures PHP 8.2+ gets
// a deterministic dep plan even though the php package has no hardcoded
// Constraints for those versions.
func (s *Service) GetBuildPlan(name string, version string, extensions []string, compilerName string) (*BuildPlan, error) {
	tc, err := compiler.Resolve(compilerName)
	if err != nil {
		return nil, err
	}
	deps, err := s.repo.GetOrderedDependencies(name, version)
	if err != nil {
		return nil, err
	}
	var warnings []string
	if name == "php" && len(extensions) > 0 {
		expanded, _ := s.ExtensionSource.ExpandImplied(extensions)
		extDeps, w := s.resolveExtensionDeps(version, expanded)
		deps = mergeDeps(deps, extDeps)
		warnings = w
	}
	var configureFlags []string
	if name == "php" {
		configureFlags = s.ExtensionSource.GetPHPConfigureFlags(version, extensions)
	} else {
		configureFlags = s.repo.GetConfigureFlags(name, version)
	}
	cflags := s.FlagSource.CompilerFlags(tc.Name, version)
	compilerRule := s.FlagSource.CompilerStdRule(version)
	var cCompilerFlags []string
	var cxxCompilerFlags []string
	// Era flags first, then the language standard last so the C standard is the
	// final -std the compiler sees.
	cCompilerFlags = append(cCompilerFlags, compilerRule.CFlags...)
	if compilerRule.CStd != "" {
		cCompilerFlags = append(cCompilerFlags, compilerRule.CStd)
	}
	if compilerRule.CXXStd != "" {
		cxxCompilerFlags = append(cxxCompilerFlags, compilerRule.CXXStd)
	}
	return &BuildPlan{
		Deps:             deps,
		ConfigureFlags:   configureFlags,
		CFlags:           cflags,
		CompilerFlags:    cCompilerFlags,
		CXXCompilerFlags: cxxCompilerFlags,
		Warnings:         warnings,
	}, nil
}

// resolveExtensionDeps walks the extension list and returns the full transitive
// set of source packages (with versions) that must be built. Deduplicates by
// package name; on conflicting versions, picks the higher and records a warning.
// Direct extension deps take priority over transitive deps from other packages.
//
// The returned list is deterministic and topologically sorted: every dependency
// appears before any package that depends on it. This is required so the
// assembler builds dependencies before dependents from a fresh state.
func (s *Service) resolveExtensionDeps(phpVersion string, extensions []string) ([]domain.Dependency, []string) {
	direct := make(map[string]string) // pkgName -> versionWithConstraint
	var warnings []string

	for _, ext := range extensions {
		pkgName, pkgVersion, ok := s.ExtensionSource.GetExtensionDependencyWithVersion(ext, phpVersion)
		if !ok || pkgName == "" {
			continue
		}
		if pkgVersion == "" {
			// Extension has RequiresPackage but no matching Versions entry
			// (e.g. bz2, libpq). Skip — the assembler's hybrid mode will
			// use the system package, or findDepPrefix will find any installed version.
			continue
		}
		if existing, dup := direct[pkgName]; dup && existing != pkgVersion {
			existingVer, _ := vsn.SplitVersionSpec(existing)
			newVer, _ := vsn.SplitVersionSpec(pkgVersion)
			if vsn.CompareVersions(newVer, existingVer) > 0 {
				warnings = append(warnings, fmt.Sprintf(
					"conflicting dep %s: extension %q wants %q, but another extension already pinned %q — using %q (higher)",
					pkgName, ext, pkgVersion, existing, pkgVersion,
				))
				direct[pkgName] = pkgVersion
			} else if vsn.CompareVersions(newVer, existingVer) < 0 {
				warnings = append(warnings, fmt.Sprintf(
					"conflicting dep %s: extension %q wants %q, but another extension already pinned %q — keeping %q (higher)",
					pkgName, ext, pkgVersion, existing, existing,
				))
			}
		} else if !dup {
			direct[pkgName] = pkgVersion
		}
	}

	// Sort direct dep names for deterministic processing. Load the full
	// dependency graph (direct + transitive) and run a post-order DFS so
	// every package is emitted after all of its dependencies.
	directNames := make([]string, 0, len(direct))
	for name := range direct {
		directNames = append(directNames, name)
	}
	sort.Strings(directNames)

	adj := make(map[string][]domain.Dependency) // name -> direct dependencies
	versions := make(map[string]string)         // name -> version to use (direct wins)

	var loadDeps func(name, ver string)
	loadDeps = func(name, ver string) {
		if _, ok := adj[name]; ok {
			return
		}
		exactVer, _ := vsn.SplitVersionSpec(ver)
		transitive, err := s.repo.GetOrderedDependencies(name, exactVer)
		if err != nil {
			transitive = nil
		}
		adj[name] = transitive
		if _, ok := versions[name]; !ok {
			versions[name] = ver
		}
		for _, td := range transitive {
			if _, ok := versions[td.Name]; !ok {
				versions[td.Name] = td.Version
			}
			loadDeps(td.Name, td.Version)
		}
	}

	// Seed direct versions first so they win over any transitive version.
	for name, ver := range direct {
		versions[name] = ver
	}
	for _, name := range directNames {
		loadDeps(name, direct[name])
	}

	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var result []domain.Dependency

	var resolve func(name string) error
	resolve = func(name string) error {
		if visited[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving %s", name)
		}
		visiting[name] = true
		for _, dep := range adj[name] {
			if err := resolve(dep.Name); err != nil {
				visiting[name] = false
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		result = append(result, domain.Dependency{Name: name, Version: versions[name]})
		return nil
	}

	for _, name := range directNames {
		if err := resolve(name); err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
	}

	return result, warnings
}

// mergeDeps merges extension-driven deps into the base dep list. Extension deps
// override base deps with the same name (extension defs are the source of truth).
func mergeDeps(base, extDeps []domain.Dependency) []domain.Dependency {
	extIndex := make(map[string]int)
	for i, d := range extDeps {
		extIndex[d.Name] = i
	}
	var merged []domain.Dependency
	for _, d := range base {
		if _, overridden := extIndex[d.Name]; !overridden {
			merged = append(merged, d)
		}
	}
	merged = append(merged, extDeps...)
	return merged
}

// GetOrderedDependencies returns all transitive dependencies for (name, version).
func (s *Service) GetOrderedDependencies(name string, version string) ([]domain.Dependency, error) {
	return s.repo.GetOrderedDependencies(name, version)
}

// GetConfigureFlags returns configure flags for a package at a specific version.
func (s *Service) GetConfigureFlags(name string, version string) []string {
	return s.repo.GetConfigureFlags(name, version)
}

// GetCompilerStdRule returns C/C++ compiler standard flags for a PHP version.
func (s *Service) GetCompilerStdRule(phpVersion string) domain.CompilerRule {
	return s.FlagSource.CompilerStdRule(phpVersion)
}

// GetCompilerFlags returns C compiler flags for a specific compiler and PHP version.
func (s *Service) GetCompilerFlags(compiler string, phpVersion string) []string {
	return s.FlagSource.CompilerFlags(compiler, phpVersion)
}

// CXXFlagsFromCFlagsWithStd converts a set of C compiler flags into their C++
// equivalents, substituting the C standard flag with the configured C++ standard
// and dropping C-only warnings. Used when PHP's build needs both CFLAGS and
// CXXFLAGS derived from a single source of truth.
// CXXFlagsFromCFlagsWithStd exposes the flag translation as a service method.
func (s *Service) CXXFlagsFromCFlagsWithStd(cflags []string, isPHPBuild bool, stdRule domain.CompilerRule) []string {
	return compiler.CXXFlagsFromCFlagsWithStd(cflags, isPHPBuild, compiler.StandardRule{CStd: stdRule.CStd, CXXStd: stdRule.CXXStd})
}
