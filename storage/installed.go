package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
)

type InstalledStorage struct {
	root string
}

func NewInstalledStorage() *InstalledStorage {
	return &InstalledStorage{
		root: viper.GetString("PHPV_ROOT"),
	}
}

func (s *InstalledStorage) List(ctx context.Context) ([]domain.Version, error) {
	versionsDir := filepath.Join(s.root, "versions")

	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Version{}, nil
		}
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	versions := make([]domain.Version, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		versionName := entry.Name()
		version, err := domain.ParseVersion(versionName)
		if err != nil {
			continue
		}

		phpBinary := filepath.Join(versionsDir, versionName, "bin", "php")
		if _, err := os.Stat(phpBinary); err != nil {
			continue
		}

		versions = append(versions, version)
	}

	sort.Slice(versions, func(i, j int) bool {
		if versions[i].Major != versions[j].Major {
			return versions[i].Major > versions[j].Major
		}
		if versions[i].Minor != versions[j].Minor {
			return versions[i].Minor > versions[j].Minor
		}
		return versions[i].Patch > versions[j].Patch
	})

	return versions, nil
}

func (s *InstalledStorage) GetPath(version domain.Version) (string, error) {
	versionDir := filepath.Join(s.root, "versions", version.String())
	phpBinary := filepath.Join(versionDir, "bin", "php")

	if _, err := os.Stat(phpBinary); err != nil {
		return "", fmt.Errorf("PHP version %s is not installed", version.String())
	}

	return versionDir, nil
}

func (s *InstalledStorage) Exists(version domain.Version) (bool, error) {
	_, err := s.GetPath(version)
	if err != nil {
		return false, nil
	}
	return true, nil
}
