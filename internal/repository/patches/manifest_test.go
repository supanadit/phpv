package patches

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifestPreservesPlaceholders(t *testing.T) {
	data := []byte(`extraCFlags:
  - "-Wno-error=incompatible-pointer-types"
  - "-Wno-incompatible-pointer-types"
configureFlags:
  - "-DZLIB_ROOT={{dep:zlib}}"
  - "--with-openssl={{dep:openssl}}"
  - "--prefix={{prefix}}"
  - "{{source}}"
`)
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	wantCFlags := []string{"-Wno-error=incompatible-pointer-types", "-Wno-incompatible-pointer-types"}
	if len(m.ExtraCFlags) != len(wantCFlags) {
		t.Fatalf("ExtraCFlags = %v, want %v", m.ExtraCFlags, wantCFlags)
	}
	for i, w := range wantCFlags {
		if m.ExtraCFlags[i] != w {
			t.Errorf("ExtraCFlags[%d] = %q, want %q", i, m.ExtraCFlags[i], w)
		}
	}
	wantConfigure := []string{
		"-DZLIB_ROOT={{dep:zlib}}",
		"--with-openssl={{dep:openssl}}",
		"--prefix={{prefix}}",
		"{{source}}",
	}
	if len(m.ConfigureFlags) != len(wantConfigure) {
		t.Fatalf("ConfigureFlags = %v, want %v", m.ConfigureFlags, wantConfigure)
	}
	for i, w := range wantConfigure {
		if m.ConfigureFlags[i] != w {
			t.Errorf("ConfigureFlags[%d] = %q, want %q", i, m.ConfigureFlags[i], w)
		}
	}
}

func TestParseManifestEmpty(t *testing.T) {
	m, err := ParseManifest([]byte(""))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if m == nil || len(m.ExtraCFlags) != 0 || len(m.ConfigureFlags) != 0 {
		t.Fatalf("expected empty manifest, got %+v", m)
	}
}

func TestParseManifestInvalidYAML(t *testing.T) {
	if _, err := ParseManifest([]byte("extraCFlags: [unclosed")); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadManifestMissingIsNil(t *testing.T) {
	dir := t.TempDir()
	m, err := LoadManifest(filepath.Join(dir, ManifestFileName))
	if err != nil {
		t.Fatalf("LoadManifest on missing file: %v", err)
	}
	if m != nil {
		t.Fatalf("expected nil manifest, got %+v", m)
	}
}

func TestLoadManifestFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(path, []byte("configureFlags:\n  - \"-DZLIB_ROOT={{dep:zlib}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.ConfigureFlags) != 1 || m.ConfigureFlags[0] != "-DZLIB_ROOT={{dep:zlib}}" {
		t.Fatalf("ConfigureFlags = %v", m.ConfigureFlags)
	}
}
