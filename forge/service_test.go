package forge

import (
	"testing"
)

type mockForgeRepo struct {
	buildCalled   bool
	installCalled bool
}

func (m *mockForgeRepo) Build(name, version, sourceDir string, extraEnv, extraConfigureFlags []string, installPrefix string) (string, map[string]string, error) {
	m.buildCalled = true
	return "/build", map[string]string{"PATH": "/prefix/bin"}, nil
}

func (m *mockForgeRepo) Install(name, version, buildDir, prefix string) error {
	m.installCalled = true
	return nil
}

func TestService_Build_Delegates(t *testing.T) {
	mock := &mockForgeRepo{}
	svc := NewService(mock)

	_, _, err := svc.Build("php", "8.3.0", "/src", nil, nil, "/prefix")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !mock.buildCalled {
		t.Fatal("Build was not delegated to repo")
	}
}

func TestService_Install_Delegates(t *testing.T) {
	mock := &mockForgeRepo{}
	svc := NewService(mock)

	err := svc.Install("php", "8.3.0", "/build", "/prefix")
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if !mock.installCalled {
		t.Fatal("Install was not delegated to repo")
	}
}
