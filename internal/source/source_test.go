package source

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
}

func TestFind_DirectBuildFile(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "configure"))
	if got := Find(dir, "php"); got != dir {
		t.Fatalf("Find = %q, want %q", got, dir)
	}
}

func TestFind_SingleNestedDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "openssl-1.1.1w")
	touch(t, filepath.Join(nested, "Configure"))
	if got := Find(dir, "openssl"); got != nested {
		t.Fatalf("Find = %q, want %q", got, nested)
	}
}

func TestFind_ICUNestedSource(t *testing.T) {
	dir := t.TempDir()
	icuSource := filepath.Join(dir, "icu", "source")
	touch(t, filepath.Join(icuSource, "configure"))
	if got := Find(dir, "icu"); got != icuSource {
		t.Fatalf("Find = %q, want %q", got, icuSource)
	}
}

func TestFind_PicksBuildableAmongSeveral(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "docs", "README.md"))
	want := filepath.Join(dir, "src", "CMakeLists.txt")
	touch(t, want)
	if got := Find(dir, "zlib"); got != filepath.Dir(want) {
		t.Fatalf("Find = %q, want %q", got, filepath.Dir(want))
	}
}

func TestFind_NotFound(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "a", "README.md"))
	if got := Find(dir, "nope"); got != "" {
		t.Fatalf("Find = %q, want empty", got)
	}
	if got := Find(filepath.Join(dir, "missing"), "nope"); got != "" {
		t.Fatalf("Find on missing dir = %q, want empty", got)
	}
}
