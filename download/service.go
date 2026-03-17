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
	"time"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	uiPkg "github.com/supanadit/phpv/internal/ui"
	"github.com/supanadit/phpv/internal/util"
)

type Service struct {
	httpClient *http.Client
}

func NewService() *Service {
	return &Service{
		httpClient: &http.Client{},
	}
}

// GetCacheDir returns the cache directory for downloaded PHP archives
func (s *Service) GetCacheDir() string {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, _ := os.UserHomeDir()
		root = filepath.Join(homeDir, ".phpv")
	}
	return filepath.Join(root, "cache", "sources")
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
	versionStr := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)

	// Always try php.net first for all versions
	// Older versions (5.3 and earlier, 4.x) will return 404 and fallback to museum
	return fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", versionStr)
}

// getCachedArchivePath returns the cache path for a PHP version archive
func (s *Service) getCachedArchivePath(version domain.Version) string {
	versionStr := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)

	// Always use standard php naming for cache
	return filepath.Join(s.GetCacheDir(), fmt.Sprintf("php-%s.tar.gz", versionStr))
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

	ui := uiPkg.GetUI()

	// Check if already downloaded
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("PHP %s is already downloaded at %s", versionStr, targetDir)
	}

	cachePath := s.getCachedArchivePath(version)

	// Check if already cached
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		// Try primary URL first (php.net), then fallback to museum.php.net for old versions
		downloadURL := s.BuildDownloadURL(version)
		ui.PrintInfo(fmt.Sprintf("Downloading PHP %s from %s...", versionStr, downloadURL))

		if err := s.downloadToCache(ctx, downloadURL, cachePath); err != nil {
			// Check if it's a 404 and we should try museum.php.net
			if strings.Contains(err.Error(), "HTTP 404") {
				museumURL := s.buildMuseumURL(version)
				ui.PrintInfo(fmt.Sprintf("php.net returned 404, trying museum.php.net..."))
				ui.PrintInfo(fmt.Sprintf("Downloading PHP %s from %s...", versionStr, museumURL))

				if museumErr := s.downloadToCache(ctx, museumURL, cachePath); museumErr != nil {
					return fmt.Errorf("failed to download (tried php.net and museum.php.net): php.net: %w, museum: %w", err, museumErr)
				}
			} else {
				return fmt.Errorf("failed to download: %w", err)
			}
		}
		ui.PrintSuccess(fmt.Sprintf("Downloaded and cached: %s", filepath.Base(cachePath)))
	} else {
		ui.PrintDim(fmt.Sprintf("Using cached archive: %s", filepath.Base(cachePath)))
	}

	// Extract the tar.gz file from cache
	ui.PrintInfo(fmt.Sprintf("Extracting to %s...", targetDir))
	if err := s.extractFromCache(cachePath, sourcesDir, versionStr); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully downloaded PHP %s to %s", versionStr, targetDir))
	return nil
}

// buildMuseumURL returns the museum.php.net URL for old PHP versions
func (s *Service) buildMuseumURL(version domain.Version) string {
	versionStr := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)

	if version.Major == 4 {
		return fmt.Sprintf("https://museum.php.net/php4/php-%s.tar.gz", versionStr)
	}
	return fmt.Sprintf("https://museum.php.net/php5/php-%s.tar.gz", versionStr)
}

// downloadToCache downloads a file to the cache directory
func (s *Service) downloadToCache(ctx context.Context, url, cachePath string) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Create temporary file first
	tmpFile, err := os.CreateTemp(filepath.Dir(cachePath), ".download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write to temporary file with progress bar
	if err := util.DownloadWithProgress(resp, tmpFile, filepath.Base(cachePath)); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download: %w", err)
	}
	tmpFile.Close()

	// Move to final cache location
	if err := os.Rename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("failed to move to cache: %w", err)
	}

	return nil
}

// extractFromCache extracts a tar.gz archive from cache
func (s *Service) extractFromCache(cachePath, destDir, versionStr string) error {
	file, err := os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cached file: %w", err)
	}
	defer file.Close()

	return s.extractTarGz(file, destDir, versionStr)
}

// extractTarGz extracts a tar.gz archive
func (s *Service) extractTarGz(r io.Reader, destDir, versionStr string) error {
	ui := uiPkg.GetUI()

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

		// Strip the first directory component (e.g., php-5.3.29) and use version string as directory
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}

		// Use versionStr (e.g., "5.3.29") as the directory name
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

	// For PHP 4.x, touch pre-generated scanner/parser files to ensure they have
	// newer timestamps than the .l/.y source files. This prevents make from
	// trying to regenerate them using flex/bison, which fails because PHP 4's
	// bundled flex.skl skeleton is incompatible with modern flex versions.
	// Also touch for PHP 5.2 and earlier since old flex/bison don't build with modern compilers
	if strings.HasPrefix(versionStr, "4.") || (strings.HasPrefix(versionStr, "5.") && versionStr <= "5.2.99") {
		if err := s.touchPregeneratedFiles(destDir, versionStr); err != nil {
			// Log warning but don't fail - extraction succeeded
			ui.PrintWarning(fmt.Sprintf("Failed to touch pre-generated files: %v", err))
		}
	}

	return nil
}

// touchPregeneratedFiles updates timestamps on pre-generated scanner/parser files
// for PHP 4.x to prevent make from trying to regenerate them.
func (s *Service) touchPregeneratedFiles(destDir, versionStr string) error {
	versionDir := filepath.Join(destDir, versionStr)

	// List of pre-generated files to touch (relative to source root)
	filesToTouch := []string{
		// Zend scanner/parser files
		"Zend/zend_language_scanner.c",
		"Zend/zend_language_parser.c",
		"Zend/zend_language_parser.h",
		"Zend/zend_ini_scanner.c",
		"Zend/zend_ini_parser.c",
		"Zend/zend_ini_parser.h",
		// ext/standard parsedate (bison-generated)
		"ext/standard/parsedate.c",
	}

	now := time.Now()
	touched := 0

	for _, file := range filesToTouch {
		fullPath := filepath.Join(versionDir, file)
		if _, err := os.Stat(fullPath); err == nil {
			if err := os.Chtimes(fullPath, now, now); err != nil {
				return fmt.Errorf("failed to touch %s: %w", file, err)
			}
			touched++
		}
	}

	if touched > 0 {
		fmt.Printf("Touched %d pre-generated scanner/parser files for PHP %s\n", touched, versionStr)
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
