package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
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

	// Step 1: Run buildconf (if it exists)
	buildconfPath := filepath.Join(sourceDir, "buildconf")
	if _, err := os.Stat(buildconfPath); err == nil {
		fmt.Println("Running buildconf...")
		if err := s.runCommand(ctx, sourceDir, "./buildconf", "--force"); err != nil {
			return fmt.Errorf("buildconf failed: %w", err)
		}
	}

	// Step 2: Configure
	fmt.Println("Configuring PHP build...")
	configureArgs := []string{
		fmt.Sprintf("--prefix=%s", installDir),
		"--enable-static",
		"--disable-shared",
		"--disable-all",
		"--enable-cli",
		"--enable-phar",
		"--enable-json",
		"--enable-mbstring",
		"--enable-ctype",
		"--enable-tokenizer",
		"--with-openssl",
		"--with-zlib",
		"--enable-filter",
		"--enable-dom",
		"--enable-xml",
		"--enable-simplexml",
		"--enable-xmlreader",
		"--enable-xmlwriter",
		"--with-curl",
		"--enable-fileinfo",
		"--enable-session",
		"--enable-pcntl",
		"--enable-posix",
	}

	env := append(os.Environ(),
		"CC=clang",
		"CXX=clang++",
	)

	if err := s.runCommandWithEnv(ctx, sourceDir, env, "./configure", configureArgs...); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}

	// Step 3: Make
	fmt.Println("Compiling PHP (this may take a while)...")
	if err := s.runCommandWithEnv(ctx, sourceDir, env, "make", "-j4"); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	// Step 4: Make install
	fmt.Println("Installing PHP...")
	if err := s.runCommandWithEnv(ctx, sourceDir, env, "make", "install"); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	// Step 5: Verify installation
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		return fmt.Errorf("PHP binary not found at %s after installation", phpBinary)
	}

	// Step 6: Test the binary
	fmt.Println("Testing PHP binary...")
	if err := s.runCommand(ctx, sourceDir, phpBinary, "--version"); err != nil {
		return fmt.Errorf("PHP binary test failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Successfully built and installed PHP %s to %s\n", versionStr, installDir)
	fmt.Printf("  Binary: %s\n", phpBinary)
	return nil
}

// runCommand runs a command in the specified directory and streams output to stdout/stderr
func (s *Service) runCommand(ctx context.Context, dir string, name string, args ...string) error {
	return s.runCommandWithEnv(ctx, dir, nil, name, args...)
}

// runCommandWithEnv runs a command with custom environment variables
func (s *Service) runCommandWithEnv(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if env != nil {
		cmd.Env = env
	}

	fmt.Printf("→ Running: %s %s\n", name, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return err
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
