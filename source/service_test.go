package source

import (
	"errors"
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockSourceRepository struct {
	versions []domain.Source
	sources  []domain.Source
	err      error
}

func (m *mockSourceRepository) GetVersions() ([]domain.Source, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.versions, nil
}

func (m *mockSourceRepository) GetSources(name, version string) ([]domain.Source, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sources, nil
}

func TestNewService(t *testing.T) {
	repo := &mockSourceRepository{
		versions: []domain.Source{},
	}
	svc := NewService(repo)

	if svc == nil {
		t.Error("expected service to not be nil")
	}
}

func TestService_GetVersions_Success(t *testing.T) {
	expectedVersions := []domain.Source{
		{Name: "php", Version: "8.2.0", URL: "https://www.php.net/distributions/php-8.2.0.tar.gz"},
		{Name: "php", Version: "8.1.0", URL: "https://www.php.net/distributions/php-8.1.0.tar.gz"},
	}

	repo := &mockSourceRepository{
		versions: expectedVersions,
	}

	svc := NewService(repo)
	versions, err := svc.GetVersions()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}

	if versions[0].Version != "8.2.0" {
		t.Errorf("expected first version to be 8.2.0, got %s", versions[0].Version)
	}
}

func TestService_GetVersions_Error(t *testing.T) {
	expectedErr := errors.New("failed to fetch versions")

	repo := &mockSourceRepository{
		err: expectedErr,
	}

	svc := NewService(repo)
	versions, err := svc.GetVersions()

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}

	if versions != nil {
		t.Error("expected versions to be nil on error")
	}
}

func TestService_GetVersions_Empty(t *testing.T) {
	repo := &mockSourceRepository{
		versions: []domain.Source{},
	}

	svc := NewService(repo)
	versions, err := svc.GetVersions()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}
}
