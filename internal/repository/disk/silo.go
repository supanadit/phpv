package disk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
)

var (
	ErrNotFound     = errors.New("item not found")
	ErrExists       = errors.New("item already exists")
	ErrInvalidInput = errors.New("invalid input")
)

type SiloRepository struct {
	fs   afero.Fs
	silo *domain.Silo
}

func NewSiloRepository() (*SiloRepository, error) {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		root = filepath.Join(homeDir, ".phpv")
	}

	return &SiloRepository{
		fs:   afero.NewOsFs(),
		silo: &domain.Silo{Root: root},
	}, nil
}

func (r *SiloRepository) GetSilo() (*domain.Silo, error) {
	return r.silo, nil
}

func (r *SiloRepository) EnsurePaths() error {
	paths := []string{
		r.silo.CachePath(),
		r.silo.SourcePath(),
		r.silo.VersionPath(),
		r.silo.BinPath(),
	}

	for _, path := range paths {
		if err := r.fs.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("failed to create path %s: %w", path, err)
		}
	}

	return nil
}

func (r *SiloRepository) validateInput(pkg, ver string) error {
	if pkg == "" {
		return fmt.Errorf("package name cannot be empty: %w", ErrInvalidInput)
	}
	if ver == "" {
		return fmt.Errorf("version cannot be empty: %w", ErrInvalidInput)
	}
	return nil
}

func (r *SiloRepository) getSourceFilePath(pkg, ver string) string {
	return filepath.Join(r.silo.GetSourcePath(pkg, ver), "source.tar.gz")
}

func (r *SiloRepository) getVersionFilePath(pkg, ver string) string {
	return filepath.Join(r.silo.GetVersionPath(pkg, ver), "version.tar.gz")
}

func (r *SiloRepository) ArchiveExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := r.silo.GetArchivePath(pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetArchivePath(pkg, ver string) string {
	return r.silo.GetArchivePath(pkg, ver)
}

func (r *SiloRepository) StoreArchive(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := r.silo.GetArchivePath(pkg, ver)
	dir := filepath.Dir(path)

	if err := r.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := r.fs.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	return nil
}

func (r *SiloRepository) RetrieveArchive(pkg, ver string) (io.ReadCloser, error) {
	if err := r.validateInput(pkg, ver); err != nil {
		return nil, err
	}

	path := r.silo.GetArchivePath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil, fmt.Errorf("archive not found: %w", ErrNotFound)
	}

	return r.fs.Open(path)
}

func (r *SiloRepository) RemoveArchive(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := r.silo.GetArchivePath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.Remove(path)
}

func (r *SiloRepository) ListArchives() []string {
	return r.listItems(r.silo.CachePath())
}

func (r *SiloRepository) SourceExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := r.getSourceFilePath(pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetSourcePath(pkg, ver string) string {
	return r.silo.GetSourcePath(pkg, ver)
}

func (r *SiloRepository) StoreSource(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := r.silo.GetSourcePath(pkg, ver)

	if err := r.fs.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	destPath := r.getSourceFilePath(pkg, ver)
	file, err := r.fs.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write source: %w", err)
	}

	return nil
}

func (r *SiloRepository) RetrieveSource(pkg, ver string) (io.ReadCloser, error) {
	if err := r.validateInput(pkg, ver); err != nil {
		return nil, err
	}

	path := r.getSourceFilePath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil, fmt.Errorf("source not found: %w", ErrNotFound)
	}

	return r.fs.Open(path)
}

func (r *SiloRepository) RemoveSource(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := r.silo.GetSourcePath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.RemoveAll(path)
}

func (r *SiloRepository) ListSources() []string {
	return r.listItems(r.silo.SourcePath())
}

func (r *SiloRepository) VersionExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := r.getVersionFilePath(pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetVersionPath(pkg, ver string) string {
	return r.silo.GetVersionPath(pkg, ver)
}

func (r *SiloRepository) StoreVersion(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := r.silo.GetVersionPath(pkg, ver)

	if err := r.fs.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	destPath := r.getVersionFilePath(pkg, ver)
	file, err := r.fs.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}

	return nil
}

func (r *SiloRepository) RetrieveVersion(pkg, ver string) (io.ReadCloser, error) {
	if err := r.validateInput(pkg, ver); err != nil {
		return nil, err
	}

	path := r.getVersionFilePath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil, fmt.Errorf("version not found: %w", ErrNotFound)
	}

	return r.fs.Open(path)
}

func (r *SiloRepository) RemoveVersion(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := r.silo.GetVersionPath(pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.RemoveAll(path)
}

func (r *SiloRepository) ListVersions() []string {
	return r.listItems(r.silo.VersionPath())
}

func (r *SiloRepository) FullClean(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	if err := r.RemoveArchive(pkg, ver); err != nil {
		return err
	}
	if err := r.RemoveSource(pkg, ver); err != nil {
		return err
	}
	if err := r.RemoveVersion(pkg, ver); err != nil {
		return err
	}

	return nil
}

func (r *SiloRepository) CleanAll() error {
	paths := []string{
		r.silo.CachePath(),
		r.silo.SourcePath(),
		r.silo.VersionPath(),
	}

	for _, path := range paths {
		if exists, _ := afero.Exists(r.fs, path); exists {
			if err := r.fs.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to clean %s: %w", path, err)
			}
		}
	}

	return nil
}

func (r *SiloRepository) listItems(basePath string) []string {
	var items []string

	entries, err := afero.ReadDir(r.fs, basePath)
	if err != nil {
		return items
	}

	for _, entry := range entries {
		if entry.IsDir() {
			items = append(items, entry.Name())
		}
	}

	return items
}
