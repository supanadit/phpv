package silo

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/registry"
)

// SiloRepository is the low-level storage interface. It owns downloads,
// archive extraction, state tracking, and default version management.
//
// Download returns true when the file was actually fetched from the network,
// and false when the file already existed (skipped).
//
// Extract returns true when the archive was actually extracted, and false
// when the source directory already existed (skipped).
type SiloRepository interface {
	Download(url string, checksumType string, checksumValue string) (downloaded bool, err error)
	Extract(archivePath string, destDir string) (extracted bool, err error)

	// GetSilo returns the storage root.
	GetSilo() domain.Silo

	// State management for any package.
	GetState(phpVersion string) (domain.InstallState, error)
	MarkInProgress(phpVersion string) error
	MarkComplete(phpVersion string) error
	MarkFailed(phpVersion string) error

	// Default version management.
	GetDefault() (string, error)
	SetDefault(version string) error

	// Path helpers.
	PHPOutputPath(phpVersion string) string
	SourcePath(pkg, version string) string
	DependencyPath(phpVersion, name, depVersion string) string
	PackagePrefix(name, version string) string
}

type Service struct {
	siloRep     SiloRepository
	registryRep *registry.Service
}

func NewService(sr SiloRepository, rr *registry.Service) *Service {
	return &Service{
		siloRep:     sr,
		registryRep: rr,
	}
}

// Download resolves the registry entry for the given name and version and
// then delegates the actual download to the SiloRepository. The registry
// entry provides the download URL and, when available, the checksum used
// to verify the integrity of the downloaded file.
func (s *Service) Download(name string, version string) (bool, error) {
	r, err := s.registryRep.Get(name, version)
	if err != nil {
		return false, err
	}
	return s.siloRep.Download(r.URL, r.ChecksumType, r.ChecksumValue)
}

// ExtractArchive extracts a downloaded archive from the cache into the
// sources directory. The archivePath is the full path to the cached file.
// destDir is the target directory under sources/.
func (s *Service) ExtractArchive(archivePath string, destDir string) (bool, error) {
	return s.siloRep.Extract(archivePath, destDir)
}

// GetSilo returns the storage root.
func (s *Service) GetSilo() domain.Silo {
	return s.siloRep.GetSilo()
}

// GetState returns the install state for a PHP version.
func (s *Service) GetState(phpVersion string) (domain.InstallState, error) {
	return s.siloRep.GetState(phpVersion)
}

// MarkInProgress marks a PHP installation as in-progress.
func (s *Service) MarkInProgress(phpVersion string) error {
	return s.siloRep.MarkInProgress(phpVersion)
}

// MarkComplete marks a PHP installation as complete.
func (s *Service) MarkComplete(phpVersion string) error {
	return s.siloRep.MarkComplete(phpVersion)
}

// MarkFailed marks a PHP installation as failed.
func (s *Service) MarkFailed(phpVersion string) error {
	return s.siloRep.MarkFailed(phpVersion)
}

// GetDefault returns the default PHP version.
func (s *Service) GetDefault() (string, error) {
	return s.siloRep.GetDefault()
}

// SetDefault sets the default PHP version.
func (s *Service) SetDefault(version string) error {
	return s.siloRep.SetDefault(version)
}

// PHPOutputPath returns the install prefix for a PHP version.
func (s *Service) PHPOutputPath(phpVersion string) string {
	return s.siloRep.PHPOutputPath(phpVersion)
}

// SourcePath returns the extracted source directory for a package.
func (s *Service) SourcePath(pkg, version string) string {
	return s.siloRep.SourcePath(pkg, version)
}

// DependencyPath returns the install prefix for a dependency of a PHP version.
func (s *Service) DependencyPath(phpVersion, name, depVersion string) string {
	return s.siloRep.DependencyPath(phpVersion, name, depVersion)
}

// PackagePrefix returns the install prefix for any package.
func (s *Service) PackagePrefix(name, version string) string {
	return s.siloRep.PackagePrefix(name, version)
}
