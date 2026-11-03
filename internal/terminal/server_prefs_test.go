package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeConfigService struct{ data map[string]string }

func (f *fakeConfigService) Get(key string) (string, error) { return f.data[key], nil }
func (f *fakeConfigService) Set(key, value string) error {
	if f.data == nil {
		f.data = map[string]string{}
	}
	f.data[key] = value
	return nil
}
func (f *fakeConfigService) List() ([]string, error) { return nil, nil }

// createFakeServerInstall lays down a packages/<server>/<version> tree with a
// chosen install time so newest-installed fallback is deterministic.
func createFakeServerInstall(t *testing.T, root, server, version string, installedAt time.Time) {
	t.Helper()
	dir := filepath.Join(root, "packages", server, version)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", server), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if installedAt.IsZero() {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, ".state"), []byte("installed."), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, filepath.Join(dir, ".state")} {
		if err := os.Chtimes(path, installedAt, installedAt); err != nil {
			t.Fatal(err)
		}
	}
}

func newServerTestHandler(t *testing.T, root string, config map[string]string) *PHPHandler {
	t.Helper()
	t.Setenv("PHPV_ROOT", root)
	t.Setenv(defaultServerEnv, "")
	h := newTestPHPHandler(root)
	h.configSvc = &fakeConfigService{data: config}
	return h
}

func TestResolveServerForPHP_PreferenceWinsOverGlobalConfig(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "7.4.33")
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now().Add(-2*time.Hour))
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now().Add(-time.Hour))

	h := newServerTestHandler(t, root, map[string]string{"server": "caddy"})
	if err := h.recordServerPreference("7.4.33", "nginx"); err != nil {
		t.Fatalf("recordServerPreference: %v", err)
	}

	got, source, err := h.resolveServerForPHP("", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "nginx" || source != serverFromPHPPreference {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (nginx, %q)", got, source, serverFromPHPPreference)
	}
}

func TestResolveServerForPHP_NonDefaultWithoutPreferenceErrors(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now().Add(-2*time.Hour))
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now().Add(-time.Hour))

	h := newServerTestHandler(t, root, map[string]string{"server": "caddy"})

	_, _, err := h.resolveServerForPHP("", "7.4.33")
	if err == nil {
		t.Fatal("expected an error for a PHP version with no server configured")
	}
	if !strings.Contains(err.Error(), "no webserver associated with PHP 7.4.33") {
		t.Fatalf("error = %q, want an association-required hint", err)
	}
	if !strings.Contains(err.Error(), "installed: caddy, nginx") {
		t.Fatalf("error = %q, want the installed servers listed", err)
	}
	if !strings.Contains(err.Error(), "phpv server use <name> --php 7.4.33") {
		t.Fatalf("error = %q, want an association command", err)
	}
	if !strings.Contains(err.Error(), "PHPV_DEFAULT_SERVER") {
		t.Fatalf("error = %q, want the default-server hint", err)
	}
}

func TestResolveServerForPHP_SingleInstalledServerStillRequiresAssociation(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Only nginx exists globally, but nothing associates it with 7.4.33.
	// Isolation wins: the version must not borrow another version's server.
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, nil)

	_, _, err := h.resolveServerForPHP("", "7.4.33")
	if err == nil {
		t.Fatal("a single globally installed server must not be reused across PHP versions")
	}
	if !strings.Contains(err.Error(), "no webserver associated with PHP 7.4.33") {
		t.Fatalf("error = %q, want an association-required hint", err)
	}
}

func TestResolveServerForPHP_EnvDefaultForUnassociatedVersion(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, nil)
	t.Setenv(defaultServerEnv, "nginx")

	got, source, err := h.resolveServerForPHP("", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "nginx" || source != serverFromEnv {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (nginx, %q)", got, source, serverFromEnv)
	}
}

func TestResolveServerForPHP_EnvDefaultMustBeInstalled(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, nil)
	t.Setenv(defaultServerEnv, "apache")

	_, _, err := h.resolveServerForPHP("", "7.4.33")
	if err == nil || !strings.Contains(err.Error(), "PHPV_DEFAULT_SERVER=apache is set but not installed") {
		t.Fatalf("error = %v, want an env-default hint", err)
	}
}

func TestResolveServerForPHP_AssociationOverridesEnvDefault(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())
	createFakeServerInstall(t, root, "apache", "2.4.62", time.Now())

	h := newServerTestHandler(t, root, nil)
	t.Setenv(defaultServerEnv, "nginx")
	if err := h.recordServerPreference("7.4.33", "apache"); err != nil {
		t.Fatal(err)
	}

	got, source, err := h.resolveServerForPHP("", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "apache" || source != serverFromPHPPreference {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (apache, %q)", got, source, serverFromPHPPreference)
	}
}

func TestResolveServerForPHP_DefaultWithoutConfigOrPreferenceErrors(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, nil)

	_, _, err := h.resolveServerForPHP("", "8.5.12")
	if err == nil || !strings.Contains(err.Error(), "no webserver associated with PHP 8.5.12") {
		t.Fatalf("error = %v, want an association-required hint", err)
	}
}

func TestResolveServerForPHP_DefaultUsesGlobalConfig(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now().Add(-time.Hour))
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, map[string]string{"server": "caddy"})

	got, source, err := h.resolveServerForPHP("", "8.5.12")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "caddy" || source != serverFromConfig {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (caddy, %q)", got, source, serverFromConfig)
	}
}

func TestResolveServerForPHP_FlagWins(t *testing.T) {
	root := t.TempDir()
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now())

	h := newServerTestHandler(t, root, map[string]string{"server": "caddy"})

	got, source, err := h.resolveServerForPHP("nginx", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "nginx" || source != serverFromFlag {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (nginx, %q)", got, source, serverFromFlag)
	}
}

func TestResolveServerPHPVersion_HonorsSession(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "7.4.33")
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PHPV_ROOT", root)
	t.Setenv("PHPV_CURRENT", "7.4.33")

	h := newTestPHPHandler(root)
	got, err := h.resolveServerPHPVersion("")
	if err != nil {
		t.Fatalf("resolveServerPHPVersion: %v", err)
	}
	if got != "7.4.33" {
		t.Fatalf("resolveServerPHPVersion = %q, want 7.4.33", got)
	}
}

func TestServerPreferenceRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	h := newTestPHPHandler(root)

	if _, ok := h.serverPreference("7.4.33"); ok {
		t.Fatal("expected no preference before recording one")
	}
	if err := h.recordServerPreference("7.4.33", "nginx"); err != nil {
		t.Fatalf("recordServerPreference: %v", err)
	}
	got, ok := h.serverPreference("7.4.33")
	if !ok || got != "nginx" {
		t.Fatalf("serverPreference = (%q, %v), want (nginx, true)", got, ok)
	}
	if _, ok := h.serverPreference("8.5.12"); ok {
		t.Fatal("preference leaked to another PHP version")
	}
}
