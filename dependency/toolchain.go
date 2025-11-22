package dependency

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// ToolchainService manages downloading and installing LLVM toolchains
type ToolchainService struct {
	httpClient *http.Client
	phpvRoot   string
}

func NewToolchainService(phpvRoot string) *ToolchainService {
	return &ToolchainService{
		httpClient: &http.Client{},
		phpvRoot:   phpvRoot,
	}
}

// GetToolchainDir returns the toolchains directory
func (s *ToolchainService) GetToolchainDir() string {
	return filepath.Join(s.phpvRoot, "toolchains")
}

// GetLLVMInstallDir returns the install directory for a specific LLVM version
func (s *ToolchainService) GetLLVMInstallDir(llvmVersion string) string {
	return filepath.Join(s.GetToolchainDir(), "llvm-"+llvmVersion)
}

// GetCacheDir returns the cache directory for downloaded toolchain archives
func (s *ToolchainService) GetCacheDir() string {
	return filepath.Join(s.phpvRoot, "cache", "toolchains")
}

// IsLLVMInstalled checks if a specific LLVM version is already installed
func (s *ToolchainService) IsLLVMInstalled(llvmVersion string) bool {
	installDir := s.GetLLVMInstallDir(llvmVersion)
	clangPath := filepath.Join(installDir, "bin", "clang")
	if stat, err := os.Stat(clangPath); err == nil && !stat.IsDir() {
		return true
	}
	return false
}

// DownloadAndInstallLLVM downloads and installs LLVM if not already present
func (s *ToolchainService) DownloadAndInstallLLVM(ctx context.Context, phpVersion domain.Version) error {
	llvmConfig := domain.GetLLVMVersionForPHP(phpVersion)

	if s.IsLLVMInstalled(llvmConfig.Version) {
		fmt.Printf("→ LLVM %s already installed, skipping\n", llvmConfig.Version)
		return nil
	}

	fmt.Printf("\n=== Installing LLVM %s ===\n", llvmConfig.Version)

	cachePath := s.getCachedArchivePath(llvmConfig.DownloadURL)

	// Check if already cached
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		// Download to cache
		fmt.Printf("Downloading LLVM %s from %s...\n", llvmConfig.Version, llvmConfig.DownloadURL)
		if err := s.downloadToCache(ctx, llvmConfig.DownloadURL, cachePath); err != nil {
			return fmt.Errorf("failed to download LLVM: %w", err)
		}
		fmt.Printf("Downloaded and cached: %s\n", filepath.Base(cachePath))
	} else {
		fmt.Printf("Using cached archive: %s\n", filepath.Base(cachePath))
	}

	// Extract to toolchains directory
	installDir := s.GetLLVMInstallDir(llvmConfig.Version)
	fmt.Printf("Extracting LLVM to %s...\n", installDir)

	if err := s.extractLLVM(cachePath, installDir); err != nil {
		return fmt.Errorf("failed to extract LLVM: %w", err)
	}

	// Verify installation
	clangPath := filepath.Join(installDir, "bin", "clang")
	if _, err := os.Stat(clangPath); os.IsNotExist(err) {
		return fmt.Errorf("LLVM installation verification failed: clang not found at %s", clangPath)
	}

	fmt.Printf("✓ LLVM %s installed successfully\n", llvmConfig.Version)
	return nil
}

// GetToolchainConfig returns the toolchain configuration for a PHP version
func (s *ToolchainService) GetToolchainConfig(phpVersion domain.Version) *domain.ToolchainConfig {
	llvmConfig := domain.GetLLVMVersionForPHP(phpVersion)

	if !s.IsLLVMInstalled(llvmConfig.Version) {
		// Return nil if not installed yet
		return nil
	}

	installDir := s.GetLLVMInstallDir(llvmConfig.Version)
	binDir := filepath.Join(installDir, "bin")

	return &domain.ToolchainConfig{
		CC:   filepath.Join(binDir, "clang"),
		CXX:  filepath.Join(binDir, "clang++"),
		Path: []string{binDir},
		// Add any additional flags needed for the toolchain
		CFlags:   []string{},
		CPPFlags: []string{},
		LDFlags:  []string{},
	}
}

// getCachedArchivePath returns the path where an LLVM archive should be cached
func (s *ToolchainService) getCachedArchivePath(url string) string {
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	return filepath.Join(s.GetCacheDir(), filename)
}

// downloadToCache downloads a file to the cache directory
func (s *ToolchainService) downloadToCache(ctx context.Context, url, cachePath string) error {
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

	// Write to temporary file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	// Move to final cache location
	if err := os.Rename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("failed to move to cache: %w", err)
	}

	return nil
}

// extractLLVM extracts the LLVM tar.xz archive
func (s *ToolchainService) extractLLVM(cachePath, installDir string) error {
	// Create install directory
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Open the cached file
	file, err := os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cached file: %w", err)
	}
	defer file.Close()

	// Use xz command to decompress to temporary file
	tmpFile, err := os.CreateTemp("", "llvm-*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temp tar file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	// Decompress using xz
	xzCmd := exec.Command("xz", "-dc", cachePath)
	tarFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	xzCmd.Stdout = tarFile

	if err := xzCmd.Run(); err != nil {
		tarFile.Close()
		return fmt.Errorf("failed to decompress with xz: %w", err)
	}
	tarFile.Close()

	// Extract tar
	tarFileReader, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open decompressed tar: %w", err)
	}
	defer tarFileReader.Close()

	return s.extractTar(tar.NewReader(tarFileReader), installDir)
}

// extractTar extracts a tar archive, stripping the first directory component
func (s *ToolchainService) extractTar(tr *tar.Reader, destDir string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip the first directory component (e.g., "LLVM-21.1.6-Linux-X64/")
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}

		target := filepath.Join(destDir, parts[1])

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
		case tar.TypeSymlink:
			// Handle symlinks
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			// Remove existing file/link if it exists
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	return nil
}
