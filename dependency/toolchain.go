package dependency

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/util"
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

// extractLLVM extracts the LLVM tar.xz archive
func (s *ToolchainService) extractLLVM(cachePath, installDir string) error {
	// Create install directory
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Use tar command directly to extract the tar.xz file
	// tar automatically handles xz decompression with -J or by detecting the format
	cmd := exec.Command("tar", "-xvf", cachePath, "-C", installDir, "--strip-components=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to extract LLVM archive: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetZigInstallDir returns the install directory for Zig
func (s *ToolchainService) GetZigInstallDir() string {
	return filepath.Join(s.GetToolchainDir(), "zig")
}

// IsZigInstalled checks if Zig is already installed
func (s *ToolchainService) IsZigInstalled() bool {
	installDir := s.GetZigInstallDir()
	zigPath := filepath.Join(installDir, "zig")
	if stat, err := os.Stat(zigPath); err == nil && !stat.IsDir() {
		return true
	}
	return false
}

// DownloadAndInstallZig downloads and installs Zig if not already present
func (s *ToolchainService) DownloadAndInstallZig(ctx context.Context) error {
	zigConfig := domain.GetZigVersion()

	if s.IsZigInstalled() {
		fmt.Printf("→ Zig %s already installed, skipping\n", zigConfig.Version)
		return nil
	}

	fmt.Printf("\n=== Installing Zig %s ===\n", zigConfig.Version)

	cachePath := s.getCachedArchivePath(zigConfig.DownloadURL)

	// Check if already cached
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		// Download to cache
		fmt.Printf("Downloading Zig %s from %s...\n", zigConfig.Version, zigConfig.DownloadURL)
		if err := s.downloadToCache(ctx, zigConfig.DownloadURL, cachePath); err != nil {
			return fmt.Errorf("failed to download Zig: %w", err)
		}
		fmt.Printf("Downloaded and cached: %s\n", filepath.Base(cachePath))
	} else {
		fmt.Printf("Using cached archive: %s\n", filepath.Base(cachePath))
	}

	// Extract to toolchains directory
	installDir := s.GetZigInstallDir()
	fmt.Printf("Extracting Zig to %s...\n", installDir)

	if err := s.extractZig(cachePath, installDir); err != nil {
		return fmt.Errorf("failed to extract Zig: %w", err)
	}

	// Verify installation
	zigPath := filepath.Join(installDir, "zig")
	if _, err := os.Stat(zigPath); os.IsNotExist(err) {
		return fmt.Errorf("Zig installation verification failed: zig not found at %s", zigPath)
	}

	fmt.Printf("✓ Zig %s installed successfully\n", zigConfig.Version)
	return nil
}

// extractZig extracts the Zig tar.xz archive
func (s *ToolchainService) extractZig(cachePath, installDir string) error {
	// Create install directory
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Use tar command directly to extract the tar.xz file
	cmd := exec.Command("tar", "-xvf", cachePath, "-C", installDir, "--strip-components=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to extract Zig archive: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetZigToolchainConfig returns the toolchain configuration for Zig
func (s *ToolchainService) GetZigToolchainConfig() *domain.ToolchainConfig {
	if !s.IsZigInstalled() {
		return nil
	}

	installDir := s.GetZigInstallDir()
	binDir := filepath.Join(installDir, "bin")

	return &domain.ToolchainConfig{
		CC:   "zig cc -target x86_64-linux-gnu",
		CXX:  "zig c++ -target x86_64-linux-gnu",
		Path: []string{binDir},
		CFlags: []string{
			"-target x86_64-linux-gnu",
		},
		CPPFlags: []string{
			"-target x86_64-linux-gnu",
		},
		LDFlags: []string{
			"-target x86_64-linux-gnu",
		},
	}
}
