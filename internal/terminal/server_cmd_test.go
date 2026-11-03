package terminal

import "testing"

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
