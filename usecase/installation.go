package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/supanadit/phpv/domain"
)

// InstallationService represent the installation usecase
type InstallationService struct {
	versionRepo      PHPVersionRepository
	installationRepo InstallationRepository
	downloader       Downloader
	builder          Builder
	filesystem       FileSystem
	baseDir          string
}

// NewInstallationService creates a new installation service
func NewInstallationService(
	versionRepo PHPVersionRepository,
	installationRepo InstallationRepository,
	downloader Downloader,
	builder Builder,
	filesystem FileSystem,
	baseDir string,
) *InstallationService {
	return &InstallationService{
		versionRepo:      versionRepo,
		installationRepo: installationRepo,
		downloader:       downloader,
		builder:          builder,
		filesystem:       filesystem,
		baseDir:          baseDir,
	}
}

// InstallVersion installs a PHP version
func (s *InstallationService) InstallVersion(ctx context.Context, versionStr string) error {
	// Parse and validate version
	version, err := domain.ParseVersion(versionStr)
	if err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}

	// Check if version is already installed
	existing, err := s.installationRepo.GetInstallationByVersion(ctx, version)
	if err == nil && existing.IsInstalled() {
		return domain.ErrConflict
	}

	// Get version details from repository (or create if not exists)
	versionDetails, err := s.versionRepo.GetVersionByString(ctx, versionStr)
	if err != nil {
		// If version not found, use the parsed version
		versionDetails = version
		if err := s.versionRepo.SaveVersion(ctx, versionDetails); err != nil {
			return fmt.Errorf("failed to save version: %w", err)
		}
	}

	// Create installation directories
	installPath := filepath.Join(s.baseDir, "versions", versionStr)
	sourcePath := filepath.Join(s.baseDir, "sources", versionStr)

	if err := s.filesystem.CreateDirectory(installPath); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	if err := s.filesystem.CreateDirectory(sourcePath); err != nil {
		return fmt.Errorf("failed to create source directory: %w", err)
	}

	// Download source code
	if err := s.downloader.DownloadSource(ctx, versionDetails, sourcePath); err != nil {
		// Cleanup on failure
		s.filesystem.RemoveDirectory(sourcePath)
		s.filesystem.RemoveDirectory(installPath)
		return fmt.Errorf("failed to download source: %w", err)
	}

	// Build PHP
	config := map[string]string{
		"--prefix":                    installPath,
		"--enable-shared":             "no",
		"--enable-static":             "yes",
		"--disable-all":               "",   // Disable most extensions for minimal build
		"--enable-cli":                "",   // Enable CLI
		"--enable-zts":                "no", // Disable thread safety
		"--with-config-file-path":     filepath.Join(installPath, "etc"),
		"--with-config-file-scan-dir": filepath.Join(installPath, "etc", "conf.d"),
	}
	if err := s.builder.Build(ctx, sourcePath, installPath, config); err != nil {
		// Cleanup on failure
		s.filesystem.RemoveDirectory(sourcePath)
		s.filesystem.RemoveDirectory(installPath)
		return fmt.Errorf("failed to build PHP: %w", err)
	}

	// Create installation record
	installation := domain.Installation{
		Version:     versionDetails,
		Path:        installPath,
		IsActive:    false,
		InstalledAt: time.Now(),
	}

	if err := s.installationRepo.SaveInstallation(ctx, installation); err != nil {
		// Cleanup on failure
		s.filesystem.RemoveDirectory(sourcePath)
		s.filesystem.RemoveDirectory(installPath)
		return fmt.Errorf("failed to save installation: %w", err)
	}

	return nil
}

// SwitchVersion switches to a different PHP version
func (s *InstallationService) SwitchVersion(ctx context.Context, versionStr string) error {
	// Parse version
	version, err := domain.ParseVersion(versionStr)
	if err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}

	// Check if version is installed
	installation, err := s.installationRepo.GetInstallationByVersion(ctx, version)
	if err != nil {
		return domain.ErrNotFound
	}

	if !installation.IsInstalled() {
		return domain.ErrVersionNotInstalled
	}

	// Check if already active
	active, err := s.installationRepo.GetActiveInstallation(ctx)
	if err == nil && active.Version.Compare(version) == 0 {
		return domain.ErrVersionAlreadyActive
	}

	// Set as active
	installation.Activate()
	if err := s.installationRepo.SetActiveInstallation(ctx, installation); err != nil {
		return fmt.Errorf("failed to set active installation: %w", err)
	}

	return nil
}

// ListInstalledVersions returns all installed PHP versions
func (s *InstallationService) ListInstalledVersions(ctx context.Context) ([]domain.Installation, error) {
	return s.installationRepo.GetAllInstallations(ctx)
}

// GetActiveVersion returns the currently active PHP version
func (s *InstallationService) GetActiveVersion(ctx context.Context) (domain.Installation, error) {
	return s.installationRepo.GetActiveInstallation(ctx)
}

// UninstallVersion removes an installed PHP version
func (s *InstallationService) UninstallVersion(ctx context.Context, versionStr string) error {
	// Parse version
	version, err := domain.ParseVersion(versionStr)
	if err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}

	// Get installation
	installation, err := s.installationRepo.GetInstallationByVersion(ctx, version)
	if err != nil {
		return domain.ErrNotFound
	}

	// Check if it's the active version
	active, err := s.installationRepo.GetActiveInstallation(ctx)
	if err == nil && active.Version.Compare(version) == 0 {
		return fmt.Errorf("cannot uninstall active version, switch to another version first")
	}

	// Remove from filesystem
	if err := s.filesystem.RemoveDirectory(installation.Path); err != nil {
		return fmt.Errorf("failed to remove installation directory: %w", err)
	}

	// Remove source directory if exists
	sourcePath := filepath.Join(s.baseDir, "sources", versionStr)
	s.filesystem.RemoveDirectory(sourcePath) // Ignore error, source might not exist

	// Remove from repository
	if err := s.installationRepo.DeleteInstallation(ctx, version); err != nil {
		return fmt.Errorf("failed to remove installation record: %w", err)
	}

	return nil
}
