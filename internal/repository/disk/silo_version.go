package disk

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *SiloRepository) VersionExists(pkg, ver string) bool {
	if err := r.validateInput(pkg, ver); err != nil {
		return false
	}
	path := r.getVersionFilePath(pkg, ver)
	exists, _ := afero.Exists(r.fs, path)
	return exists
}

func (r *SiloRepository) GetVersionPath(pkg, ver string) string {
	return utils.GetVersionPath(r.silo, pkg, ver)
}

func (r *SiloRepository) StoreVersion(pkg, ver string, data io.Reader) error {
	if err := r.validateInput(pkg, ver); err != nil {
		return err
	}

	path := utils.GetVersionPath(r.silo, pkg, ver)

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

	path := utils.GetVersionPath(r.silo, pkg, ver)
	if exists, _ := afero.Exists(r.fs, path); !exists {
		return nil
	}

	return r.fs.RemoveAll(path)
}

func (r *SiloRepository) ListVersions() []string {
	basePath := utils.VersionPath(r.silo)
	entries, err := afero.ReadDir(r.fs, basePath)
	if err != nil {
		return nil
	}

	var items []string
	for _, entry := range entries {
		if entry.IsDir() {
			versionPath := filepath.Join(basePath, entry.Name())
			outputPath := filepath.Join(versionPath, "output", "bin", "php")
			if exists, _ := afero.Exists(r.fs, outputPath); exists {
				items = append(items, entry.Name())
			}
		}
	}

	return items
}
