package disk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (r *SiloRepository) getDepsInfoFilePath(phpVersion string) string {
	return filepath.Join(utils.PHPVersionPath(r.silo, phpVersion), ".deps.json")
}

func (r *SiloRepository) SaveDependencyInfo(phpVersion string, deps []domain.DependencyInfo) error {
	versionPath := utils.PHPVersionPath(r.silo, phpVersion)
	if err := r.fs.MkdirAll(versionPath, 0o755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	depInfoPath := r.getDepsInfoFilePath(phpVersion)
	data, err := json.MarshalIndent(deps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dependency info: %w", err)
	}

	if err := afero.WriteFile(r.fs, depInfoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write dependency info: %w", err)
	}

	return nil
}

func (r *SiloRepository) GetDependencyInfo(phpVersion string) ([]domain.DependencyInfo, error) {
	depInfoPath := r.getDepsInfoFilePath(phpVersion)
	data, err := afero.ReadFile(r.fs, depInfoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.DependencyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read dependency info: %w", err)
	}

	var deps []domain.DependencyInfo
	if err := json.Unmarshal(data, &deps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependency info: %w", err)
	}

	return deps, nil
}

func (r *SiloRepository) RemoveDependencyInfo(phpVersion string) error {
	depInfoPath := r.getDepsInfoFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, depInfoPath); !exists {
		return nil
	}

	if err := r.fs.Remove(depInfoPath); err != nil {
		return fmt.Errorf("failed to remove dependency info: %w", err)
	}

	return nil
}
