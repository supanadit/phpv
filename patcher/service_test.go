package patcher

import (
	"testing"
)

type mockPatcherRepo struct {
	called bool
}

func (m *mockPatcherRepo) PatchesFor(name, version string) []Patch {
	m.called = true
	return nil
}

func TestService_PatchesFor_Delegates(t *testing.T) {
	mock := &mockPatcherRepo{}
	svc := NewService(mock)

	patches := svc.PatchesFor("php", "8.3.0")
	if patches == nil {
		// nil is valid — just means no patches
	}
	if !mock.called {
		t.Fatal("PatchesFor was not delegated to repo")
	}
}
