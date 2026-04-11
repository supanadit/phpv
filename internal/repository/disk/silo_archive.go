package disk

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *SiloRepository) getSourceFilePath(pkg, ver string) string {
	return filepath.Join(utils.GetSourcePath(r.silo, pkg, ver), "source.tar.gz")
}

func (r *SiloRepository) getVersionFilePath(pkg, ver string) string {
	return filepath.Join(utils.GetVersionPath(r.silo, pkg, ver), "version.tar.gz")
}

func (r *SiloRepository) ArchiveExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := utils.GetArchivePath(r.silo, pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetArchivePath(pkg, ver string) string {
	return utils.GetArchivePath(r.silo, pkg, ver)
}

func (r *SiloRepository) StoreArchive(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetArchivePath(r.silo, pkg, ver)
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

	path := utils.GetArchivePath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil, fmt.Errorf("archive not found: %w", ErrNotFound)
	}

	return r.fs.Open(path)
}

func (r *SiloRepository) RemoveArchive(pkg, ver string) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetArchivePath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.Remove(path)
}

func (r *SiloRepository) ListArchives() []string {
	return r.listItems(utils.CachePath(r.silo))
}
