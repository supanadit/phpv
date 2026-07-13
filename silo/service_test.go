package silo

import (
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/registry"
)

type mockSiloRepo struct {
	silo          domain.Silo
	state         domain.InstallState
	stateErr      error
	defaultVer    string
	defaultErr    error
	phpOutputPath string
	sourcePath    string
	depPath       string
}

func (m *mockSiloRepo) Download(url, checksumType, checksumValue string) (bool, error) {
	return false, nil
}
func (m *mockSiloRepo) Extract(archivePath, destDir string) (bool, error) {
	return false, nil
}
func (m *mockSiloRepo) GetSilo() domain.Silo {
	return m.silo
}
func (m *mockSiloRepo) GetState(name, version string) (domain.InstallState, error) {
	return m.state, m.stateErr
}
func (m *mockSiloRepo) MarkInProgress(name, version string) error {
	return nil
}
func (m *mockSiloRepo) MarkComplete(name, version string) error {
	return nil
}
func (m *mockSiloRepo) MarkFailed(name, version string) error {
	return nil
}
func (m *mockSiloRepo) GetDefault() (string, error) {
	return m.defaultVer, m.defaultErr
}
func (m *mockSiloRepo) SetDefault(version string) error {
	return nil
}
func (m *mockSiloRepo) PHPOutputPath(phpVersion string) string {
	return m.phpOutputPath
}
func (m *mockSiloRepo) SourcePath(pkg, version string) string {
	return m.sourcePath
}
func (m *mockSiloRepo) PackagePrefix(name, version string) string {
	return "/prefix/" + name + "/" + version
}

type mockRegistryRepo struct{}

func (m *mockRegistryRepo) List(name string, checksum bool, os string) ([]domain.Registry, error) {
	return nil, nil
}
func (m *mockRegistryRepo) Get(name, version string, checksum bool, os string) (domain.Registry, error) {
	return domain.Registry{}, nil
}

func TestService_GetSilo(t *testing.T) {
	mock := &mockSiloRepo{silo: domain.Silo{Root: "/test/root"}}
	svc := NewService(mock, registry.NewService(&mockRegistryRepo{}))

	s := svc.GetSilo()
	if s.Root != "/test/root" {
		t.Fatalf("GetSilo().Root = %q, want /test/root", s.Root)
	}
}

func TestService_GetState(t *testing.T) {
	mock := &mockSiloRepo{state: domain.StateInstalled}
	svc := NewService(mock, registry.NewService(&mockRegistryRepo{}))

	state, err := svc.GetState("php", "8.4.0")
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state != domain.StateInstalled {
		t.Fatalf("GetState = %q, want installed", state)
	}
}

func TestService_GetDefault(t *testing.T) {
	mock := &mockSiloRepo{defaultVer: "8.4.0"}
	svc := NewService(mock, registry.NewService(&mockRegistryRepo{}))

	ver, err := svc.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault returned error: %v", err)
	}
	if ver != "8.4.0" {
		t.Fatalf("GetDefault = %q, want 8.4.0", ver)
	}
}

func TestService_PathHelpers(t *testing.T) {
	mock := &mockSiloRepo{
		phpOutputPath: "/root/packages/php/8.4.0",
		sourcePath:    "/root/sources/php/8.4.0",
	}
	svc := NewService(mock, registry.NewService(&mockRegistryRepo{}))

	if got := svc.PHPOutputPath("8.4.0"); got != "/root/packages/php/8.4.0" {
		t.Fatalf("PHPOutputPath = %q", got)
	}
	if got := svc.SourcePath("php", "8.4.0"); got != "/root/sources/php/8.4.0" {
		t.Fatalf("SourcePath = %q", got)
	}
	if got := svc.PackagePrefix("openssl", "1.1.1w"); got != "/prefix/openssl/1.1.1w" {
		t.Fatalf("PackagePrefix = %q", got)
	}
}
