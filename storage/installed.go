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

func (s *InstalledStorage) Remove(version domain.Version) error {
	versionDir := filepath.Join(s.root, "versions", version.String())

	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("PHP version %s is not installed", version.String())
	}

	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove PHP version %s: %w", version.String(), err)
	}

	return nil
}

func (s *InstalledStorage) RemoveDependencies(version domain.Version) error {
	versionStr := version.String()

	depsDir := filepath.Join(s.root, "dependencies", versionStr)
	if _, err := os.Stat(depsDir); err == nil {
		if err := os.RemoveAll(depsDir); err != nil {
			return fmt.Errorf("failed to remove dependencies directory: %w", err)
		}
	}

	depsSrcDir := filepath.Join(s.root, "dependencies-src", versionStr)
	if _, err := os.Stat(depsSrcDir); err == nil {
		if err := os.RemoveAll(depsSrcDir); err != nil {
			return fmt.Errorf("failed to remove dependencies-src directory: %w", err)
		}
	}

	return nil
}
