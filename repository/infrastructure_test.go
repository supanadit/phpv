package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestHTTPDownloader_DownloadSource(t *testing.T) {
	downloader := NewHTTPDownloader()
	ctx := context.Background()

	// Test with a valid PHP version
	version := domain.PHPVersion{
		Version:     "8.1.0",
		Major:       8,
		Minor:       1,
		Patch:       0,
		ReleaseType: "stable",
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "phpv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test download (this will actually download from php.net)
	destPath := filepath.Join(tempDir, "source")
	err = downloader.DownloadSource(ctx, version, destPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Check if files were extracted
	entries, err := os.ReadDir(destPath)
	if err != nil {
		t.Fatalf("Failed to read destination directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No files were extracted")
	}

	// Check for some expected PHP source files
	expectedFiles := []string{"main", "Zend", "configure.ac"}
	foundFiles := make(map[string]bool)

	for _, entry := range entries {
		foundFiles[entry.Name()] = true
	}

	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("Expected file/directory %s not found in extracted source", expected)
		}
	}
}

func TestHTTPDownloader_DownloadSource_InvalidVersion(t *testing.T) {
	downloader := NewHTTPDownloader()
	ctx := context.Background()

	// Test with an invalid PHP version that doesn't exist
	version := domain.PHPVersion{
		Version:     "99.99.99",
		Major:       99,
		Minor:       99,
		Patch:       99,
		ReleaseType: "stable",
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "phpv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	destPath := filepath.Join(tempDir, "source")
	err = downloader.DownloadSource(ctx, version, destPath)
	if err == nil {
		t.Error("Expected download to fail for non-existent version")
	}
}
