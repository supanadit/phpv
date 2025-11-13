package repository

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	// Construct the download URL for PHP source
	// Format: https://www.php.net/distributions/php-{version}.tar.gz
	url := fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", version.Version)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 300000000000, // 5 minutes timeout
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Create temporary file for download
	tempFile, err := os.CreateTemp("", "php-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file
	defer tempFile.Close()

	// Download the file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Close the temp file for reading
	tempFile.Close()

	// Extract the tar.gz file
	if err := d.extractTarGz(tempFile.Name(), destPath); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	return nil
}

// extractTarGz extracts a tar.gz file to the specified destination
func (d *HTTPDownloader) extractTarGz(archivePath, destPath string) error {
	// Open the tar.gz file
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Extract files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct the full path for the extracted file
		// Remove the top-level directory from the path (php-X.Y.Z/)
		parts := strings.Split(header.Name, "/")
		if len(parts) > 1 {
			// Skip the first part (top-level directory)
			targetPath := filepath.Join(destPath, filepath.Join(parts[1:]...))
			if targetPath == destPath {
				continue // Skip if it's just the destination directory
			}

			// Create directory if needed
			if header.Typeflag == tar.TypeDir {
				if err := os.MkdirAll(targetPath, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
				}
				continue
			}

			// Create parent directory
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}

			// Extract file
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			file.Close()
		}
	}

	return nil
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

	// Extract version from source path (format: .../sources/{version}/)
	// This is a bit hacky but works for the current implementation
	version := "unknown"
	if parts := strings.Split(strings.TrimRight(sourcePath, "/"), "/"); len(parts) > 0 {
		version = parts[len(parts)-1]
	}

	// Create a placeholder binary to simulate successful build
	binPath := filepath.Join(installPath, "bin", "php")
	binDir := filepath.Dir(binPath)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	content := fmt.Sprintf("#!/bin/bash\necho 'PHP %s (simulated)'\n", version)
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
