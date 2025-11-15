package download

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

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
)

type Service struct {
	httpClient *http.Client
}

func NewService() *Service {
	return &Service{
		httpClient: &http.Client{},
	}
}

// GetSourcesDir returns the sources directory path
func (s *Service) GetSourcesDir() string {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, _ := os.UserHomeDir()
		root = filepath.Join(homeDir, ".phpv")
	}
	return filepath.Join(root, "sources")
}

// GetDownloadSource returns the download source configuration
func (s *Service) GetDownloadSource() domain.DownloadSource {
	phpSource := viper.GetString("PHP_SOURCE")

	switch strings.ToLower(phpSource) {
	case "official", "php.net":
		return domain.DownloadSource{
			Type: domain.SourceTypeOfficial,
			URL:  "https://www.php.net/distributions",
		}
	default:
		// Default to GitHub
		return domain.DownloadSource{
			Type: domain.SourceTypeGitHub,
			URL:  "https://github.com/php/php-src",
		}
	}
}

// BuildDownloadURL constructs the download URL based on source and version
func (s *Service) BuildDownloadURL(version domain.Version) string {
	source := s.GetDownloadSource()
	versionStr := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)

	switch source.Type {
	case domain.SourceTypeOfficial:
		return fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", versionStr)
	default:
		// GitHub
		return fmt.Sprintf("https://github.com/php/php-src/archive/refs/tags/php-%s.tar.gz", versionStr)
	}
}

// Download downloads and extracts the PHP source code
func (s *Service) Download(ctx context.Context, version domain.Version) error {
	sourcesDir := s.GetSourcesDir()

	// Create sources directory if it doesn't exist
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return fmt.Errorf("failed to create sources directory: %w", err)
	}

	versionStr := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)
	targetDir := filepath.Join(sourcesDir, versionStr)

	// Check if already downloaded
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("PHP %s is already downloaded at %s", versionStr, targetDir)
	}

	downloadURL := s.BuildDownloadURL(version)
	fmt.Printf("Downloading PHP %s from %s...\n", versionStr, downloadURL)

	// Download the file
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	// Extract the tar.gz file
	fmt.Printf("Extracting to %s...\n", targetDir)
	if err := s.extractTarGz(resp.Body, sourcesDir, versionStr); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	fmt.Printf("Successfully downloaded PHP %s to %s\n", versionStr, targetDir)
	return nil
}

// extractTarGz extracts a tar.gz archive
func (s *Service) extractTarGz(r io.Reader, destDir, versionStr string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip the first directory component and replace with version string
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}

		target := filepath.Join(destDir, versionStr, parts[1])

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

// FindMatchingVersion finds a matching version based on the input
func (s *Service) FindMatchingVersion(ctx context.Context, versions []domain.Version, major int, minor *int, patch *int) (domain.Version, error) {
	var matches []domain.Version

	for _, v := range versions {
		if v.Major != major {
			continue
		}
		if minor != nil && v.Minor != *minor {
			continue
		}
		if patch != nil && v.Patch != *patch {
			continue
		}
		matches = append(matches, v)
	}

	if len(matches) == 0 {
		if patch != nil {
			return domain.Version{}, fmt.Errorf("no version found matching %d.%d.%d", major, *minor, *patch)
		} else if minor != nil {
			return domain.Version{}, fmt.Errorf("no version found matching %d.%d", major, *minor)
		}
		return domain.Version{}, fmt.Errorf("no version found matching %d", major)
	}

	// Return the latest matching version (assuming versions are sorted descending)
	return matches[0], nil
}
