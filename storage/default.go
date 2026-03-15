package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
)

type DefaultVersionStorage struct {
	root string
}

func NewDefaultVersionStorage() *DefaultVersionStorage {
	return &DefaultVersionStorage{
		root: viper.GetString("PHPV_ROOT"),
	}
}

func (s *DefaultVersionStorage) Get() (*domain.Version, error) {
	versionFile := filepath.Join(s.root, "version")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read version file: %w", err)
	}

	versionStr := string(data)
	versionStr = filepath.Base(versionStr)
	versionStr = filepath.Clean(versionStr)

	if versionStr == "" || versionStr == "." {
		return nil, nil
	}

	version, err := domain.ParseVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version from file: %w", err)
	}

	return &version, nil
}

func (s *DefaultVersionStorage) Set(version domain.Version) error {
	versionFile := filepath.Join(s.root, "version")
	versionStr := version.String()

	if version.Extra != "" {
		versionStr = fmt.Sprintf("%d.%d.%d-%s", version.Major, version.Minor, version.Patch, version.Extra)
	} else {
		versionStr = fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)
	}

	if err := os.WriteFile(versionFile, []byte(versionStr), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}

func (s *DefaultVersionStorage) Unset() error {
	versionFile := filepath.Join(s.root, "version")
	if err := os.Remove(versionFile); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove version file: %w", err)
	}
	return nil
}
