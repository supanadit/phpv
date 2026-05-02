package disk

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func TestBundlerRepository_BuildTools(t *testing.T) {
	expectedTools := map[string]bool{
		"m4":       true,
		"autoconf": true,
		"automake": true,
		"libtool":  true,
		"perl":     true,
		"bison":    true,
		"flex":     true,
		"re2c":     true,
		"zig":      true,
	}

	for tool := range expectedTools {
		if !utils.BuildTools[tool] {
			t.Errorf("Expected %s to be a build tool", tool)
		}
	}

	if utils.BuildTools["openssl"] {
		t.Error("openssl should not be a build tool")
	}

	if utils.BuildTools["libxml2"] {
		t.Error("libxml2 should not be a build tool")
	}
}

func TestBundlerRepository_IsBuildTool(t *testing.T) {
	buildToolTests := []struct {
		name     string
		expected bool
	}{
		{"m4", true},
		{"autoconf", true},
		{"automake", true},
		{"libtool", true},
		{"perl", true},
		{"bison", true},
		{"flex", true},
		{"re2c", true},
		{"zig", true},
		{"gcc", false},
		{"make", false},
		{"openssl", false},
		{"libxml2", false},
		{"curl", false},
	}

	for _, tc := range buildToolTests {
		t.Run(tc.name, func(t *testing.T) {
			if utils.BuildTools[tc.name] != tc.expected {
				t.Errorf("BuildTools[%s] = %v, expected %v", tc.name, utils.BuildTools[tc.name], tc.expected)
			}
		})
	}
}

func TestBundlerRepository_FreshClean_EmptyInputs(t *testing.T) {
	baseDir := t.TempDir()
	silo := &domain.Silo{Root: baseDir}
	fs := afero.NewOsFs()

	repo := &bundlerRepository{
		silo: silo,
		fs:   fs,
	}

	err := repo.freshClean("", "8.4.0", nil)
	if err != nil {
		t.Errorf("freshClean should not fail with empty package name: %v", err)
	}

	err = repo.freshClean("php", "", nil)
	if err != nil {
		t.Errorf("freshClean should not fail with empty version: %v", err)
	}
}
