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
	t.Setenv(defaultSAPIEnv, "")
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

func TestResolveServerForPHP_FallsBackToFirstInstalled(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now().Add(-2*time.Hour))
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now().Add(-time.Hour))

	h := newServerTestHandler(t, root, nil)

	// No association, no env and no global default: the deterministic
	// fallback is the first installed server in ServerNames order, never
	// "newest installed".
	got, source, err := h.resolveServerForPHP("", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "caddy" || source != serverFromFallback {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (caddy, %q)", got, source, serverFromFallback)
	}
}

func TestResolveServerForPHP_GlobalDefaultAppliesToAllVersions(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now())
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	// `phpv server default nginx` persists config server=nginx.
	h := newServerTestHandler(t, root, map[string]string{"server": "nginx"})

	got, source, err := h.resolveServerForPHP("", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "nginx" || source != serverFromConfig {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (nginx, %q)", got, source, serverFromConfig)
	}
}

func TestResolveServerForPHP_FallbackUsesOnlyInstalledServer(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, nil)

	got, source, err := h.resolveServerForPHP("", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "nginx" || source != serverFromFallback {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (nginx, %q)", got, source, serverFromFallback)
	}
}

func TestResolveServerForPHP_NoServersInstalled(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")

	h := newServerTestHandler(t, root, nil)

	_, _, err := h.resolveServerForPHP("", "8.5.12")
	if err == nil {
		t.Fatal("expected an error when no webserver is installed")
	}
	if !strings.Contains(err.Error(), "no webserver installed — run: phpv server install <caddy|nginx|apache>") {
		t.Fatalf("error = %q, want the install command", err)
	}
}

func TestResolveServerForPHP_EnvDefaultForUnassociatedVersion(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "caddy", "2.10.2", time.Now())
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	// Env wins over the persisted global default.
	h := newServerTestHandler(t, root, map[string]string{"server": "nginx"})
	t.Setenv(defaultServerEnv, "caddy")

	got, source, err := h.resolveServerForPHP("", "7.4.33")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "caddy" || source != serverFromEnv {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (caddy, %q)", got, source, serverFromEnv)
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

func TestResolveServerForPHP_DefaultWithoutConfigFallsBack(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	if err := os.WriteFile(filepath.Join(root, "default"), []byte("8.5.12\n"), 0644); err != nil {
		t.Fatal(err)
	}
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())

	h := newServerTestHandler(t, root, nil)

	got, source, err := h.resolveServerForPHP("", "8.5.12")
	if err != nil {
		t.Fatalf("resolveServerForPHP: %v", err)
	}
	if got != "nginx" || source != serverFromFallback {
		t.Fatalf("resolveServerForPHP = (%q, %q), want (nginx, %q)", got, source, serverFromFallback)
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

func TestResolveServerForPHP_InvalidServerNameHintsPHPVersionMixup(t *testing.T) {
	h := &PHPHandler{}

	_, _, err := h.resolveServerForPHP("7", "8.5.12")
	if err == nil {
		t.Fatal("expected an error for a PHP version passed as a server name")
	}
	if !strings.Contains(err.Error(), "PHP versions are switched with `phpv use <version>`") {
		t.Fatalf("error = %q, want the version mix-up hint", err)
	}
}

func TestResolveSAPIForPHP_DefaultsToFPM(t *testing.T) {
	root := t.TempDir()
	h := newServerTestHandler(t, root, nil)

	sapi, err := h.resolveSAPIForPHP("", "apache", "8.5.12")
	if err != nil {
		t.Fatalf("resolveSAPIForPHP: %v", err)
	}
	if sapi != "fpm" {
		t.Fatalf("sapi = %q, want fpm", sapi)
	}
}

func TestResolveSAPIForPHP_PerVersionAssociation(t *testing.T) {
	root := t.TempDir()
	h := newServerTestHandler(t, root, nil)
	if err := h.recordServerSelection("8.5.12", "apache", "modphp"); err != nil {
		t.Fatal(err)
	}

	sapi, err := h.resolveSAPIForPHP("", "apache", "8.5.12")
	if err != nil {
		t.Fatalf("resolveSAPIForPHP: %v", err)
	}
	if sapi != "modphp" {
		t.Fatalf("sapi = %q, want modphp", sapi)
	}
	// Another version is unaffected.
	if sapi, _ := h.resolveSAPIForPHP("", "apache", "7.4.33"); sapi != "fpm" {
		t.Fatalf("sapi for unassociated version = %q, want fpm", sapi)
	}
}

func TestResolveSAPIForPHP_EnvAndConfig(t *testing.T) {
	root := t.TempDir()
	h := newServerTestHandler(t, root, map[string]string{"server_sapi": "modphp"})
	t.Setenv(defaultSAPIEnv, "fpm")

	sapi, err := h.resolveSAPIForPHP("", "apache", "8.5.12")
	if err != nil {
		t.Fatalf("resolveSAPIForPHP: %v", err)
	}
	if sapi != "fpm" {
		t.Fatalf("env SAPI should win: got %q", sapi)
	}

	t.Setenv(defaultSAPIEnv, "")
	sapi, err = h.resolveSAPIForPHP("", "apache", "8.5.12")
	if err != nil {
		t.Fatalf("resolveSAPIForPHP: %v", err)
	}
	if sapi != "modphp" {
		t.Fatalf("config SAPI should apply: got %q", sapi)
	}
	// mod_php cannot apply to a non-Apache fallback; fall back to FPM.
	if sapi, err := h.resolveSAPIForPHP("", "caddy", "8.5.12"); err != nil || sapi != "fpm" {
		t.Fatalf("non-apache config SAPI = (%q, %v), want fpm", sapi, err)
	}
}

func TestResolveSAPIForPHP_FlagWinsAndValidates(t *testing.T) {
	root := t.TempDir()
	h := newServerTestHandler(t, root, map[string]string{"server_sapi": "fpm"})

	sapi, err := h.resolveSAPIForPHP("modphp", "apache", "8.5.12")
	if err != nil || sapi != "modphp" {
		t.Fatalf("flag SAPI = (%q, %v), want modphp", sapi, err)
	}
	if _, err := h.resolveSAPIForPHP("modphp", "nginx", "8.5.12"); err == nil {
		t.Fatal("modphp with nginx must error")
	}
	if _, err := h.resolveSAPIForPHP("bogus", "apache", "8.5.12"); err == nil {
		t.Fatal("unknown SAPI must error")
	}
}

func TestServerPreferences_BackwardCompatibleSAPI(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PHPV_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "servers"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `{"servers":{"8.5.12":"apache"}}`
	if err := os.WriteFile(filepath.Join(root, "servers", "preferences.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newTestPHPHandler(root)
	if got := h.serverSAPI("8.5.12"); got != "fpm" {
		t.Fatalf("legacy SAPI = %q, want fpm", got)
	}
}
