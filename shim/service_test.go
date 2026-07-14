package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidVersion(t *testing.T) {
	cases := []struct {
		v    string
		want bool
	}{
		{"8.4.4", true},
		{"8.4", true},
		{"8", true},
		{"0.1.0", true},
		{"", false},
		{"; rm -rf /", false},
		{"$(whoami)", false},
		{"8.4.4; echo pwned", false},
		{"a.b.c", false},
		{"8.4.4.4", true},
	}
	for _, c := range cases {
		got := IsValidVersion(c.v)
		if got != c.want {
			t.Errorf("IsValidVersion(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestRenderShim_Binary(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, "#!/bin/bash") {
		t.Error("shim should start with #!/bin/bash")
	}
	if !strings.Contains(content, "PHPV_ROOT") {
		t.Error("shim should reference PHPV_ROOT")
	}
	if !strings.Contains(content, "PHPV_CURRENT") {
		t.Error("shim should reference PHPV_CURRENT")
	}
	if !strings.Contains(content, "LD_LIBRARY_PATH") {
		t.Error("shim should set LD_LIBRARY_PATH")
	}
	if !strings.Contains(content, `exec "$PHPV_PREFIX/bin/php" "$@"`) {
		t.Error("binary shim should exec php from prefix")
	}
	if !strings.Contains(content, "command -v php") {
		t.Error("binary shim should have system block with command -v")
	}
}

func TestRenderShim_DropsParentLdLibraryPath(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if strings.Contains(content, `"$PHPV_PREFIX/lib:$LD_LIBRARY_PATH"`) {
		t.Error("shim should NOT prepend to parent LD_LIBRARY_PATH")
	}
	if !strings.Contains(content, "PHPV_EXTRA_LD_LIBRARY_PATH") {
		t.Error("shim should support PHPV_EXTRA_LD_LIBRARY_PATH allowlist")
	}
}

func TestRenderShim_HasCleanRoomBlock(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, "LD_LIBRARY_PATH=\"$PHPV_PREFIX/lib\"") {
		t.Error("shim should start LD_LIBRARY_PATH with only PHP prefix")
	}
	if !strings.Contains(content, "export LD_LIBRARY_PATH") {
		t.Error("shim should export LD_LIBRARY_PATH")
	}
}

func TestRenderShim_SystemModeUnaffected(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "php", Kind: kindBinary})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	// The system block should exec the system binary without touching LD_LIBRARY_PATH.
	if !strings.Contains(content, `exec "$PHP_PATH" "$@"`) {
		t.Error("system block should exec system binary as-is")
	}
}

func TestRenderShim_Phar(t *testing.T) {
	s := &Service{}
	content, err := s.renderShim(shimDef{Name: "composer", Kind: kindPhar, PharRel: "phar/composer.phar"})
	if err != nil {
		t.Fatalf("renderShim: %v", err)
	}
	if !strings.Contains(content, `exec "$PHPV_PREFIX/bin/php" "$PHPV_PREFIX/phar/composer.phar" "$@"`) {
		t.Error("phar shim should exec php with phar path")
	}
	if !strings.Contains(content, "command -v php") {
		t.Error("phar shim should have system block with command -v")
	}
}

func TestWriteShim_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	if err := s.WriteShim(shimDef{Name: "php", Kind: kindBinary}); err != nil {
		t.Fatalf("WriteShim: %v", err)
	}
	path := filepath.Join(dir, "php")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat shim: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("shim should be executable")
	}
}

func TestWriteAll_CreatesAllShims(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	if err := s.WriteAll(); err != nil {
		t.Fatalf("WriteAll: %v", err)
	}
	expected := []string{"php", "phpize", "php-config", "php-cgi", "phpdbg"}
	for _, name := range expected {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("shim %s not created", name)
		}
	}
}

func TestRegenerateAll_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	oldPath := filepath.Join(dir, "php")
	if err := os.WriteFile(oldPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.RegenerateAll(); err != nil {
		t.Fatalf("RegenerateAll: %v", err)
	}
	data, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "old content") {
		t.Error("shim was not overwritten")
	}
}

func TestWritePhar(t *testing.T) {
	dir := t.TempDir()
	s := &Service{bin: dir}
	if err := s.WritePhar("composer", "phar/composer.phar"); err != nil {
		t.Fatalf("WritePhar: %v", err)
	}
	path := filepath.Join(dir, "composer")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("phar shim not created")
	}
}
