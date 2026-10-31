package patches

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoEmbeddedPatchData pins the "zero baked-in patch data" rule: the
// binary ships no patches, so no go:embed directive may exist in this
// package (or anywhere it could pull patch content from) and patch
// content must only be reachable through injected filesystem roots.
func TestNoEmbeddedPatchData(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "go:embed") {
			t.Errorf("%s contains a go:embed directive; patches must never be embedded", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
