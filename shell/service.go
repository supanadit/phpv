package shell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/storage"
)

type Service struct {
	installedStorage *storage.InstalledStorage
	defaultStorage   *storage.DefaultVersionStorage
	root             string
}

func NewService() *Service {
	return &Service{
		installedStorage: storage.NewInstalledStorage(),
		defaultStorage:   storage.NewDefaultVersionStorage(),
		root:             viper.GetString("PHPV_ROOT"),
	}
}

func (s *Service) Use(ctx context.Context, versionSpec string) (string, error) {
	version, err := s.resolveVersion(ctx, versionSpec)
	if err != nil {
		return "", err
	}

	exists, err := s.installedStorage.Exists(version)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("version %s is not installed", versionSpec)
	}

	output := fmt.Sprintf("export PHPV_VERSION=%s", version.String())
	return output, nil
}

func (s *Service) Unuse(ctx context.Context) string {
	return "unset PHPV_VERSION"
}

func (s *Service) SetDefault(ctx context.Context, versionSpec string) error {
	version, err := s.resolveVersion(ctx, versionSpec)
	if err != nil {
		return err
	}

	exists, err := s.installedStorage.Exists(version)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("version %s is not installed", versionSpec)
	}

	return s.defaultStorage.Set(version)
}

func (s *Service) GetDefault(ctx context.Context) (*domain.Version, error) {
	return s.defaultStorage.Get()
}

func (s *Service) UnsetDefault(ctx context.Context) error {
	return s.defaultStorage.Unset()
}

func (s *Service) GetCurrent(ctx context.Context) (*domain.Version, error) {
	currentVersion := os.Getenv("PHPV_VERSION")
	if currentVersion != "" {
		version, err := domain.ParseVersion(currentVersion)
		if err == nil {
			return &version, nil
		}
	}

	defaultVersion, err := s.defaultStorage.Get()
	if err != nil {
		return nil, err
	}
	if defaultVersion != nil {
		return defaultVersion, nil
	}

	return nil, nil
}

func (s *Service) Which(ctx context.Context) (string, error) {
	current, err := s.GetCurrent(ctx)
	if err != nil {
		return "", err
	}
	if current == nil {
		return "", fmt.Errorf("no PHP version selected")
	}

	versionPath, err := s.installedStorage.GetPath(*current)
	if err != nil {
		return "", err
	}

	return filepath.Join(versionPath, "bin", "php"), nil
}

func (s *Service) ListInstalled(ctx context.Context) ([]domain.Version, error) {
	return s.installedStorage.List(ctx)
}

func (s *Service) resolveVersion(ctx context.Context, versionSpec string) (domain.Version, error) {
	installed, err := s.installedStorage.List(ctx)
	if err != nil {
		return domain.Version{}, err
	}

	spec := strings.TrimSpace(versionSpec)

	for _, v := range installed {
		if v.String() == spec {
			return v, nil
		}

		matched := false
		parts := strings.Split(spec, ".")
		if len(parts) >= 1 {
			maj, err := strconv.Atoi(parts[0])
			if err == nil && maj == v.Major {
				if len(parts) == 1 {
					matched = true
				} else if len(parts) >= 2 {
					min, err := strconv.Atoi(parts[1])
					if err == nil && min == v.Minor {
						if len(parts) == 2 {
							matched = true
						} else if len(parts) >= 3 {
							pat, err := strconv.Atoi(parts[2])
							if err == nil && pat == v.Patch {
								matched = true
							}
						}
					}
				}
			}
		}

		if matched {
			return v, nil
		}
	}

	if len(installed) > 0 {
		suggestion := installed[0].String()
		return domain.Version{}, fmt.Errorf("version %s not found. Installed versions: %s. Did you mean: %s?",
			spec, formatVersions(installed), suggestion)
	}

	return domain.Version{}, fmt.Errorf("version %s not found. No versions installed.", spec)
}

func formatVersions(versions []domain.Version) string {
	strs := make([]string, len(versions))
	for i, v := range versions {
		strs[i] = v.String()
	}
	return strings.Join(strs, ", ")
}
