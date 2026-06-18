package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteShims_Success(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	expectedShims := []string{"php", "phpize", "php-config", "php-cgi", "composer", "pie", "wp"}

	for _, name := range expectedShims {
		shimPath := filepath.Join(binPath, name)
		if _, err := os.Stat(shimPath); os.IsNotExist(err) {
			t.Errorf("Shim %s was not created", name)
			continue
		}

		info, err := os.Stat(shimPath)
		if err != nil {
			t.Errorf("Failed to stat shim %s: %v", name, err)
			continue
		}

		if info.Mode()&0111 == 0 {
			t.Errorf("Shim %s is not executable", name)
		}
	}
}

func TestWriteShims_OverwritesExisting(t *testing.T) {
	binPath := t.TempDir()
	phpShimPath := filepath.Join(binPath, "php")

	if err := os.WriteFile(phpShimPath, []byte("old content"), 0644); err != nil {
		t.Fatalf("Failed to create existing shim: %v", err)
	}

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	content, err := os.ReadFile(phpShimPath)
	if err != nil {
		t.Fatalf("Failed to read shim: %v", err)
	}

	if strings.Contains(string(content), "old content") {
		t.Error("Shim was not overwritten")
	}
}

func TestShimContent_ContainsDynamicResolution(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	phpShimPath := filepath.Join(binPath, "php")
	content, err := os.ReadFile(phpShimPath)
	if err != nil {
		t.Fatalf("Failed to read php shim: %v", err)
	}

	shimContent := string(content)

	if !strings.Contains(shimContent, "#!/bin/bash") {
		t.Error("Shim should start with #!/bin/bash")
	}

	if !strings.Contains(shimContent, "PHPV_ROOT") {
		t.Error("Shim should reference PHPV_ROOT")
	}

	if !strings.Contains(shimContent, "PHPV_CURRENT") {
		t.Error("Shim should reference PHPV_CURRENT")
	}

	if !strings.Contains(shimContent, "LD_LIBRARY_PATH") {
		t.Error("Shim should set LD_LIBRARY_PATH")
	}
}

func TestShimContent_NoVersionSelected(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	phpShimPath := filepath.Join(binPath, "php")
	content, err := os.ReadFile(phpShimPath)
	if err != nil {
		t.Fatalf("Failed to read php shim: %v", err)
	}

	shimContent := string(content)

	if !strings.Contains(shimContent, "Error: No PHP version selected") {
		t.Error("Shim should contain error message for no version selected")
	}
}

func TestShimContent_VersionNotFound(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	phpShimPath := filepath.Join(binPath, "php")
	content, err := os.ReadFile(phpShimPath)
	if err != nil {
		t.Fatalf("Failed to read php shim: %v", err)
	}

	shimContent := string(content)

	if !strings.Contains(shimContent, "not found") {
		t.Error("Shim should contain error message for version not found")
	}
}

func TestShimContent_ExecWithArgs(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	phpShimPath := filepath.Join(binPath, "php")
	content, err := os.ReadFile(phpShimPath)
	if err != nil {
		t.Fatalf("Failed to read php shim: %v", err)
	}

	shimContent := string(content)

	if !strings.Contains(shimContent, "exec") {
		t.Error("Shim should contain exec command")
	}

	if !strings.Contains(shimContent, `"$@"`) {
		t.Error("Shim should pass arguments with \"$@\"")
	}
}

func TestPharShim_ContainsPharPath(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	for _, tool := range DefaultPharTools {
		shimPath := filepath.Join(binPath, tool.Name)
		content, err := os.ReadFile(shimPath)
		if err != nil {
			t.Errorf("Failed to read %s shim: %v", tool.Name, err)
			continue
		}

		shimContent := string(content)

		if !strings.Contains(shimContent, tool.PharFile) {
			t.Errorf("%s shim should reference %s", tool.Name, tool.PharFile)
		}

		if !strings.Contains(shimContent, "exec") {
			t.Errorf("%s shim should contain exec command", tool.Name)
		}

		if !strings.Contains(shimContent, `"$@"`) {
			t.Errorf("%s shim should pass arguments with \"$@\"", tool.Name)
		}
	}
}

func TestPharShim_AllToolsCreated(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	if len(DefaultPharTools) == 0 {
		t.Fatal("DefaultPharTools should not be empty")
	}

	for _, tool := range DefaultPharTools {
		shimPath := filepath.Join(binPath, tool.Name)
		info, err := os.Stat(shimPath)
		if err != nil {
			t.Errorf("Shim for %s was not created: %v", tool.Name, err)
			continue
		}

		if info.Mode()&0111 == 0 {
			t.Errorf("Shim %s is not executable", tool.Name)
		}
	}
}

func TestShimContent_DefaultLookup(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	phpShimPath := filepath.Join(binPath, "php")
	content, err := os.ReadFile(phpShimPath)
	if err != nil {
		t.Fatalf("Failed to read php shim: %v", err)
	}

	shimContent := string(content)

	if !strings.Contains(shimContent, "$PHPV_ROOT/default") {
		t.Error("Shim should look up default version from $PHPV_ROOT/default")
	}
}

func TestShimContent_EnvironmentFallback(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(ShimConfig{BinPath: binPath})
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	phpShimPath := filepath.Join(binPath, "php")
	content, err := os.ReadFile(phpShimPath)
	if err != nil {
		t.Fatalf("Failed to read php shim: %v", err)
	}

	shimContent := string(content)

	if !strings.Contains(shimContent, "${PHPV_ROOT:") {
		t.Error("Shim should have default fallback for PHPV_ROOT")
	}

	if !strings.Contains(shimContent, "$HOME/.phpv") {
		t.Error("Shim should default to $HOME/.phpv")
	}
}

func TestSystemMarker(t *testing.T) {
	root := t.TempDir()

	// Initially not system mode
	if IsSystemMode(root) {
		t.Error("Should not be in system mode without marker")
	}

	// Write marker
	if err := WriteSystemMarker(root); err != nil {
		t.Fatalf("WriteSystemMarker failed: %v", err)
	}

	if !IsSystemMode(root) {
		t.Error("Should be in system mode after writing marker")
	}

	// Remove marker
	if err := RemoveSystemMarker(root); err != nil {
		t.Fatalf("RemoveSystemMarker failed: %v", err)
	}

	if IsSystemMode(root) {
		t.Error("Should not be in system mode after removing marker")
	}

	// Removing again should be a no-op
	if err := RemoveSystemMarker(root); err != nil {
		t.Fatalf("RemoveSystemMarker on clean state failed: %v", err)
	}
}

func TestDefaultPharTools_NotEmpty(t *testing.T) {
	if len(DefaultPharTools) == 0 {
		t.Fatal("DefaultPharTools must have at least one tool")
	}

	for _, tool := range DefaultPharTools {
		if tool.Name == "" {
			t.Error("PharTool.Name must not be empty")
		}
		if tool.PharFile == "" {
			t.Errorf("PharTool %s has empty PharFile", tool.Name)
		}
		if tool.BinName == "" {
			t.Errorf("PharTool %s has empty BinName", tool.Name)
		}
	}
}

func TestDetectPharPath(t *testing.T) {
	for _, tool := range DefaultPharTools {
		path := DetectPharPath(tool, "/home/test/.phpv")
		if path == "" {
			t.Logf("Phar %s not found, skipping detection test", tool.Name)
			continue
		}

		if !filepath.IsAbs(path) {
			t.Errorf("DetectPharPath for %s returned non-absolute path: %s", tool.Name, path)
		}
	}
}
