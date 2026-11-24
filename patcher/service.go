package patcher

import (
	"github.com/supanadit/phpv/domain"
	vsn "github.com/supanadit/phpv/pkg/version"
)

// Patch describes a source modification applied before building.
type Patch = domain.SourcePatch

// PatcherRepository resolves the list of patches to apply for a given package.
type PatcherRepository interface {
	PatchesFor(name string, version string) []Patch
}

// PreparedPatches holds the result of preparing patches for a package.
type PreparedPatches = domain.PreparedPatches

// FlagCatalog is the swappable source of compiler and configure flag rules.
// The built-in catalog lives in the memory driver; a REST or SQL driver can
// replace it without changing this module.
type FlagCatalog interface {
	CompilerFlags(compiler, phpVersion string) []string
	CompilerStdRule(phpVersion string) domain.CompilerRule
	ConfigureFlagRules(name string) []domain.ConfigureFlagRule
	DefaultExtensionCandidates() []string
}

// Service wraps PatcherRepository and adds value by filtering patches
// by version constraint and providing a high-level Prepare method that
// applies patches and returns combined flags.
type Service struct {
	repo  PatcherRepository
	flags FlagCatalog
}

// NewService creates a patcher service backed by the given repository and flag catalog.
func NewService(r PatcherRepository, flags FlagCatalog) *Service {
	return &Service{repo: r, flags: flags}
}

func (s *Service) CompilerFlags(compiler, phpVersion string) []string {
	return s.flags.CompilerFlags(compiler, phpVersion)
}
func (s *Service) CompilerStdRule(phpVersion string) domain.CompilerRule {
	return s.flags.CompilerStdRule(phpVersion)
}
func (s *Service) ConfigureFlagRules(name string) []domain.ConfigureFlagRule {
	return s.flags.ConfigureFlagRules(name)
}
func (s *Service) DefaultExtensionCandidates() []string {
	return s.flags.DefaultExtensionCandidates()
}

// PatchesFor returns patches that apply to the given (name, version),
// filtered by each patch's VersionRange constraint.
func (s *Service) PatchesFor(name string, version string) []Patch {
	all := s.repo.PatchesFor(name, version)
	if len(all) == 0 {
		return nil
	}
	filtered := make([]Patch, 0, len(all))
	for _, p := range all {
		if p.VersionRange == "" || vsn.MatchVersionRange(p.VersionRange, version) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// Prepare applies all matching patches to sourceDir and returns the
// combined ExtraCFlags and ConfigureFlags. This consolidates the
// three separate calls (PatchesFor + Apply + read flags) into one.
func (s *Service) Prepare(name string, version string, sourceDir string) (*PreparedPatches, error) {
	patches := s.PatchesFor(name, version)
	if len(patches) == 0 {
		return &PreparedPatches{}, nil
	}
	result := &PreparedPatches{}
	for _, p := range patches {
		if p.Apply != nil {
			if err := p.Apply(sourceDir); err != nil {
				return nil, err
			}
			result.Applied = append(result.Applied, p.Name)
		}
		if len(p.ExtraCFlags) > 0 {
			result.ExtraCFlags = append(result.ExtraCFlags, p.ExtraCFlags...)
		}
		if len(p.ConfigureFlags) > 0 {
			result.ConfigureFlags = append(result.ConfigureFlags, p.ConfigureFlags...)
		}
	}
	return result, nil
}
