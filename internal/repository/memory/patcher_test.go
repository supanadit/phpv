package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/patcher"
)

// TestFindFile_DoesNotMatchDirectoryNameSubstring verifies that findFile
// returns the actual target file, not a path whose directory name happens
// to contain the target name.
func TestFindFile_DoesNotMatchDirectoryNameSubstring(t *testing.T) {
	tmpDir := t.TempDir()
	// Decoy directory whose name contains "configure".
	decoyDir := filepath.Join(tmpDir, "php-8.0.30", ".github", "actions", "configure-macos")
	if err := os.MkdirAll(decoyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decoyDir, "action.yml"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	// Actual target configure file.
	configurePath := filepath.Join(tmpDir, "php-8.0.30", "configure")
	if err := os.WriteFile(configurePath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := findFile(filepath.Join(tmpDir, "php-8.0.30"), "configure")
	if err != nil {
		t.Fatalf("findFile failed: %v", err)
	}
	if got != configurePath {
		t.Errorf("findFile = %q, want %q", got, configurePath)
	}
}

// TestPatchPhpIntlCxx17_AppliesToGeneratedConfigure verifies that the
// php-intl-cxx17 patch replaces the C++11 standard assignment with C++17.
func TestPatchPhpIntlCxx17_AppliesToGeneratedConfigure(t *testing.T) {
	srcDir := t.TempDir()
	configurePath := filepath.Join(srcDir, "configure")
	content := "before\n        eval PHP_INTL_STDCXX=\"$switch\"\nafter\n"
	if err := os.WriteFile(configurePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repo := NewPatcherRepository()
	svc := patcher.NewService(repo)
	prepared, err := svc.Prepare("php", "8.0.30", srcDir)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if len(prepared.Applied) == 0 || prepared.Applied[0] != "php-intl-cxx17" {
		t.Fatalf("expected php-intl-cxx17 to be applied, got %v", prepared.Applied)
	}

	data, err := os.ReadFile(configurePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "eval PHP_INTL_STDCXX=\"$switch\"") {
		t.Errorf("configure still contains original C++11 assignment")
	}
	if !strings.Contains(string(data), "PHP_INTL_STDCXX=\"-std=gnu++17\"") {
		t.Errorf("configure does not contain C++17 assignment: %s", string(data))
	}
}
