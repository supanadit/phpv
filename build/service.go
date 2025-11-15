package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/dependency"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/util"
)

type Service struct {
	depService *dependency.Service
}

func NewService() *Service {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, _ := os.UserHomeDir()
		root = filepath.Join(homeDir, ".phpv")
	}

	return &Service{
		depService: dependency.NewService(root),
	}
}

// GetVersionsDir returns the versions directory path where compiled PHP binaries are installed
func (s *Service) GetVersionsDir() string {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, _ := os.UserHomeDir()
		root = filepath.Join(homeDir, ".phpv")
	}
	return filepath.Join(root, "versions")
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

// CheckClang verifies that clang is installed and available
func (s *Service) CheckClang() error {
	cmd := exec.Command("clang", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clang is not installed or not in PATH. Please install clang first")
	}
	return nil
}

// Build compiles PHP from source using Clang and installs it to ~/.phpv/versions
func (s *Service) Build(ctx context.Context, version domain.Version) error {
	// Check if clang is available
	if err := s.CheckClang(); err != nil {
		return err
	}

	versionStr := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)
	sourcesDir := s.GetSourcesDir()
	sourceDir := filepath.Join(sourcesDir, versionStr)

	// Check if source exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s source not found at %s. Please download it first using 'phpv download %s'", versionStr, sourceDir, versionStr)
	}

	versionsDir := s.GetVersionsDir()
	installDir := filepath.Join(versionsDir, versionStr)

	// Check if already built
	phpBinary := filepath.Join(installDir, "bin", "php")
	if _, err := os.Stat(phpBinary); err == nil {
		return fmt.Errorf("PHP %s is already built at %s", versionStr, installDir)
	}

	// Create versions directory
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	fmt.Printf("Building PHP %s from source using Clang...\n", versionStr)
	fmt.Printf("Source: %s\n", sourceDir)
	fmt.Printf("Target: %s\n", installDir)
	fmt.Println()

	// Step 1: Build dependencies
	fmt.Println("Step 1: Building dependencies...")
	if err := s.depService.BuildDependencies(ctx, version); err != nil {
		return fmt.Errorf("failed to build dependencies: %w", err)
	}

	// Step 2: Run buildconf (if it exists)
	buildconfPath := filepath.Join(sourceDir, "buildconf")
	if _, err := os.Stat(buildconfPath); err == nil {
		fmt.Println("Step 2: Running buildconf...")
		if err := util.RunCommand(ctx, sourceDir, nil, "./buildconf", "--force"); err != nil {
			return fmt.Errorf("buildconf failed: %w", err)
		}
	}

	// Step 3: Configure with dependency paths
	fmt.Println("Step 3: Configuring PHP build...")

	// Base configure arguments
	configureArgs := []string{
		fmt.Sprintf("--prefix=%s", installDir),
		"--enable-static",
		"--disable-shared",
		"--disable-all",
		"--enable-cli",
		"--enable-phar",
		"--enable-mbstring",
		"--enable-ctype",
		"--enable-tokenizer",
		"--enable-filter",
		"--enable-dom",
		"--enable-xml",
		"--enable-simplexml",
		"--enable-xmlreader",
		"--enable-xmlwriter",
		"--enable-fileinfo",
		"--enable-session",
		"--enable-pcntl",
		"--enable-posix",
	}

	// Add version-specific flags
	if version.Major == 7 {
		// PHP 7.x specific flags
		configureArgs = append(configureArgs, "--enable-hash")
		configureArgs = append(configureArgs, "--enable-json")
	}

	// Add dependency-specific configure flags
	depFlags := s.depService.GetPHPConfigureFlags(version)
	configureArgs = append(configureArgs, depFlags...)

	// Get environment with dependency paths
	env := s.depService.GetPHPEnvironment(version)

	if err := util.RunCommand(ctx, sourceDir, env, "./configure", configureArgs...); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}

	// Step 4: Make
	fmt.Println("Step 4: Compiling PHP (this may take a while)...")
	if err := util.RunCommand(ctx, sourceDir, env, "make", "-j4"); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	// Step 5: Make install
	fmt.Println("Step 5: Installing PHP...")
	if err := util.RunCommand(ctx, sourceDir, env, "make", "install"); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	// Step 6: Verify installation
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		return fmt.Errorf("PHP binary not found at %s after installation", phpBinary)
	}

	// Step 7: Test the binary
	fmt.Println("Step 6: Testing PHP binary...")
	if err := util.RunCommand(ctx, sourceDir, nil, phpBinary, "--version"); err != nil {
		return fmt.Errorf("PHP binary test failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Successfully built and installed PHP %s to %s\n", versionStr, installDir)
	fmt.Printf("  Binary: %s\n", phpBinary)
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
