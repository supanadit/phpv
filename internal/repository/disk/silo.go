package disk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

var (
	ErrNotFound     = errors.New("item not found")
	ErrExists       = errors.New("item already exists")
	ErrInvalidInput = errors.New("invalid input")
)

type SiloRepository struct {
	fs              afero.Fs
	silo            *domain.Silo
	buildToolsMutex sync.Mutex
}

func NewSiloRepository() (*SiloRepository, error) {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		root = filepath.Join(homeDir, ".phpv")
	}

	return &SiloRepository{
		fs:   afero.NewOsFs(),
		silo: &domain.Silo{Root: root},
	}, nil
}

func (r *SiloRepository) GetSilo() (*domain.Silo, error) {
	return r.silo, nil
}

func (r *SiloRepository) EnsurePaths() error {
	paths := []string{
		utils.CachePath(r.silo),
		utils.SourcePath(r.silo),
		utils.VersionPath(r.silo),
		utils.BinPath(r.silo),
		utils.PharPath(r.silo),
	}

	for _, path := range paths {
		if err := r.fs.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("failed to create path %s: %w", path, err)
		}
	}

	return nil
}

func (r *SiloRepository) validateInput(pkg, ver string) error {
	if pkg == "" {
		return fmt.Errorf("package name cannot be empty: %w", ErrInvalidInput)
	}
	if ver == "" {
		return fmt.Errorf("version cannot be empty: %w", ErrInvalidInput)
	}
	return nil
}

func (r *SiloRepository) listItems(basePath string) []string {
	var items []string

	entries, err := afero.ReadDir(r.fs, basePath)
	if err != nil {
		return items
	}

	for _, entry := range entries {
		if entry.IsDir() {
			items = append(items, entry.Name())
		}
	}

	return items
}
