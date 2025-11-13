package repository

import (
	"context"
	"sync"

	"github.com/supanadit/phpv/domain"
)

// InMemoryPHPVersionRepository implements PHPVersionRepository with in-memory storage
type InMemoryPHPVersionRepository struct {
	mu       sync.RWMutex
	versions map[string]domain.PHPVersion
}

// NewInMemoryPHPVersionRepository creates a new in-memory PHP version repository
func NewInMemoryPHPVersionRepository() *InMemoryPHPVersionRepository {
	return &InMemoryPHPVersionRepository{
		versions: make(map[string]domain.PHPVersion),
	}
}

// GetAvailableVersions returns all available PHP versions
func (r *InMemoryPHPVersionRepository) GetAvailableVersions(ctx context.Context) ([]domain.PHPVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions := make([]domain.PHPVersion, 0, len(r.versions))
	for _, v := range r.versions {
		versions = append(versions, v)
	}
	return versions, nil
}

// GetVersionByString returns a PHP version by its string representation
func (r *InMemoryPHPVersionRepository) GetVersionByString(ctx context.Context, version string) (domain.PHPVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if v, exists := r.versions[version]; exists {
		return v, nil
	}
	return domain.PHPVersion{}, domain.ErrNotFound
}

// SaveVersion saves a PHP version
func (r *InMemoryPHPVersionRepository) SaveVersion(ctx context.Context, version domain.PHPVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.versions[version.Version] = version
	return nil
}
