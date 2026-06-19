package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
)

func (r *SiloRepository) getStateFilePath(phpVersion string) string {
	return filepath.Join(silo.PHPVersionPath(r.silo, phpVersion), ".state")
}

func (r *SiloRepository) MarkInProgress(phpVersion string) error {
	versionPath := silo.PHPVersionPath(r.silo, phpVersion)
	if err := r.fs.MkdirAll(versionPath, 0o755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	statePath := r.getStateFilePath(phpVersion)
	if err := afero.WriteFile(r.fs, statePath, []byte("in_progress"), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (r *SiloRepository) MarkComplete(phpVersion string) error {
	statePath := r.getStateFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, statePath); !exists {
		return nil
	}

	if err := afero.WriteFile(r.fs, statePath, []byte("installed"), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (r *SiloRepository) MarkFailed(phpVersion string) error {
	statePath := r.getStateFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, statePath); !exists {
		return nil
	}

	if err := afero.WriteFile(r.fs, statePath, []byte("failed"), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (r *SiloRepository) GetState(phpVersion string) (domain.InstallState, error) {
	statePath := r.getStateFilePath(phpVersion)
	data, err := afero.ReadFile(r.fs, statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.StateNone, nil
		}
		return domain.StateNone, fmt.Errorf("failed to read state file: %w", err)
	}

	state := strings.TrimSpace(string(data))
	switch state {
	case "in_progress":
		return domain.StateInProgress, nil
	case "installed":
		return domain.StateInstalled, nil
	case "failed":
		return domain.StateFailed, nil
	default:
		return domain.StateNone, nil
	}
}

func (r *SiloRepository) Rollback(phpVersion string) error {
	// Only remove PHP build output — preserve successfully built dependencies
	outputPath := silo.PHPOutputPath(r.silo, phpVersion)
	if exists, _ := afero.Exists(r.fs, outputPath); exists {
		if err := r.fs.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("failed to remove output directory: %w", err)
		}
	}

	depInfo := r.getDepsInfoFilePath(phpVersion)
	if exists, _ := afero.Exists(r.fs, depInfo); exists {
		if err := r.fs.Remove(depInfo); err != nil {
			return fmt.Errorf("failed to remove dependency info: %w", err)
		}
	}

	return nil
}
