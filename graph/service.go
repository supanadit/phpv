package graph

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/supanadit/phpv/domain"
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
type BuildPlan struct {
	Deps             []domain.Dependency
	ConfigureFlags   []string
	CFlags           []string
	CompilerFlags    []string
	CXXCompilerFlags []string
	Warnings         []string // non-fatal warnings (e.g. conflicting dep versions)
}

// Service wraps GraphRepository and adds value by consolidating
// multiple queries into a single GetBuildPlan call.
type Service struct {
	repo GraphRepository
}

// NewService creates a graph service backed by the given repository.
func NewService(repo GraphRepository) *Service {
	return &Service{repo: repo}
}

// GetBuildPlan returns everything needed to build (name, version) in one call.
// For PHP builds, the dependency list is derived from the requested extensions'
// RequiresPackage + Versions (resolved per PHP version), merged with any
// package-level deps from GetOrderedDependencies. This ensures PHP 8.2+ gets
// a deterministic dep plan even though the php package has no hardcoded
// Constraints for those versions.
func (s *Service) GetBuildPlan(name string, version string, extensions []string) (*BuildPlan, error) {
	deps, err := s.repo.GetOrderedDependencies(name, version)
	if err != nil {
		return nil, err
	}
	var warnings []string
	if name == "php" && len(extensions) > 0 {
		expanded, _ := s.repo.ExpandImplied(extensions)
		extDeps, w := s.resolveExtensionDeps(version, expanded)
		deps = mergeDeps(deps, extDeps)
		warnings = w
	}
	var configureFlags []string
	if name == "php" {
		configureFlags = s.repo.GetPHPConfigureFlags(version, extensions)
	} else {
		configureFlags = s.repo.GetConfigureFlags(name, version)
	}
	cflags := s.repo.GetCompilerFlags(detectCompiler(), version)
	compilerRule := s.repo.GetCompilerStdRule(version)
	var cCompilerFlags []string
	var cxxCompilerFlags []string
	if compilerRule.CStd != "" {
		cCompilerFlags = append(cCompilerFlags, compilerRule.CStd)
	}
	if compilerRule.CXXStd != "" {
		cxxCompilerFlags = append(cxxCompilerFlags, compilerRule.CXXStd)
	}
	return &BuildPlan{
		Deps:            deps,
		ConfigureFlags:  configureFlags,
		CFlags:          cflags,
		CompilerFlags:   cCompilerFlags,
		CXXCompilerFlags: cxxCompilerFlags,
		Warnings:        warnings,
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
		pkgName, pkgVersion, ok := s.repo.GetExtensionDependencyWithVersion(ext, phpVersion)
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
			existingVer := extractExactVersion(existing)
			newVer := extractExactVersion(pkgVersion)
			if compareVersions(newVer, existingVer) > 0 {
				warnings = append(warnings, fmt.Sprintf(
					"conflicting dep %s: extension %q wants %q, but another extension already pinned %q — using %q (higher)",
					pkgName, ext, pkgVersion, existing, pkgVersion,
				))
				direct[pkgName] = pkgVersion
			} else if compareVersions(newVer, existingVer) < 0 {
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
		transitive, err := s.repo.GetOrderedDependencies(name, extractExactVersion(ver))
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

// extractExactVersion returns the exact version part from a "exact|constraint" string.
func extractExactVersion(v string) string {
	if idx := strings.Index(v, "|"); idx != -1 {
		return v[:idx]
	}
	return v
}

// compareVersions compares two version strings numerically.
func compareVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var an, bn int
		if i < len(ap) {
			fmt.Sscanf(ap[i], "%d", &an)
		}
		if i < len(bp) {
			fmt.Sscanf(bp[i], "%d", &bn)
		}
		if an > bn {
			return 1
		}
		if an < bn {
			return -1
		}
	}
	return 0
}

// GetOrderedDependencies returns all transitive dependencies for (name, version).
func (s *Service) GetOrderedDependencies(name string, version string) ([]domain.Dependency, error) {
	return s.repo.GetOrderedDependencies(name, version)
}

// GetExtensionDef returns the definition for a named extension.
func (s *Service) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return s.repo.GetExtensionDef(name)
}

// IsExtensionValidForPHPVersion checks if an extension is valid for a PHP version.
func (s *Service) IsExtensionValidForPHPVersion(name string, phpVersion string) bool {
	return s.repo.IsExtensionValidForPHPVersion(name, phpVersion)
}

// GetConflictingExtensions returns extensions that conflict with the given one.
func (s *Service) GetConflictingExtensions(name string) []string {
	return s.repo.GetConflictingExtensions(name)
}

// GetExtensionDependency returns the package name an extension depends on.
func (s *Service) GetExtensionDependency(name string) (string, bool) {
	return s.repo.GetExtensionDependency(name)
}

// GetExtensionDependencyWithVersion returns package name and version for an extension.
func (s *Service) GetExtensionDependencyWithVersion(extName string, phpVersion string) (string, string, bool) {
	return s.repo.GetExtensionDependencyWithVersion(extName, phpVersion)
}

// ValidateExtensions validates a list of extensions and returns unknown ones.
func (s *Service) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	return s.repo.ValidateExtensions(extensions, phpVersion)
}

// CheckExtensionConflicts checks for extension conflicts in the given list.
func (s *Service) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	return s.repo.CheckExtensionConflicts(extensions)
}

// ListExtensions returns all known extensions.
func (s *Service) ListExtensions() []domain.ExtensionInfo {
	return s.repo.ListExtensions()
}

// ListExtensionsForPHP returns extensions valid for a specific PHP version.
func (s *Service) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	return s.repo.ListExtensionsForPHP(phpVersion)
}

// ExpandImplied returns the full extension set after expanding implied deps.
func (s *Service) ExpandImplied(extensions []string) (expanded []string, added []string) {
	return s.repo.ExpandImplied(extensions)
}

// DefaultExtensions returns the recommended default extension set for a
// typical PHP install, filtered to those compatible with the given PHP version.
func (s *Service) DefaultExtensions(phpVersion string) ([]string, []string) {
	return s.repo.DefaultExtensions(phpVersion)
}

// SharedOnlyExtensions returns extensions that must be built as shared.
func (s *Service) SharedOnlyExtensions(phpVersion string, requested []string) []string {
	return s.repo.SharedOnlyExtensions(phpVersion, requested)
}

// GetConfigureFlags returns configure flags for a package at a specific version.
func (s *Service) GetConfigureFlags(name string, version string) []string {
	return s.repo.GetConfigureFlags(name, version)
}

// GetPHPConfigureFlags returns configure flags for PHP with given extensions.
func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return s.repo.GetPHPConfigureFlags(phpVersion, extensions)
}

// GetExtensionConfigureFlags returns configure flags for a single extension
// at a specific PHP version. Flags are version-gated (e.g., --with-external-pcre
// only in PHP 7.4+).
func (s *Service) GetExtensionConfigureFlags(name string, phpVersion string) []string {
	return s.repo.GetExtensionConfigureFlags(name, phpVersion)
}

// GetCompilerStdRule returns C/C++ compiler standard flags for a PHP version.
func (s *Service) GetCompilerStdRule(phpVersion string) domain.CompilerRule {
	return s.repo.GetCompilerStdRule(phpVersion)
}

// GetCompilerFlags returns C compiler flags for a specific compiler and PHP version.
func (s *Service) GetCompilerFlags(compiler string, phpVersion string) []string {
	return s.repo.GetCompilerFlags(compiler, phpVersion)
}

// detectCompiler returns the compiler name to use for flag selection.
// macOS uses clang by default; Linux uses gcc.
func detectCompiler() string {
	if runtime.GOOS == "darwin" {
		return "clang"
	}
	return "gcc"
}
