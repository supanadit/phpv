package flagresolver

import (
	"github.com/supanadit/phpv/domain"
)

var ErrUnknownExtension = domain.ErrUnknownExtension
var ErrExtensionConflict = domain.ErrExtensionConflict

// CStdRule defines C/C++ compiler standard flags for a specific PHP version range.
//
// Use GetCompilerStdRule() to retrieve the appropriate rule for a PHP version.
// The rule provides:
//   - CStd: C standard flag (e.g., "-std=gnu11")
//   - CXXStd: C++ standard flag (e.g., "-std=gnu++17")
//
// This is primarily used to handle GCC 15+ compatibility with ICU headers,
// which require C++14+ features (std::enable_if_t, std::u16string_view, etc.).
//
// Example:
//
//	rule := s.GetCompilerStdRule("8.0.30")
//	env = append(env, "CFLAGS="+rule.CStd)
//	env = append(env, "CXXFLAGS="+rule.CXXStd)
type CStdRule struct {
	// MinPHP is the minimum PHP version (inclusive) for this rule.
	// Empty string means no minimum.
	MinPHP string

	// MaxPHP is the maximum PHP version (inclusive) for this rule.
	// Empty string means no maximum.
	MaxPHP string

	// CStd is the C standard flag (e.g., "-std=gnu11").
	CStd string

	// CXXStd is the C++ standard flag (e.g., "-std=gnu++17").
	CXXStd string
}

// Repository defines the interface for flag resolution operations.
// Implementations should provide package-specific and PHP-specific configure flags.
type Repository interface {
	// GetConfigureFlags returns configure flags for a package at a specific version.
	GetConfigureFlags(name string, version string) []string

	// GetPHPConfigureFlags returns configure flags for PHP with given extensions.
	GetPHPConfigureFlags(phpVersion string, extensions []string) []string

	// GetExtensionDef returns the definition for a named extension.
	GetExtensionDef(name string) (domain.ExtensionDef, bool)

	// IsExtensionValidForPHPVersion checks if an extension is valid for a PHP version.
	IsExtensionValidForPHPVersion(name string, phpVersion string) bool

	// GetConflictingExtensions returns list of extensions that conflict with the given one.
	GetConflictingExtensions(name string) []string

	// GetExtensionDependency returns the package name an extension depends on.
	GetExtensionDependency(name string) (string, bool)

	// GetExtensionDependencyWithVersion returns package name and version for an extension.
	GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool)

	// ValidateExtensions validates a list of extensions and returns unknown ones.
	ValidateExtensions(extensions []string, phpVersion string) ([]string, error)

	// CheckExtensionConflicts checks for extension conflicts in the given list.
	CheckExtensionConflicts(extensions []string) ([]string, [][]string)

	// GetCompilerStdRule returns C/C++ standard flags for a PHP version.
	GetCompilerStdRule(phpVersion string) CStdRule
}

// Service provides flag resolution operations for PHP and its extensions.
type Service struct {
	repo Repository
}

// NewService creates a new flag resolver service with the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetConfigureFlags returns configure flags for a package at a specific version.
func (s *Service) GetConfigureFlags(name string, version string) []string {
	return s.repo.GetConfigureFlags(name, version)
}

// GetPHPConfigureFlags returns configure flags for PHP with given extensions.
func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return s.repo.GetPHPConfigureFlags(phpVersion, extensions)
}

// GetExtensionDef returns the definition for a named extension.
func (s *Service) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return s.repo.GetExtensionDef(name)
}

// IsExtensionValidForPHPVersion checks if an extension is valid for a PHP version.
func (s *Service) IsExtensionValidForPHPVersion(name string, phpVersion string) bool {
	return s.repo.IsExtensionValidForPHPVersion(name, phpVersion)
}

// GetConflictingExtensions returns list of extensions that conflict with the given one.
func (s *Service) GetConflictingExtensions(name string) []string {
	return s.repo.GetConflictingExtensions(name)
}

// GetExtensionDependency returns the package name an extension depends on.
func (s *Service) GetExtensionDependency(name string) (string, bool) {
	return s.repo.GetExtensionDependency(name)
}

// GetExtensionDependencyWithVersion returns package name and version for an extension.
func (s *Service) GetExtensionDependencyWithVersion(ext, phpVersion string) (string, string, bool) {
	return s.repo.GetExtensionDependencyWithVersion(ext, phpVersion)
}

// ValidateExtensions validates a list of extensions and returns an error if any are unknown.
func (s *Service) ValidateExtensions(extensions []string, phpVersion string) error {
	unknown, err := s.repo.ValidateExtensions(extensions, phpVersion)
	if err != nil {
		return err
	}
	if len(unknown) > 0 {
		return ErrUnknownExtension
	}
	return nil
}

// CheckExtensionConflicts checks for extension conflicts and returns them.
// Returns (conflicts, conflictPairs, error).
func (s *Service) CheckExtensionConflicts(extensions []string) ([]string, [][]string, error) {
	conflicts, conflictPairs := s.repo.CheckExtensionConflicts(extensions)
	if len(conflicts) > 0 {
		return conflicts, conflictPairs, ErrExtensionConflict
	}
	return nil, nil, nil
}

// GetCompilerStdRule returns C/C++ compiler standard flags for a PHP version.
// Use this to get appropriate -std flags for building PHP with GCC 15+ compatibility.
func (s *Service) GetCompilerStdRule(phpVersion string) CStdRule {
	return s.repo.GetCompilerStdRule(phpVersion)
}
