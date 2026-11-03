package terminal

import (
	"bytes"
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
	if !strings.Contains(err.Error(), "no webserver configured for PHP 7.4.33") {
		t.Fatalf("error = %q, want an install-first hint", err)
	}
	if !strings.Contains(err.Error(), "phpv server install nginx") {
		t.Fatalf("error = %q, want a suggested server", err)
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
	if err == nil || !strings.Contains(err.Error(), "no webserver configured for PHP 8.5.12") {
		t.Fatalf("error = %v, want an install-first hint", err)
	}
}

func TestServerConfiguredFor(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now())

	h := newServerTestHandler(t, root, map[string]string{"server": "caddy"})
	if !h.serverConfiguredFor("8.5.12") {
		t.Fatal("default version should be configured by global config")
	}
	if h.serverConfiguredFor("7.4.33") {
		t.Fatal("non-default version must not inherit global config")
	}
	if err := h.recordServerPreference("7.4.33", "caddy"); err != nil {
		t.Fatal(err)
	}
	if !h.serverConfiguredFor("7.4.33") {
		t.Fatal("recorded preference should configure the version")
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

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()
	w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestNoteMissingServer_WarnsOnlyWhenUnconfigured(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, nil)

	out := captureStderr(t, func() { h.noteMissingServer("7.3.33") })
	if !strings.Contains(out, "no webserver configured for PHP 7.3.33") {
		t.Fatalf("stderr = %q, want an install-first hint", out)
	}
	if !strings.Contains(out, "phpv server install nginx") {
		t.Fatalf("stderr = %q, want an actionable command", out)
	}

	if err := h.recordServerPreference("7.3.33", "nginx"); err != nil {
		t.Fatal(err)
	}
	if out := captureStderr(t, func() { h.noteMissingServer("7.3.33") }); out != "" {
		t.Fatalf("stderr = %q, want no warning once configured", out)
	}
}
