package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
)

// HTTPDownloader implements Downloader using HTTP
type HTTPDownloader struct{}

// NewHTTPDownloader creates a new HTTP downloader
func NewHTTPDownloader() *HTTPDownloader {
	return &HTTPDownloader{}
}

// DownloadSource downloads PHP source code from the official repository
func (d *HTTPDownloader) DownloadSource(ctx context.Context, version domain.PHPVersion, destPath string) error {
	// For now, simulate download - in real implementation, this would use HTTP client
	// to download from https://www.php.net/distributions/php-{version}.tar.gz

	// Create a placeholder file to simulate download
	readmePath := filepath.Join(destPath, "README.md")
	content := fmt.Sprintf("# PHP %s Source Code\n\nThis is a placeholder for the actual PHP source download.\n", version.Version)

	return os.WriteFile(readmePath, []byte(content), 0644)
}

// SourceBuilder implements Builder for building PHP from source
type SourceBuilder struct{}

// NewSourceBuilder creates a new source builder
func NewSourceBuilder() *SourceBuilder {
	return &SourceBuilder{}
}

// Build builds PHP from source code
func (b *SourceBuilder) Build(ctx context.Context, sourcePath string, installPath string, config map[string]string) error {
	// For now, simulate build process - in real implementation, this would:
	// 1. Run ./configure with appropriate flags
	// 2. Run make
	// 3. Run make install

	// Create a placeholder binary to simulate successful build
	binPath := filepath.Join(installPath, "bin", "php")
	binDir := filepath.Dir(binPath)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	content := fmt.Sprintf("#!/bin/bash\necho 'PHP %s (simulated)'\n", "8.1.0")
	if err := os.WriteFile(binPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to create placeholder binary: %w", err)
	}

	return nil
}

// OSFileSystem implements FileSystem using OS operations
type OSFileSystem struct{}

// NewOSFileSystem creates a new OS filesystem
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// CreateDirectory creates a directory
func (fs *OSFileSystem) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// RemoveDirectory removes a directory recursively
func (fs *OSFileSystem) RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}

// FileExists checks if a file exists
func (fs *OSFileSystem) FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirectoryExists checks if a directory exists
func (fs *OSFileSystem) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
