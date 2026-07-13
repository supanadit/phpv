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
	GetState(name, version string) (domain.InstallState, error)
	MarkInProgress(name, version string) error
	MarkComplete(name, version string) error
	MarkFailed(name, version string) error

	// Default version management.
	GetDefault() (string, error)
	SetDefault(version string) error

	// Path helpers.
	PHPOutputPath(phpVersion string) string
	SourcePath(pkg, version string) string
	PackagePrefix(name, version string) string
	PECLArchivePath(name, version string) string
	BuildLogPath(pkg, version, logName string) string

	// Extension manifest.
	GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error)
	SaveExtensionManifest(phpVersion string, m *domain.ExtensionManifest) error

	// System mode.
	IsSystemMode() bool
	SetSystemMode(enabled bool) error
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

// DownloadURL downloads a file from the given URL with optional checksum
// verification. This is the low-level method used by the assembler when
// it already has the registry entry resolved.
func (s *Service) DownloadURL(url, checksumType, checksumValue string) (bool, error) {
	return s.siloRep.Download(url, checksumType, checksumValue)
}

// Extract extracts a downloaded archive into the destination directory.
func (s *Service) Extract(archivePath, destDir string) (bool, error) {
	return s.siloRep.Extract(archivePath, destDir)
}

// GetSilo returns the storage root.
func (s *Service) GetSilo() domain.Silo {
	return s.siloRep.GetSilo()
}

// GetState returns the install state for a package.
func (s *Service) GetState(name, version string) (domain.InstallState, error) {
	return s.siloRep.GetState(name, version)
}

// MarkInProgress marks a package installation as in-progress.
func (s *Service) MarkInProgress(name, version string) error {
	return s.siloRep.MarkInProgress(name, version)
}

// MarkComplete marks a package installation as complete.
func (s *Service) MarkComplete(name, version string) error {
	return s.siloRep.MarkComplete(name, version)
}

// MarkFailed marks a package installation as failed.
func (s *Service) MarkFailed(name, version string) error {
	return s.siloRep.MarkFailed(name, version)
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

// PackagePrefix returns the install prefix for any package.
func (s *Service) PackagePrefix(name, version string) string {
	return s.siloRep.PackagePrefix(name, version)
}

// PECLArchivePath returns the download cache path for a PECL archive.
func (s *Service) PECLArchivePath(name, version string) string {
	return s.siloRep.PECLArchivePath(name, version)
}

// BuildLogPath returns the path to a build log file.
func (s *Service) BuildLogPath(pkg, version, logName string) string {
	return s.siloRep.BuildLogPath(pkg, version, logName)
}

// GetExtensionManifest returns the extension manifest for a PHP version.
func (s *Service) GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error) {
	return s.siloRep.GetExtensionManifest(phpVersion)
}

// SaveExtensionManifest saves the extension manifest for a PHP version.
func (s *Service) SaveExtensionManifest(phpVersion string, m *domain.ExtensionManifest) error {
	return s.siloRep.SaveExtensionManifest(phpVersion, m)
}

// IsSystemMode returns true if the .phpv_system marker exists.
func (s *Service) IsSystemMode() bool {
	return s.siloRep.IsSystemMode()
}

// SetSystemMode creates or removes the .phpv_system marker.
func (s *Service) SetSystemMode(enabled bool) error {
	return s.siloRep.SetSystemMode(enabled)
}
