package disk

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *SiloRepository) SourceExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := r.getSourceFilePath(pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetSourcePath(pkg, ver string) string {
	return utils.GetSourcePath(r.silo, pkg, ver)
}

func (r *SiloRepository) StoreSource(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetSourcePath(r.silo, pkg, ver)

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

	path := utils.GetSourcePath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.RemoveAll(path)
}

func (r *SiloRepository) ListSources() []string {
	return r.listItems(utils.SourcePath(r.silo))
}
