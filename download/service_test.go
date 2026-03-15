package download

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
)

func TestGetSourcesDir(t *testing.T) {
	svc := NewService()

	// Test with default (no PHPV_ROOT set)
	viper.Reset()
	viper.AutomaticEnv()
	dir := svc.GetSourcesDir()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".phpv", "sources")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}

	// Test with custom PHPV_ROOT
	viper.Set("PHPV_ROOT", "/custom/path")
	dir = svc.GetSourcesDir()
	expected = filepath.Join("/custom/path", "sources")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}

	viper.Reset()
}

func TestGetDownloadSource(t *testing.T) {
	svc := NewService()

	tests := []struct {
		name         string
		phpSource    string
		expectedType domain.SourceType
	}{
		{"Default GitHub", "", domain.SourceTypeGitHub},
		{"Explicit GitHub", "github", domain.SourceTypeGitHub},
		{"Official", "official", domain.SourceTypeOfficial},
		{"PHP.net", "php.net", domain.SourceTypeOfficial},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			viper.AutomaticEnv()
			if tt.phpSource != "" {
				viper.Set("PHP_SOURCE", tt.phpSource)
			}

			source := svc.GetDownloadSource()
			if source.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, source.Type)
			}

			viper.Reset()
		})
	}
}

func TestBuildDownloadURL(t *testing.T) {
	svc := NewService()

	tests := []struct {
		name        string
		phpSource   string
		version     domain.Version
		expectedURL string
	}{
		{
			name:        "GitHub source",
			phpSource:   "github",
			version:     domain.Version{Major: 8, Minor: 3, Patch: 14},
			expectedURL: "https://github.com/php/php-src/archive/refs/tags/php-8.3.14.tar.gz",
		},
		{
			name:        "Official source",
			phpSource:   "official",
			version:     domain.Version{Major: 8, Minor: 3, Patch: 14},
			expectedURL: "https://www.php.net/distributions/php-8.3.14.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			viper.AutomaticEnv()
			viper.Set("PHP_SOURCE", tt.phpSource)

			url := svc.BuildDownloadURL(tt.version)
			if url != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, url)
			}

			viper.Reset()
		})
	}
}

func TestFindMatchingVersion(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	versions := []domain.Version{
		{Major: 8, Minor: 4, Patch: 14},
		{Major: 8, Minor: 4, Patch: 13},
		{Major: 8, Minor: 3, Patch: 27},
		{Major: 8, Minor: 3, Patch: 26},
		{Major: 7, Minor: 4, Patch: 33},
	}

	tests := []struct {
		name          string
		major         int
		minor         *int
		patch         *int
		expectedPatch int
		shouldError   bool
	}{
		{
			name:          "Match major only",
			major:         8,
			minor:         nil,
			patch:         nil,
			expectedPatch: 14,
			shouldError:   false,
		},
		{
			name:          "Match major.minor",
			major:         8,
			minor:         intPtr(3),
			patch:         nil,
			expectedPatch: 27,
			shouldError:   false,
		},
		{
			name:          "Match specific version",
			major:         8,
			minor:         intPtr(4),
			patch:         intPtr(13),
			expectedPatch: 13,
			shouldError:   false,
		},
		{
			name:        "No match",
			major:       9,
			minor:       nil,
			patch:       nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := svc.FindMatchingVersion(ctx, versions, tt.major, tt.minor, tt.patch)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if version.Patch != tt.expectedPatch {
					t.Errorf("Expected patch %d, got %d", tt.expectedPatch, version.Patch)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestGetCacheDir(t *testing.T) {
	svc := NewService()

	// Test with default (no PHPV_ROOT set)
	viper.Reset()
	viper.AutomaticEnv()
	dir := svc.GetCacheDir()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".phpv", "cache", "sources")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}

	// Test with custom PHPV_ROOT
	viper.Set("PHPV_ROOT", "/custom/path")
	dir = svc.GetCacheDir()
	expected = filepath.Join("/custom/path", "cache", "sources")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}

	viper.Reset()
}

func TestGetCachedArchivePath(t *testing.T) {
	svc := NewService()
	viper.Set("PHPV_ROOT", "/test/phpv")

	tests := []struct {
		name         string
		version      domain.Version
		phpSource    string
		expectedFile string
	}{
		{
			name:         "GitHub source",
			version:      domain.Version{Major: 8, Minor: 3, Patch: 14},
			phpSource:    "github",
			expectedFile: "php-8.3.14-github.tar.gz",
		},
		{
			name:         "Official source",
			version:      domain.Version{Major: 8, Minor: 4, Patch: 1},
			phpSource:    "official",
			expectedFile: "php-8.4.1.tar.gz",
		},
		{
			name:         "Default (GitHub)",
			version:      domain.Version{Major: 7, Minor: 4, Patch: 33},
			phpSource:    "",
			expectedFile: "php-7.4.33-github.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("PHP_SOURCE", tt.phpSource)
			cachePath := svc.getCachedArchivePath(tt.version)
			expected := filepath.Join("/test/phpv", "cache", "sources", tt.expectedFile)
			if cachePath != expected {
				t.Errorf("Expected %s, got %s", expected, cachePath)
			}
		})
	}

	viper.Reset()
}
