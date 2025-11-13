package repository

import (
	"context"
	"sync"

	"github.com/supanadit/phpv/domain"
)

// InMemoryInstallationRepository implements InstallationRepository with in-memory storage
type InMemoryInstallationRepository struct {
	mu            sync.RWMutex
	installations map[string]domain.Installation
	activeVersion string
}

// NewInMemoryInstallationRepository creates a new in-memory installation repository
func NewInMemoryInstallationRepository() *InMemoryInstallationRepository {
	return &InMemoryInstallationRepository{
		installations: make(map[string]domain.Installation),
	}
}

// GetAllInstallations returns all installations
func (r *InMemoryInstallationRepository) GetAllInstallations(ctx context.Context) ([]domain.Installation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	installations := make([]domain.Installation, 0, len(r.installations))
	for _, inst := range r.installations {
		installations = append(installations, inst)
	}
	return installations, nil
}

// GetInstallationByVersion returns an installation by version
func (r *InMemoryInstallationRepository) GetInstallationByVersion(ctx context.Context, version domain.PHPVersion) (domain.Installation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if inst, exists := r.installations[version.Version]; exists {
		return inst, nil
	}
	return domain.Installation{}, domain.ErrNotFound
}

// GetActiveInstallation returns the active installation
func (r *InMemoryInstallationRepository) GetActiveInstallation(ctx context.Context) (domain.Installation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.activeVersion == "" {
		return domain.Installation{}, domain.ErrNotFound
	}

	if inst, exists := r.installations[r.activeVersion]; exists {
		return inst, nil
	}
	return domain.Installation{}, domain.ErrNotFound
}

// SaveInstallation saves an installation
func (r *InMemoryInstallationRepository) SaveInstallation(ctx context.Context, installation domain.Installation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.installations[installation.Version.Version] = installation
	return nil
}

// SetActiveInstallation sets the active installation
func (r *InMemoryInstallationRepository) SetActiveInstallation(ctx context.Context, installation domain.Installation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.activeVersion = installation.Version.Version
	r.installations[installation.Version.Version] = installation
	return nil
}

// DeleteInstallation deletes an installation
func (r *InMemoryInstallationRepository) DeleteInstallation(ctx context.Context, version domain.PHPVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.activeVersion == version.Version {
		r.activeVersion = ""
	}

	delete(r.installations, version.Version)
	return nil
}
