package graph

import (
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
	Deps           []domain.Dependency
	ConfigureFlags []string
	CFlags         []string
	CompilerFlags  []string
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
func (s *Service) GetBuildPlan(name string, version string, extensions []string) (*BuildPlan, error) {
	deps, err := s.repo.GetOrderedDependencies(name, version)
	if err != nil {
		return nil, err
	}
	configureFlags := s.repo.GetConfigureFlags(name, version)
	cflags := s.repo.GetCompilerFlags("gcc", version)
	compilerRule := s.repo.GetCompilerStdRule(version)
	var compilerFlags []string
	if compilerRule.CStd != "" {
		compilerFlags = append(compilerFlags, compilerRule.CStd)
	}
	if compilerRule.CXXStd != "" {
		compilerFlags = append(compilerFlags, compilerRule.CXXStd)
	}
	return &BuildPlan{
		Deps:           deps,
		ConfigureFlags: configureFlags,
		CFlags:         cflags,
		CompilerFlags:  compilerFlags,
	}, nil
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
