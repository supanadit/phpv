package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/dependency"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/util"
)

type Service struct {
	depService *dependency.Service
	toolchain  *domain.ToolchainConfig
}

func NewService() *Service {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, _ := os.UserHomeDir()
		root = filepath.Join(homeDir, ".phpv")
	}

	// User-provided toolchain configuration takes precedence
	toolchain := loadToolchainConfig()

	depSvc := dependency.NewServiceWithToolchain(root, toolchain)

	return &Service{
		depService: depSvc,
		toolchain:  toolchain,
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

// CheckCompiler verifies that the selected compiler is available
func (s *Service) CheckCompiler() error {
	// If user provided custom toolchain, check that
	if s.toolchain != nil && s.toolchain.CC != "" {
		compiler := s.toolchain.CC
		parts := strings.Fields(compiler)
		if len(parts) > 0 && parts[0] != "" {
			compiler = parts[0]
		}
		cmd := exec.Command(compiler, "--version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s is not installed or not in PATH. Please install it or adjust PHPV_TOOLCHAIN_CC", compiler)
		}
	}
	// LLVM will be downloaded automatically during dependency build
	return nil
}

// Build compiles PHP from source using Clang and installs it to ~/.phpv/versions
func (s *Service) Build(ctx context.Context, version domain.Version) error {
	// Check if selected compiler is available
	if err := s.CheckCompiler(); err != nil {
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

	fmt.Printf("Building PHP %s from source using %s...\n", versionStr, s.compilerDisplayName(version))
	fmt.Printf("Source: %s\n", sourceDir)
	fmt.Printf("Target: %s\n", installDir)
	fmt.Println()

	// Step 1: Build dependencies (this will download and install LLVM)
	fmt.Println("Step 1: Building dependencies...")
	if err := s.depService.BuildDependencies(ctx, version); err != nil {
		return fmt.Errorf("failed to build dependencies: %w", err)
	}

	// Get environment with dependency paths (needed for buildconf and configure)
	env := s.depService.GetPHPEnvironment(version)

	// Step 2: Run buildconf (if it exists)
	buildconfPath := filepath.Join(sourceDir, "buildconf")
	if _, err := os.Stat(buildconfPath); err == nil {
		fmt.Println("Step 2: Running buildconf...")
		// Use per-version autoconf for PHP buildconf
		autoconfBin := filepath.Join(s.depService.GetDependencyInstallDir(version, "autoconf"), "bin")
		if _, err := os.Stat(filepath.Join(autoconfBin, "autoconf")); err == nil {
			env = s.addToPathEnv(env, autoconfBin)
			env = s.setEnvVar(env, "AUTOCONF", filepath.Join(autoconfBin, "autoconf"))
			env = s.setEnvVar(env, "AUTOHEADER", filepath.Join(autoconfBin, "autoheader"))
			env = s.setEnvVar(env, "AUTOMAKE", filepath.Join(autoconfBin, "automake"))
			env = s.setEnvVar(env, "ACLOCAL", filepath.Join(autoconfBin, "aclocal"))
		}
		if err := util.RunCommand(ctx, sourceDir, env, "./buildconf", "--force"); err != nil {
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
	switch version.Major {
	case 5:
		// PHP 5.x requires libxml to be explicitly enabled
		configureArgs = append(configureArgs, "--enable-libxml")
	case 7:
		// PHP 7.x specific flags
		// Note: PHP 7.4+ has libxml and hash built-in, only needed for 7.0-7.3
		if version.Minor < 4 {
			configureArgs = append(configureArgs, "--enable-libxml")
			configureArgs = append(configureArgs, "--enable-hash")
		}
		// JSON extension exists in all PHP 7.x versions
		configureArgs = append(configureArgs, "--enable-json")
	case 4:
		// PHP 4.x specific flags
		configureArgs = append(configureArgs, "--enable-libxml")
		configureArgs = append(configureArgs, "--with-regex=system")
		// PHP 4 uses different session handling
		configureArgs = append(configureArgs, "--enable-track-vars")
		// Enable standard extensions that exist in PHP 4
		configureArgs = append(configureArgs, "--enable-pcre")
		configureArgs = append(configureArgs, "--enable-ftp")
		configureArgs = append(configureArgs, "--enable-zlib")
		configureArgs = append(configureArgs, "--with-curl")
		// Note: OpenSSL is skipped because PHP 4's OpenSSL extension is
		// incompatible with modern OpenSSL versions
	}

	// Add dependency-specific configure flags
	depFlags := s.depService.GetPHPConfigureFlags(version)
	configureArgs = append(configureArgs, depFlags...)

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

func (s *Service) compilerDisplayName(version domain.Version) string {
	// Check if user provided a custom toolchain
	if s.toolchain != nil && s.toolchain.CC != "" {
		return s.toolchain.CC
	}

	// Check if we should use LLVM or system GCC
	if domain.ShouldUseLLVMToolchain(version) {
		// Get LLVM version for this PHP version
		llvmVersion := domain.GetLLVMVersionForPHP(version)
		return fmt.Sprintf("LLVM %s (clang)", llvmVersion.Version)
	}

	// Use system GCC
	return "System GCC (gcc)"
}

func (s *Service) getCompilerBinary(version domain.Version) string {
	// Check if user provided a custom toolchain
	if s.toolchain != nil && s.toolchain.CC != "" {
		cc := s.toolchain.CC
		parts := strings.Fields(cc)
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Check if we should use LLVM or system GCC
	if domain.ShouldUseLLVMToolchain(version) {
		// Use LLVM from dependencies
		llvmVersion := domain.GetLLVMVersionForPHP(version)
		root := viper.GetString("PHPV_ROOT")
		if root == "" {
			homeDir, _ := os.UserHomeDir()
			root = filepath.Join(homeDir, ".phpv")
		}
		llvmBin := filepath.Join(root, "toolchains", "llvm-"+llvmVersion.Version, "bin", "clang")
		return llvmBin
	}

	// Use system GCC
	return "gcc"
}

func loadToolchainConfig() *domain.ToolchainConfig {
	cfg := &domain.ToolchainConfig{}
	cfg.CC = getToolchainValue("PHPV_TOOLCHAIN_CC")
	cfg.CXX = getToolchainValue("PHPV_TOOLCHAIN_CXX")
	cfg.Sysroot = getToolchainValue("PHPV_TOOLCHAIN_SYSROOT")

	if pathVal := getToolchainValue("PHPV_TOOLCHAIN_PATH"); pathVal != "" {
		cfg.Path = parsePathList(pathVal)
	}
	if cflags := getToolchainValue("PHPV_TOOLCHAIN_CFLAGS"); cflags != "" {
		cfg.CFlags = parseFlagList(cflags)
	}
	if cppflags := getToolchainValue("PHPV_TOOLCHAIN_CPPFLAGS"); cppflags != "" {
		cfg.CPPFlags = parseFlagList(cppflags)
	}
	if ldflags := getToolchainValue("PHPV_TOOLCHAIN_LDFLAGS"); ldflags != "" {
		cfg.LDFlags = parseFlagList(ldflags)
	}

	if cfg.IsEmpty() {
		return nil
	}
	return cfg
}

func getToolchainValue(key string) string {
	val := strings.TrimSpace(viper.GetString(key))
	if val == "" {
		val = strings.TrimSpace(os.Getenv(key))
	}
	return val
}

func parsePathList(value string) []string {
	segments := strings.Split(value, string(os.PathListSeparator))
	var cleaned []string
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment != "" {
			cleaned = append(cleaned, segment)
		}
	}
	return cleaned
}

func parseFlagList(value string) []string {
	return strings.Fields(value)
}

func (s *Service) addToPathEnv(env []string, path string) []string {
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + path + ":" + strings.TrimPrefix(e, "PATH=")
			return env
		}
	}
	return append(env, "PATH="+path)
}

func (s *Service) setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// cleanSourceDir removes old build artifacts from the source directory
// This ensures a clean build when switching between different configurations
func (s *Service) cleanSourceDir(sourceDir string) error {
	// Try to run make clean first to remove build artifacts
	cleanMakefile := filepath.Join(sourceDir, "Makefile")
	if _, err := os.Stat(cleanMakefile); err == nil {
		// Try make clean
		cmd := exec.Command("make", "clean")
		cmd.Dir = sourceDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// If make clean fails, continue with manual cleanup
		}
	}

	// Also manually remove files that might cause issues
	patterns := []string{
		"Makefile",
		"Makefile.fragments",
		"Makefile.global",
		"Makefile.objects",
		"acinclude.m4",
		"build.mk",
		"configure",
		"main/php_config.h",
		"main/internal_functions.c",
		"main/internal_functions_cli.c",
		"*.lo",
		"*.o",
		".deps",
		"libtool",
		"ltmain.sh",
		"autom4te.cache",
	}

	// Remove files matching patterns
	for _, pattern := range patterns {
		if strings.Contains(pattern, "*") {
			// Handle wildcards - find matching files
			files, err := filepath.Glob(filepath.Join(sourceDir, pattern))
			if err != nil {
				continue
			}
			for _, f := range files {
				if err := os.RemoveAll(f); err != nil {
					fmt.Printf("Warning: Failed to remove %s: %v\n", f, err)
				}
			}
		} else {
			// Handle specific files/directories
			path := filepath.Join(sourceDir, pattern)
			if _, err := os.Stat(path); err == nil {
				if err := os.RemoveAll(path); err != nil {
					fmt.Printf("Warning: Failed to remove %s: %v\n", path, err)
				}
			}
		}
	}

	// Also clean any ext/*/Makefile files
	extDir := filepath.Join(sourceDir, "ext")
	if entries, err := os.ReadDir(extDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				extMakefile := filepath.Join(extDir, entry.Name(), "Makefile")
				if _, err := os.Stat(extMakefile); err == nil {
					os.RemoveAll(extMakefile)
				}
			}
		}
	}

	fmt.Println("  Cleaned source directory")
	return nil
}
