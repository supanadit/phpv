package silo

import (
	"io"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
)

type SiloRepository interface {
	GetSilo() (*domain.Silo, error)
	EnsurePaths() error

	ArchiveExists(pkg, ver string) bool
	GetArchivePath(pkg, ver string) string
	StoreArchive(pkg, ver string, data io.Reader) error
	RetrieveArchive(pkg, ver string) (io.ReadCloser, error)
	RemoveArchive(pkg, ver string) error
	ListArchives() []string

	SourceExists(pkg, ver string) bool
	GetSourcePath(pkg, ver string) string
	StoreSource(pkg, ver string, data io.Reader) error
	RetrieveSource(pkg, ver string) (io.ReadCloser, error)
	RemoveSource(pkg, ver string) error
	ListSources() []string

	VersionExists(pkg, ver string) bool
	GetVersionPath(pkg, ver string) string
	StoreVersion(pkg, ver string, data io.Reader) error
	RetrieveVersion(pkg, ver string) (io.ReadCloser, error)
	RemoveVersion(pkg, ver string) error
	ListVersions() []string
	GetDefault() (string, error)
	SetDefault(version string) error

	FullClean(pkg, ver string) error
	CleanAll() error

	MarkInProgress(phpVersion string) error
	MarkComplete(phpVersion string) error
	MarkFailed(phpVersion string) error
	GetState(phpVersion string) (domain.InstallState, error)
	Rollback(phpVersion string) error

	SaveDependencyInfo(phpVersion string, deps []domain.DependencyInfo) error
	GetDependencyInfo(phpVersion string) ([]domain.DependencyInfo, error)
	RemoveDependencyInfo(phpVersion string) error

	IncrementBuildToolRef(name, version, phpVersion string) error
	DecrementBuildToolRef(name, version, phpVersion string) error
	GetBuildToolRefs() (map[string][]string, error)
	RemoveBuildToolRef(name, version string) error

	RemovePHPInstallation(phpVersion string) ([]string, error)
	GetInstalledBuildTools() ([]string, error)
	RemoveUnusedBuildTools(dryRun bool) ([]string, []string, error)
}

type Service struct {
	repo SiloRepository
}

func NewService(repo SiloRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSilo() (*domain.Silo, error)      { return s.repo.GetSilo() }
func (s *Service) EnsurePaths() error                   { return s.repo.EnsurePaths() }
func (s *Service) ArchiveExists(pkg, ver string) bool   { return s.repo.ArchiveExists(pkg, ver) }
func (s *Service) GetArchivePath(pkg, ver string) string { return s.repo.GetArchivePath(pkg, ver) }
func (s *Service) StoreArchive(pkg, ver string, data io.Reader) error { return s.repo.StoreArchive(pkg, ver, data) }
func (s *Service) RetrieveArchive(pkg, ver string) (io.ReadCloser, error) { return s.repo.RetrieveArchive(pkg, ver) }
func (s *Service) RemoveArchive(pkg, ver string) error { return s.repo.RemoveArchive(pkg, ver) }
func (s *Service) ListArchives() []string               { return s.repo.ListArchives() }
func (s *Service) SourceExists(pkg, ver string) bool    { return s.repo.SourceExists(pkg, ver) }
func (s *Service) GetSourcePath(pkg, ver string) string { return s.repo.GetSourcePath(pkg, ver) }
func (s *Service) StoreSource(pkg, ver string, data io.Reader) error { return s.repo.StoreSource(pkg, ver, data) }
func (s *Service) RetrieveSource(pkg, ver string) (io.ReadCloser, error) { return s.repo.RetrieveSource(pkg, ver) }
func (s *Service) RemoveSource(pkg, ver string) error { return s.repo.RemoveSource(pkg, ver) }
func (s *Service) ListSources() []string               { return s.repo.ListSources() }
func (s *Service) VersionExists(pkg, ver string) bool  { return s.repo.VersionExists(pkg, ver) }
func (s *Service) GetVersionPath(pkg, ver string) string { return s.repo.GetVersionPath(pkg, ver) }
func (s *Service) StoreVersion(pkg, ver string, data io.Reader) error { return s.repo.StoreVersion(pkg, ver, data) }
func (s *Service) RetrieveVersion(pkg, ver string) (io.ReadCloser, error) { return s.repo.RetrieveVersion(pkg, ver) }
func (s *Service) RemoveVersion(pkg, ver string) error { return s.repo.RemoveVersion(pkg, ver) }
func (s *Service) ListVersions() []string              { return s.repo.ListVersions() }
func (s *Service) GetDefault() (string, error)          { return s.repo.GetDefault() }
func (s *Service) SetDefault(version string) error     { return s.repo.SetDefault(version) }
func (s *Service) FullClean(pkg, ver string) error     { return s.repo.FullClean(pkg, ver) }
func (s *Service) CleanAll() error                     { return s.repo.CleanAll() }
func (s *Service) MarkInProgress(phpVersion string) error { return s.repo.MarkInProgress(phpVersion) }
func (s *Service) MarkComplete(phpVersion string) error { return s.repo.MarkComplete(phpVersion) }
func (s *Service) MarkFailed(phpVersion string) error  { return s.repo.MarkFailed(phpVersion) }
func (s *Service) GetState(phpVersion string) (domain.InstallState, error) { return s.repo.GetState(phpVersion) }
func (s *Service) Rollback(phpVersion string) error    { return s.repo.Rollback(phpVersion) }
func (s *Service) SaveDependencyInfo(phpVersion string, deps []domain.DependencyInfo) error { return s.repo.SaveDependencyInfo(phpVersion, deps) }
func (s *Service) GetDependencyInfo(phpVersion string) ([]domain.DependencyInfo, error) { return s.repo.GetDependencyInfo(phpVersion) }
func (s *Service) RemoveDependencyInfo(phpVersion string) error { return s.repo.RemoveDependencyInfo(phpVersion) }
func (s *Service) IncrementBuildToolRef(name, version, phpVersion string) error { return s.repo.IncrementBuildToolRef(name, version, phpVersion) }
func (s *Service) DecrementBuildToolRef(name, version, phpVersion string) error { return s.repo.DecrementBuildToolRef(name, version, phpVersion) }
func (s *Service) GetBuildToolRefs() (map[string][]string, error) { return s.repo.GetBuildToolRefs() }
func (s *Service) RemoveBuildToolRef(name, version string) error { return s.repo.RemoveBuildToolRef(name, version) }
func (s *Service) RemovePHPInstallation(phpVersion string) ([]string, error) { return s.repo.RemovePHPInstallation(phpVersion) }
func (s *Service) GetInstalledBuildTools() ([]string, error) { return s.repo.GetInstalledBuildTools() }
func (s *Service) RemoveUnusedBuildTools(dryRun bool) ([]string, []string, error) { return s.repo.RemoveUnusedBuildTools(dryRun) }

// Path helpers (pure logic — operate on domain.Silo)
func RootPath(silo *domain.Silo) string { return silo.Root }
func CachePath(silo *domain.Silo) string { return filepath.Join(silo.Root, "cache") }
func SourcePath(silo *domain.Silo) string { return filepath.Join(silo.Root, "sources") }
func VersionPath(silo *domain.Silo) string { return filepath.Join(silo.Root, "versions") }
func BinPath(silo *domain.Silo) string { return filepath.Join(silo.Root, "bin") }
func PharPath(silo *domain.Silo) string { return filepath.Join(silo.Root, "phar") }
func VersionPharPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(silo.Root, "versions", phpVersion, "phar")
}
func ArchiveKey(pkg, ver string) string { return filepath.Join("cache", pkg, ver, "archive") }
func SourceKey(pkg, ver string) string { return filepath.Join("sources", pkg, ver) }
func VersionKey(pkg, ver string) string { return filepath.Join("versions", pkg, ver) }
func SourceDirKey(pkg, ver string) string { return filepath.Join("sources", pkg, ver, "src") }
func ArchivePkgPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, ArchiveKey(pkg, ver))
}
func SourcePkgPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, SourceKey(pkg, ver))
}
func VersionPkgPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, VersionKey(pkg, ver))
}
func SourceDirPkgPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, SourceDirKey(pkg, ver))
}
func PHPVersionPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(silo.Root, "versions", phpVersion)
}
func PHPOutputPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(PHPVersionPath(silo, phpVersion), "output")
}
func DependencyPath(silo *domain.Silo, phpVersion, pkg, ver string) string {
	return filepath.Join(PHPVersionPath(silo, phpVersion), "dependency", pkg, ver)
}
func DependencyRootPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(PHPVersionPath(silo, phpVersion), "dependency")
}
func VersionWrapperPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(PHPVersionPath(silo, phpVersion), "wrapper")
}
func VersionWrapperBinPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(VersionWrapperPath(silo, phpVersion), "bin")
}
func VersionWrapperLibPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(VersionWrapperPath(silo, phpVersion), "lib")
}
func VersionWrapperIncludePath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(VersionWrapperPath(silo, phpVersion), "include")
}
func BuildToolsPath(silo *domain.Silo) string { return filepath.Join(silo.Root, "build-tools") }
func BuildToolPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(BuildToolsPath(silo), pkg, ver)
}
func BuildToolBinPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(BuildToolPath(silo, pkg, ver), "bin")
}
