package dependency

import (
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestGetDependenciesForVersion(t *testing.T) {
	tests := []struct {
		name          string
		version       domain.Version
		expectedCount int
		expectedFirst string
	}{
		{
			name:          "PHP 8.3",
			version:       domain.Version{Major: 8, Minor: 3, Patch: 27},
			expectedCount: 5,
			expectedFirst: "zlib",
		},
		{
			name:          "PHP 8.4",
			version:       domain.Version{Major: 8, Minor: 4, Patch: 14},
			expectedCount: 5,
			expectedFirst: "zlib",
		},
		{
			name:          "PHP 8.0",
			version:       domain.Version{Major: 8, Minor: 0, Patch: 30},
			expectedCount: 5,
			expectedFirst: "zlib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := GetDependenciesForVersion(tt.version)

			if len(deps) != tt.expectedCount {
				t.Errorf("expected %d dependencies, got %d", tt.expectedCount, len(deps))
			}

			if len(deps) > 0 && deps[0].Name != tt.expectedFirst {
				t.Errorf("expected first dependency to be %s, got %s", tt.expectedFirst, deps[0].Name)
			}

			// Verify all dependencies have required fields
			for _, dep := range deps {
				if dep.Name == "" {
					t.Error("dependency has empty name")
				}
				if dep.Version == "" {
					t.Error("dependency has empty version")
				}
				if dep.DownloadURL == "" {
					t.Error("dependency has empty download URL")
				}
			}
		})
	}
}

func TestDependencyVersions(t *testing.T) {
	version := domain.Version{Major: 8, Minor: 3, Patch: 27}
	deps := GetDependenciesForVersion(version)

	expectedDeps := map[string]bool{
		"zlib":      true,
		"libxml2":   true,
		"openssl":   true,
		"curl":      true,
		"oniguruma": true,
	}

	for _, dep := range deps {
		if !expectedDeps[dep.Name] {
			t.Errorf("unexpected dependency: %s", dep.Name)
		}
		delete(expectedDeps, dep.Name)
	}

	if len(expectedDeps) > 0 {
		for name := range expectedDeps {
			t.Errorf("missing expected dependency: %s", name)
		}
	}
}
