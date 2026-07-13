package patcher

import "github.com/supanadit/phpv/internal/repository"

// Patch describes a single source modification applied to extracted source code
// before building. Patches are needed when upstream packages cannot build on
// modern toolchains (e.g., GCC 15's stricter C23 pointer-type checking).
type Patch struct {
	// Name identifies the patch for logging.
	Name string
	// Package is the package name this patch applies to.
	Package string
	// VersionRange is an optional constraint (e.g., ">=6.9.0, <6.10.0").
	// Empty means "any version".
	VersionRange string
	// Apply mutates the extracted source tree in place.
	Apply func(sourceDir string) error
	// ExtraCFlags, if non-nil, are additional CFLAGS injected into the
	// package's build environment (e.g., to relax strict warnings).
	ExtraCFlags []string
	// ConfigureFlags are appended to the package's ./configure invocation.
	// Special placeholders are resolved by the assembler:
	//   {{prefix}} → the package's install prefix
	//   {{source}} → the extracted source directory
	//   {{dep:NAME}} → the install prefix of dependency NAME (e.g., openssl)
	ConfigureFlags []string
}

// PatcherRepository resolves the list of patches to apply for a given package.
type PatcherRepository interface {
	PatchesFor(name string, version string) []Patch
}

// PreparedPatches holds the result of preparing patches for a package.
type PreparedPatches struct {
	ExtraCFlags    []string
	ConfigureFlags []string
	Applied        []string
}

// Service wraps PatcherRepository and adds value by filtering patches
// by version constraint and providing a high-level Prepare method that
// applies patches and returns combined flags.
type Service struct {
	repo PatcherRepository
}

// NewService creates a patcher service backed by the given repository.
func NewService(r PatcherRepository) *Service {
	return &Service{repo: r}
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
		if p.VersionRange == "" || repository.MatchVersionRange(p.VersionRange, version) {
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
		if p.ExtraCFlags != nil {
			result.ExtraCFlags = p.ExtraCFlags
		}
		if len(p.ConfigureFlags) > 0 {
			result.ConfigureFlags = p.ConfigureFlags
		}
	}
	return result, nil
}
