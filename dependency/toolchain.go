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

	// Create wrapper scripts for PATH persistence
	if err := s.CreateZigWrappers(); err != nil {
		fmt.Printf("⚠ Warning: Failed to create Zig wrappers: %v\n", err)
		// Non-fatal, continue anyway
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
	// Zig binary is directly in installDir, not in bin/ subdirectory
	binDir := installDir

	// Ensure lightweight wrappers exist to satisfy CMake when using Zig
	// These wrappers point to the Zig binary with the appropriate sub-command,
	// allowing CMake to invoke a compiler wrapper like zcc/zcpp.
	_ = ensureZigWrappers(binDir, installDir)

	// Add system library paths for Zig dynamic linking
	systemLibPaths := domain.GetSystemLibPaths()
	var ldFlags []string
	ldFlags = append(ldFlags, "-target x86_64-linux-gnu")
	// Use --undefined-version to allow version scripts to reference missing symbols
	// (common in libxml2 when features like FTP/HTTP are disabled)
	ldFlags = append(ldFlags, "-Wl,--undefined-version")
	// Use compiler-rt to resolve built-in symbols (like __extenddftf2)
	ldFlags = append(ldFlags, "-rtlib=compiler-rt")
	// Force static linking of compiler runtime to avoid symbol lookup errors in the resulting binary
	ldFlags = append(ldFlags, "-static-libgcc")
	// Allow undefined symbols in shared libraries (needed for some complex dependency chains)
	ldFlags = append(ldFlags, "-Wl,--allow-shlib-undefined")

	for _, libPath := range systemLibPaths {
		ldFlags = append(ldFlags, "-L"+libPath)
		// Add rpath for runtime library discovery
		if libPath != "/usr/lib" && libPath != "/usr/lib64" {
			// Only add rpath for non-standard paths; /usr/lib is handled by ld.so.conf
			ldFlags = append(ldFlags, "-Wl,-rpath,"+libPath)
		}
	}

	// Add system include paths for Zig
	systemIncludePaths := domain.GetSystemIncludePaths()
	var cFlags []string
	cFlags = append(cFlags, "-target x86_64-linux-gnu")
	for _, incPath := range systemIncludePaths {
		cFlags = append(cFlags, "-I"+incPath)
	}

	return &domain.ToolchainConfig{
		CC:   filepath.Join(binDir, "zcc"),
		CXX:  filepath.Join(binDir, "zcpp"),
		Path: []string{binDir},
		CFlags: cFlags,
		CPPFlags: cFlags,
		LDFlags: ldFlags,
	}
}

// ensureZigWrappers ensures small wrapper scripts exist to adapt Zig's
// multi-part launcher for use as C/C++ compilers via CMake.
// It creates zcc and zcpp wrappers in the Zig install directory that call
// zig cc/c++ with a fixed target, delegating all further flags/args.
func ensureZigWrappers(binDir, zigInstallDir string) error {
	zigPath := filepath.Join(zigInstallDir, "zig")
	// wrappers live next to the Zig install dir
	zcc := filepath.Join(zigInstallDir, "zcc")
	zcpp := filepath.Join(zigInstallDir, "zcpp")

	// Content for wrappers
	// We add -rtlib=compiler-rt -static-libgcc to ensure compiler runtime symbols are included
	// We add -fno-sanitize=undefined and -fno-sanitize-trap=undefined because PHP uses some pointer arithmetic that Zig's default UBSan considers invalid
	// We add -D_GNU_SOURCE and -D_DEFAULT_SOURCE to ensure consistent glibc feature selection
	// We use -fuse-ld=/usr/bin/ld to force using host linker instead of Zig's internal LLD
	commonFlags := "-target x86_64-linux-gnu -Wl,--undefined-version -rtlib=compiler-rt -static-libgcc -fno-sanitize=undefined -fno-sanitize-trap=undefined -fno-stack-check -D_GNU_SOURCE -D_DEFAULT_SOURCE -fuse-ld=/usr/bin/ld"
	contentZcc := "#!/bin/sh\nexec \"" + zigPath + "\" cc " + commonFlags + " \"$@\"\n"
	contentZcpp := "#!/bin/sh\nexec \"" + zigPath + "\" c++ " + commonFlags + " \"$@\"\n"

	// Write wrappers if not present or different
	if err := os.WriteFile(zcc, []byte(contentZcc), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(zcpp, []byte(contentZcpp), 0755); err != nil {
		return err
	}
	return nil
}

// CreateZigWrappers creates wrapper scripts for Zig tools in ~/.phpv/bin/
// This ensures Zig is accessible regardless of PATH manipulation during builds
func (s *ToolchainService) CreateZigWrappers() error {
	installDir := s.GetZigInstallDir()
	zigPath := filepath.Join(installDir, "zig")

	if _, err := os.Stat(zigPath); os.IsNotExist(err) {
		return fmt.Errorf("Zig not installed, cannot create wrappers")
	}

	// Get phpv bin directory (from ToolchainService)
	// We need to find the phpv root - it's the parent of toolchains directory
	phpvRoot := filepath.Dir(s.GetToolchainDir())
	binDir := filepath.Join(phpvRoot, "bin")

	// Create bin directory if it doesn't exist
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Create wrappers for Zig tools
	wrappers := []string{"zig", "zigcc", "zigc++"}
	// Map to actual commands
	commands := map[string]string{
		"zig":    "zig",
		"zigcc":  "zig cc -target x86_64-linux-gnu",
		"zigc++": "zig c++ -target x86_64-linux-gnu",
	}

	for _, wrapper := range wrappers {
		wrapperPath := filepath.Join(binDir, wrapper)
		cmd := commands[wrapper]

		// Create wrapper script
		content := fmt.Sprintf("#!/bin/bash\nexec %s \"$@\"\n", cmd)
		if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to create wrapper %s: %w", wrapperPath, err)
		}
	}

	fmt.Printf("✓ Created Zig wrapper scripts in %s\n", binDir)
	return nil
}
