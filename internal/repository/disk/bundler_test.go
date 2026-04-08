package disk

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
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
		if !buildTools[tool] {
			t.Errorf("Expected %s to be a build tool", tool)
		}
	}

	if buildTools["openssl"] {
		t.Error("openssl should not be a build tool")
	}

	if buildTools["libxml2"] {
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
			if buildTools[tc.name] != tc.expected {
				t.Errorf("buildTools[%s] = %v, expected %v", tc.name, buildTools[tc.name], tc.expected)
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

	err := repo.freshClean("", "8.4.0")
	if err != nil {
		t.Errorf("freshClean should not fail with empty package name: %v", err)
	}

	err = repo.freshClean("php", "")
	if err != nil {
		t.Errorf("freshClean should not fail with empty version: %v", err)
	}
}
