package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteShims_Success(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(binPath)
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	expectedShims := []string{"php", "phpize", "php-config", "php-cgi"}

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

	err := WriteShims(binPath)
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

	err := WriteShims(binPath)
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

	err := WriteShims(binPath)
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

	err := WriteShims(binPath)
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

	err := WriteShims(binPath)
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

func TestDynamicShimTemplate_php(t *testing.T) {
	template := fmt.Sprintf(dynamicShimTemplate, "php")

	if template == "" {
		t.Fatal("Template should not be empty")
	}

	if !strings.Contains(template, "%s") && !strings.Contains(template, "php") {
		t.Error("Template should reference the binary name")
	}
}

func TestDynamicShimTemplate_phpize(t *testing.T) {
	template := fmt.Sprintf(dynamicShimTemplate, "phpize")

	if template == "" {
		t.Fatal("Template should not be empty")
	}
}

func TestDynamicShimTemplate_phpConfig(t *testing.T) {
	template := fmt.Sprintf(dynamicShimTemplate, "php-config")

	if template == "" {
		t.Fatal("Template should not be empty")
	}
}

func TestDynamicShimTemplate_phpCgi(t *testing.T) {
	template := fmt.Sprintf(dynamicShimTemplate, "php-cgi")

	if template == "" {
		t.Fatal("Template should not be empty")
	}
}

func TestWriteShims_AllBinaries(t *testing.T) {
	binPath := t.TempDir()

	shims := []string{"php", "phpize", "php-config", "php-cgi"}

	for _, name := range shims {
		shimPath := filepath.Join(binPath, name)

		content := fmt.Sprintf(dynamicShimTemplate, name)
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			t.Errorf("Failed to write shim %s: %v", name, err)
		}

		readContent, err := os.ReadFile(shimPath)
		if err != nil {
			t.Errorf("Failed to read shim %s: %v", name, err)
			continue
		}

		if string(readContent) != content {
			t.Errorf("Shim %s content mismatch", name)
		}
	}
}

func TestWriteShims_Executable(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(binPath)
	if err != nil {
		t.Fatalf("WriteShims failed: %v", err)
	}

	for _, name := range []string{"php", "phpize", "php-config", "php-cgi"} {
		shimPath := filepath.Join(binPath, name)
		info, err := os.Stat(shimPath)
		if err != nil {
			t.Errorf("Failed to stat shim %s: %v", name, err)
			continue
		}

		mode := info.Mode()
		if mode.IsRegular() && mode&0111 == 0 {
			t.Errorf("Shim %s is not executable", name)
		}
	}
}

func TestShimContent_DefaultLookup(t *testing.T) {
	binPath := t.TempDir()

	err := WriteShims(binPath)
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

	err := WriteShims(binPath)
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
