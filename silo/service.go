package silo

import (
	"io"

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

	FullClean(pkg, ver string) error
	CleanAll() error
}

type Service struct {
	repo SiloRepository
}

func NewService(repo SiloRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetSilo() (*domain.Silo, error) {
	return s.repo.GetSilo()
}

func (s *Service) EnsurePaths() error {
	return s.repo.EnsurePaths()
}

func (s *Service) ArchiveExists(pkg, ver string) bool {
	return s.repo.ArchiveExists(pkg, ver)
}

func (s *Service) GetArchivePath(pkg, ver string) string {
	return s.repo.GetArchivePath(pkg, ver)
}

func (s *Service) StoreArchive(pkg, ver string, data io.Reader) error {
	return s.repo.StoreArchive(pkg, ver, data)
}

func (s *Service) RetrieveArchive(pkg, ver string) (io.ReadCloser, error) {
	return s.repo.RetrieveArchive(pkg, ver)
}

func (s *Service) RemoveArchive(pkg, ver string) error {
	return s.repo.RemoveArchive(pkg, ver)
}

func (s *Service) ListArchives() []string {
	return s.repo.ListArchives()
}

func (s *Service) SourceExists(pkg, ver string) bool {
	return s.repo.SourceExists(pkg, ver)
}

func (s *Service) GetSourcePath(pkg, ver string) string {
	return s.repo.GetSourcePath(pkg, ver)
}

func (s *Service) StoreSource(pkg, ver string, data io.Reader) error {
	return s.repo.StoreSource(pkg, ver, data)
}

func (s *Service) RetrieveSource(pkg, ver string) (io.ReadCloser, error) {
	return s.repo.RetrieveSource(pkg, ver)
}

func (s *Service) RemoveSource(pkg, ver string) error {
	return s.repo.RemoveSource(pkg, ver)
}

func (s *Service) ListSources() []string {
	return s.repo.ListSources()
}

func (s *Service) VersionExists(pkg, ver string) bool {
	return s.repo.VersionExists(pkg, ver)
}

func (s *Service) GetVersionPath(pkg, ver string) string {
	return s.repo.GetVersionPath(pkg, ver)
}

func (s *Service) StoreVersion(pkg, ver string, data io.Reader) error {
	return s.repo.StoreVersion(pkg, ver, data)
}

func (s *Service) RetrieveVersion(pkg, ver string) (io.ReadCloser, error) {
	return s.repo.RetrieveVersion(pkg, ver)
}

func (s *Service) RemoveVersion(pkg, ver string) error {
	return s.repo.RemoveVersion(pkg, ver)
}

func (s *Service) ListVersions() []string {
	return s.repo.ListVersions()
}

func (s *Service) FullClean(pkg, ver string) error {
	return s.repo.FullClean(pkg, ver)
}

func (s *Service) CleanAll() error {
	return s.repo.CleanAll()
}
