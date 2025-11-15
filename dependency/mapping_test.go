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
		{
			name:          "PHP 7.4",
			version:       domain.Version{Major: 7, Minor: 4, Patch: 33},
			expectedCount: 5,
			expectedFirst: "zlib",
		},
		{
			name:          "PHP 7.3",
			version:       domain.Version{Major: 7, Minor: 3, Patch: 33},
			expectedCount: 5,
			expectedFirst: "zlib",
		},
		{
			name:          "PHP 7.0",
			version:       domain.Version{Major: 7, Minor: 0, Patch: 33},
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

func TestPHP7DependencyVersions(t *testing.T) {
	version := domain.Version{Major: 7, Minor: 4, Patch: 33}
	deps := GetDependenciesForVersion(version)

	// Test that PHP 7.x gets appropriate dependency versions
	expectedVersions := map[string]string{
		"zlib":      "1.2.13", // Older stable version for PHP 7
		"libxml2":   "2.9.14", // libxml2 2.9.x for PHP 7
		"openssl":   "1.1.1w", // OpenSSL 1.1.1 series for PHP 7
		"curl":      "7.88.1", // curl 7.x for PHP 7
		"oniguruma": "6.9.8",  // Slightly older oniguruma
	}

	for _, dep := range deps {
		expectedVersion, exists := expectedVersions[dep.Name]
		if !exists {
			t.Errorf("unexpected dependency: %s", dep.Name)
			continue
		}
		if dep.Version != expectedVersion {
			t.Errorf("dependency %s: expected version %s, got %s", dep.Name, expectedVersion, dep.Version)
		}
	}
}

func TestPHP8DependencyVersions(t *testing.T) {
	version := domain.Version{Major: 8, Minor: 3, Patch: 27}
	deps := GetDependenciesForVersion(version)

	// Test that PHP 8.3+ gets newer dependency versions
	expectedVersions := map[string]string{
		"zlib":      "1.3.1",  // Newer zlib for PHP 8
		"libxml2":   "2.12.7", // libxml2 2.12.x for PHP 8.3+
		"openssl":   "3.3.2",  // OpenSSL 3.x for PHP 8
		"curl":      "8.10.1", // curl 8.x for PHP 8
		"oniguruma": "6.9.9",  // Latest oniguruma
	}

	for _, dep := range deps {
		expectedVersion, exists := expectedVersions[dep.Name]
		if !exists {
			t.Errorf("unexpected dependency: %s", dep.Name)
			continue
		}
		if dep.Version != expectedVersion {
			t.Errorf("dependency %s: expected version %s, got %s", dep.Name, expectedVersion, dep.Version)
		}
	}
}
