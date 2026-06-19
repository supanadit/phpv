package disk

import (
	"errors"
	"fmt"
	"sync"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/domain"
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

func NewSiloRepository(root string) (*SiloRepository, error) {
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
		silo.CachePath(r.silo),
		silo.SourcePath(r.silo),
		silo.VersionPath(r.silo),
		silo.BinPath(r.silo),
		silo.PharPath(r.silo),
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
