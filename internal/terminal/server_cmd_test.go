package terminal

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Regression: `server install` used to silently associate the installed
// server with the active PHP version, making every install change the default.
// Installation and association must stay separate commands.
func TestServerCmd_InstallDoesNotAssociate(t *testing.T) {
	root := (&PHPHandler{}).serverCmd()

	install, _, err := root.Find([]string{"install"})
	if err != nil {
		t.Fatalf("find install: %v", err)
	}
	if install.Name() != "install" {
		t.Fatalf("found %q, want install", install.Name())
	}
	if install.Flags().Lookup("php") != nil {
		t.Fatal("server install must not take --php; association belongs to `phpv server use`")
	}

	use, _, err := root.Find([]string{"use"})
	if err != nil {
		t.Fatalf("find use: %v", err)
	}
	if use.Name() != "use" {
		t.Fatalf("found %q, want use", use.Name())
	}
	if use.Flags().Lookup("php") == nil {
		t.Fatal("server use must accept --php")
	}
}

// The global default is a separate, persistent command; it must exist and
// write the `server` config key.
func TestServerDefaultCommand_SetsGlobalConfig(t *testing.T) {
	root := t.TempDir()
	createFakeServerInstall(t, root, "nginx", "1.28.0", time.Now())
	h := newServerTestHandler(t, root, nil)

	cmd := h.serverCmd()
	def, _, err := cmd.Find([]string{"default"})
	if err != nil {
		t.Fatalf("find default: %v", err)
	}
	if def.Name() != "default" {
		t.Fatalf("found %q, want default", def.Name())
	}
	def.SetOut(io.Discard)
	def.SetErr(io.Discard)
	if err := def.RunE(def, []string{"nginx"}); err != nil {
		t.Fatalf("server default nginx: %v", err)
	}

	got, err := h.configSvc.Get("server")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if got != "nginx" {
		t.Fatalf("config server = %q, want nginx", got)
	}
	if sapi, _ := h.configSvc.Get("server_sapi"); sapi != "fpm" {
		t.Fatalf("config server_sapi = %q, want fpm", sapi)
	}
}

func TestServerUseCommand_StoresPerVersionSAPI(t *testing.T) {
	root := t.TempDir()
	createFakePHPInstall(t, root, "8.5.12")
	createFakeServerInstall(t, root, "apache", "2.4.62", time.Now())
	h := newServerTestHandler(t, root, nil)

	cmd := h.serverCmd()
	use, _, err := cmd.Find([]string{"use"})
	if err != nil {
		t.Fatalf("find use: %v", err)
	}
	if err := use.Flags().Set("php", "8.5.12"); err != nil {
		t.Fatal(err)
	}
	if err := use.Flags().Set("sapi", "modphp"); err != nil {
		t.Fatal(err)
	}
	if err := use.RunE(use, []string{"apache"}); err != nil {
		t.Fatalf("server use: %v", err)
	}
	if got := h.serverSAPI("8.5.12"); got != "modphp" {
		t.Fatalf("stored SAPI = %q, want modphp", got)
	}
	if got, _ := h.serverPreference("8.5.12"); got != "apache" {
		t.Fatalf("stored server = %q, want apache", got)
	}

	// A plain `server use` clears the mod_php choice (back to FPM).
	if err := use.Flags().Set("sapi", ""); err != nil {
		t.Fatal(err)
	}
	if err := use.RunE(use, []string{"apache"}); err != nil {
		t.Fatalf("server use without --sapi: %v", err)
	}
	if got := h.serverSAPI("8.5.12"); got != "fpm" {
		t.Fatalf("stored SAPI after reset = %q, want fpm", got)
	}
}

func TestServerDefaultCommand_ModPHP(t *testing.T) {
	root := t.TempDir()
	createFakeServerInstall(t, root, "apache", "2.4.62", time.Now())
	h := newServerTestHandler(t, root, nil)

	cmd := h.serverCmd()
	def, _, err := cmd.Find([]string{"default"})
	if err != nil {
		t.Fatalf("find default: %v", err)
	}
	if err := def.Flags().Set("sapi", "modphp"); err != nil {
		t.Fatal(err)
	}
	if err := def.RunE(def, []string{"apache"}); err != nil {
		t.Fatalf("server default apache --sapi modphp: %v", err)
	}
	if got, _ := h.configSvc.Get("server"); got != "apache" {
		t.Fatalf("config server = %q, want apache", got)
	}
	if got, _ := h.configSvc.Get("server_sapi"); got != "modphp" {
		t.Fatalf("config server_sapi = %q, want modphp", got)
	}
}

func TestApacheThreadedDetectsMPM(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	httpd := filepath.Join(dir, "bin", "httpd")
	if err := os.WriteFile(httpd, []byte("#!/bin/sh\necho '  threaded:     yes'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !apacheThreaded(dir) {
		t.Fatal("threaded httpd not detected")
	}
	if err := os.WriteFile(httpd, []byte("#!/bin/sh\necho '  threaded:     no'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if apacheThreaded(dir) {
		t.Fatal("prefork httpd misdetected as threaded")
	}
}
