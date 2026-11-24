package disk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

// PHP 5.4's install-cli and install-cgi both call an shtool mkdir -p wrapper.
// Under make -jN the check-then-mkdir races and one target fails with
// "File exists". Installation must stay serial even when the build is not.
func TestForgeInstallRunsMakeSerially(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(root, "make-args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$MAKE_ARGS_FILE\"\n"
	if err := os.WriteFile(filepath.Join(bin, "make"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("MAKE_ARGS_FILE", argsFile)

	prefix := filepath.Join(root, "prefix")
	if err := (&ForgeRepository{}).Install(context.Background(), domain.InstallRequest{
		Name: "php", BuildDir: root, Prefix: prefix, Jobs: 8,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := "-j1\ninstall\n"; string(got) != want {
		t.Fatalf("make install args = %q, want %q", strings.ReplaceAll(string(got), "\n", " "), strings.ReplaceAll(want, "\n", " "))
	}
}
