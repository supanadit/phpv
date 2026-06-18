package advisor

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

type AdvisorRepository interface {
	Check(name string, version string, phpVersion string) (domain.AdvisorCheck, error)
	IsCompilerAvailable(compilerType domain.CompilerType) bool
	GetCompilerReadiness(phpVersion string) (domain.CompilerInfo, error)
}

type Service struct {
	repo AdvisorRepository
}

func NewAdvisorService(repo AdvisorRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Check(name string, version string, phpVersion string) (domain.AdvisorCheck, error) {
	return s.repo.Check(name, version, phpVersion)
}

// GetRequiredCompilerForPHP determines which compiler is preferred for the given PHP version.
// This is the preferred compiler based on PHP version requirements (not availability).
func (s *Service) GetRequiredCompilerForPHP(phpVersion string, forceCompiler domain.CompilerType) domain.CompilerType {
	if phpVersion == "" {
		return domain.CompilerTypeGCC
	}

	v := utils.ParseVersion(phpVersion)

	// For forced compiler selection
	if forceCompiler == domain.CompilerTypeZig {
		return domain.CompilerTypeZig
	} else if forceCompiler == domain.CompilerTypeGCC {
		return domain.CompilerTypeGCC
	}

	// PHP versions 5.x through 7.x prefer gcc
	if v.Major >= 5 && v.Major < 8 {
		return domain.CompilerTypeGCC
	}

	// PHP versions < 5 or >= 8 prefer zig
	return domain.CompilerTypeZig
}

// GetEffectiveCompilerForPHP returns the compiler that will actually be used for building.
// This considers both version requirements and actual availability.
func (s *Service) GetEffectiveCompilerForPHP(phpVersion string) domain.CompilerType {
	if phpVersion == "" {
		return domain.CompilerTypeGCC
	}

	v := utils.ParseVersion(phpVersion)

	// PHP 5+: always use gcc if available, else zig
	if v.Major >= 5 {
		if s.repo.IsCompilerAvailable(domain.CompilerTypeGCC) {
			return domain.CompilerTypeGCC
		}
		return domain.CompilerTypeZig
	}

	// PHP < 5: only zig (legacy)
	if s.repo.IsCompilerAvailable(domain.CompilerTypeZig) {
		return domain.CompilerTypeZig
	}
	return "" // No viable compiler
}

// UsesZigForPHP returns whether zig will be used for the given PHP version.
func (s *Service) UsesZigForPHP(phpVersion string) bool {
	return s.GetEffectiveCompilerForPHP(phpVersion) == domain.CompilerTypeZig
}

// GetZigTarget returns the zig target for the current platform.
func (s *Service) GetZigTarget() string {
	return utils.GetZigTarget()
}

// GetZigTargetForGlibc returns the zig target with a specific glibc version.
func (s *Service) GetZigTargetForGlibc(glibcVersion string) string {
	return utils.GetZigTargetForGlibc(glibcVersion)
}

// IsCompilerAvailable checks if a compiler is available via the repository.
func (s *Service) IsCompilerAvailable(compilerType domain.CompilerType) bool {
	return s.repo.IsCompilerAvailable(compilerType)
}

// GetCompilerReadiness returns compiler readiness info for the given PHP version.
func (s *Service) GetCompilerReadiness(phpVersion string) (domain.CompilerInfo, error) {
	return s.repo.GetCompilerReadiness(phpVersion)
}
