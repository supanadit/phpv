package terminal

import (
	"io"
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
}
