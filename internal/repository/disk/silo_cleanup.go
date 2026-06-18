package disk

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/silo"
)

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
		silo.CachePath(r.silo),
		silo.SourcePath(r.silo),
		silo.VersionPath(r.silo),
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
