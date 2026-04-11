package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

func (r *SiloRepository) GetDefault() (string, error) {
	defaultPath := filepath.Join(r.silo.Root, "default")
	data, err := afero.ReadFile(r.fs, defaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read default file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (r *SiloRepository) SetDefault(version string) error {
	defaultPath := filepath.Join(r.silo.Root, "default")
	if err := afero.WriteFile(r.fs, defaultPath, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to write default file: %w", err)
	}
	return nil
}
