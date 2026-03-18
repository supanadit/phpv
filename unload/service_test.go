package unload

import (
	"errors"
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockUnloadRepository struct {
	unload *domain.Unload
	err    error
}

func (m *mockUnloadRepository) Unpack(source, destination string) (*domain.Unload, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.unload, nil
}

func TestNewUnloadService(t *testing.T) {
	repo := &mockUnloadRepository{}
	svc := NewService(repo)

	if svc == nil {
		t.Error("expected service to not be nil")
	}

	if svc.unloadRepository != repo {
		t.Error("expected unloadRepository to be set")
	}
}

func TestService_Unpack_Success(t *testing.T) {
	expectedUnload := &domain.Unload{
		Source:      "/tmp/test.tar.gz",
		Destination: "/tmp/output",
		Extracted:   5,
	}

	repo := &mockUnloadRepository{
		unload: expectedUnload,
	}

	svc := NewService(repo)
	unload, err := svc.Unpack("/tmp/test.tar.gz", "/tmp/output")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if unload.Source != expectedUnload.Source {
		t.Errorf("expected Source to be '%s', got '%s'", expectedUnload.Source, unload.Source)
	}

	if unload.Destination != expectedUnload.Destination {
		t.Errorf("expected Destination to be '%s', got '%s'", expectedUnload.Destination, unload.Destination)
	}

	if unload.Extracted != expectedUnload.Extracted {
		t.Errorf("expected Extracted to be %d, got %d", expectedUnload.Extracted, unload.Extracted)
	}
}

func TestService_Unpack_Error(t *testing.T) {
	expectedErr := errors.New("failed to unpack")

	repo := &mockUnloadRepository{
		err: expectedErr,
	}

	svc := NewService(repo)
	unload, err := svc.Unpack("/tmp/test.zip", "/tmp/output")

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}

	if unload != nil {
		t.Error("expected unload to be nil on error")
	}
}
