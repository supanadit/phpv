package forge

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

// ForgeRepository handles building and installing packages from source.
// Build and Install are separate so you can compile once and install later,
// or re-install without recompiling.
type ForgeRepository interface {
	// Build compiles the package from sourceDir. Returns the build directory
	// and environment variables needed for installation.
	Build(name string, version string, sourceDir string) (buildDir string, env map[string]string, err error)

	// Install installs a previously built package into prefix.
	Install(name string, version string, buildDir string, prefix string) error
}

// Service orchestrates the full forge pipeline for a single package:
// resolve registry → download → extract → build → install.
// It is agnostic — the caller provides sourceDir and prefix.
type Service struct {
	forgeRep    ForgeRepository
	registryRep registry.RegistryRepository
	siloRep     silo.SiloRepository
}

func NewService(fr ForgeRepository, rr registry.RegistryRepository, sr silo.SiloRepository) *Service {
	return &Service{
		forgeRep:    fr,
		registryRep: rr,
		siloRep:     sr,
	}
}

// ForgeResult holds the outcome of a full forge pipeline for a package.
type ForgeResult struct {
	Name       string
	Version    string
	Downloaded bool
	Extracted  bool
	Forged     bool
	Err        error
}

// ForgePackage runs the full pipeline for a single package:
// 1. Resolve registry entry (URL, checksum)
// 2. Download to cache (skip if exists)
// 3. Extract to sources (skip if exists)
// 4. Build from source
// 5. Install into prefix
//
// cacheDir and sourceDir are the directories where archives and extracted
// sources live. prefix is where the built package gets installed.
func (s *Service) ForgePackage(name string, version string, cacheDir string, sourceDir string, prefix string) (*ForgeResult, *domain.Forge) {
	result := &ForgeResult{Name: name, Version: version}

	// 1. Resolve registry entry.
	r, err := s.registryRep.Get(name, version, false, "")
	if err != nil {
		result.Err = fmt.Errorf("registry resolve %s@%s: %w", name, version, err)
		return result, nil
	}

	// 2. Download to cache.
	downloaded, err := s.siloRep.Download(r.URL, r.ChecksumType, r.ChecksumValue)
	if err != nil {
		result.Err = fmt.Errorf("download %s@%s: %w", name, version, err)
		return result, nil
	}
	result.Downloaded = downloaded

	// 3. Extract to sources.
	extracted, err := s.siloRep.Extract(sourceDir, sourceDir)
	if err != nil {
		result.Err = fmt.Errorf("extract %s@%s: %w", name, version, err)
		return result, nil
	}
	result.Extracted = extracted

	// 4. Build from source.
	buildDir, env, err := s.forgeRep.Build(name, version, sourceDir)
	if err != nil {
		result.Err = fmt.Errorf("build %s@%s: %w", name, version, err)
		return result, nil
	}

	// 5. Install into prefix.
	if err := s.forgeRep.Install(name, version, buildDir, prefix); err != nil {
		result.Err = fmt.Errorf("install %s@%s: %w", name, version, err)
		return result, nil
	}
	result.Forged = true

	return result, &domain.Forge{
		Prefix: prefix,
		Env:    env,
	}
}