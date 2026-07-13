package forge

import (
	"testing"
)

type mockForgeRepo struct {
	buildCalled   bool
	installCalled bool
	lastFlags     []string
}

func (m *mockForgeRepo) Build(name, version, sourceDir string, extraEnv, extraConfigureFlags []string, installPrefix string) (string, map[string]string, error) {
	m.buildCalled = true
	m.lastFlags = extraConfigureFlags
	return "/build", map[string]string{"PATH": "/prefix/bin"}, nil
}

func (m *mockForgeRepo) Install(name, version, buildDir, prefix string) error {
	m.installCalled = true
	return nil
}

func TestService_Build_ResolvesPlaceholders(t *testing.T) {
	mock := &mockForgeRepo{}
	svc := NewService(mock)

	flags := []string{"--with-openssl={{prefix}}", "--with-source={{source}}"}
	_, _, err := svc.Build("php", "8.3.0", "/src", nil, flags, "/opt/php/8.3.0")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !mock.buildCalled {
		t.Fatal("Build was not delegated to repo")
	}
	if len(mock.lastFlags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(mock.lastFlags))
	}
	if mock.lastFlags[0] != "--with-openssl=/opt/php/8.3.0" {
		t.Fatalf("flag[0] = %q, want --with-openssl=/opt/php/8.3.0", mock.lastFlags[0])
	}
	if mock.lastFlags[1] != "--with-source=/src" {
		t.Fatalf("flag[1] = %q, want --with-source=/src", mock.lastFlags[1])
	}
}

func TestService_Build_NoPlaceholders(t *testing.T) {
	mock := &mockForgeRepo{}
	svc := NewService(mock)

	flags := []string{"--enable-mbstring", "--with-pdo-mysql=mysqlnd"}
	_, _, err := svc.Build("php", "8.3.0", "/src", nil, flags, "/opt/php/8.3.0")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if mock.lastFlags[0] != "--enable-mbstring" {
		t.Fatalf("flag[0] = %q, want --enable-mbstring", mock.lastFlags[0])
	}
	if mock.lastFlags[1] != "--with-pdo-mysql=mysqlnd" {
		t.Fatalf("flag[1] = %q, want --with-pdo-mysql=mysqlnd", mock.lastFlags[1])
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

func TestResolvePlaceholders_Empty(t *testing.T) {
	result := resolvePlaceholders(nil, "/prefix", "/src")
	if result != nil {
		t.Fatal("expected nil for nil input")
	}
}
