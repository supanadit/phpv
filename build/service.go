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
	"github.com/supanadit/phpv/internal/ui"
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

	ui := ui.GetUI()

	// Create versions directory
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	compiler := s.compilerDisplayName(version)
	ui.PrintInfo(fmt.Sprintf("Building PHP %s from source using %s...", versionStr, compiler))

	info := map[string]string{
		"Source": sourceDir,
		"Target": installDir,
	}
	ui.PrintBuildInfo("PHP Build Information", info)
	ui.Println()

	// Step 1: Build dependencies (this will download and install LLVM)
	ui.PrintProcessingStep(1, 6, "Building dependencies...")
	if err := s.depService.BuildDependencies(ctx, version); err != nil {
		return fmt.Errorf("failed to build dependencies: %w", err)
	}
	ui.StopProcessingStep()

	// Get environment with dependency paths (needed for buildconf and configure)
	env := s.depService.GetPHPEnvironment(version)

	// Step 2: Run buildconf (if it exists)
	// For old PHP versions (5.2 and earlier), skip buildconf entirely - the configure script is already present
	// and regenerating it causes compatibility issues with modern shells
	buildconfPath := filepath.Join(sourceDir, "buildconf")
	skipBuildconf := version.Major == 5 && version.Minor <= 2
	if skipBuildconf {
		ui.PrintProcessingStep(2, 6, "Skipping buildconf (using pre-generated configure)...")
		ui.StopProcessingStep()
	} else if _, err := os.Stat(buildconfPath); err == nil {
		ui.PrintProcessingStep(2, 6, "Running buildconf...")

		// Set up PATH with version-specific autoconf first
		autoconfBin := filepath.Join(s.depService.GetDependencyInstallDir(version, "autoconf"), "bin")
		env = s.addToPathEnv(env, autoconfBin)
		// Use the current env's PATH, not os.Getenv
		env = s.setEnvVar(env, "PATH", autoconfBin+":"+s.getEnvVar(env, "PATH"))

		if skipBuildconf {
			// Patch buildcheck.sh to bypass autoconf version check
			buildcheckPath := filepath.Join(sourceDir, "build", "buildcheck.sh")
			if data, err := os.ReadFile(buildcheckPath); err == nil {
				// Replace the version check that fails for autoconf > 2.59
				patched := strings.Replace(string(data),
					`if test "$1" = "2" -a "$2" -gt "59"; then`,
					`if test "$1" = "2" -a "$2" -gt "99"; then`, 1)
				os.WriteFile(buildcheckPath, []byte(patched), 0755)
			}

			// Run buildconf with --force
			if err := util.RunCommand(ctx, sourceDir, env, "./buildconf", "--force"); err != nil {
				return fmt.Errorf("buildconf failed: %w", err)
			}
		} else {
			if _, err := os.Stat(filepath.Join(autoconfBin, "autoconf")); err == nil {
				env = s.setEnvVar(env, "AUTOCONF", filepath.Join(autoconfBin, "autoconf"))
				env = s.setEnvVar(env, "AUTOHEADER", filepath.Join(autoconfBin, "autoheader"))
				env = s.setEnvVar(env, "AUTOMAKE", filepath.Join(autoconfBin, "automake"))
				env = s.setEnvVar(env, "ACLOCAL", filepath.Join(autoconfBin, "aclocal"))
			}
			if err := util.RunCommand(ctx, sourceDir, env, "./buildconf", "--force"); err != nil {
				return fmt.Errorf("buildconf failed: %w", err)
			}
		}
		ui.StopProcessingStep()
	}

	// Step 3: Configure with dependency paths
	ui.PrintProcessingStep(3, 6, "Configuring PHP build...")

	// Base configure arguments
	configureArgs := []string{
		fmt.Sprintf("--prefix=%s", installDir),
		"--disable-static",
		"--enable-shared",
		"--disable-all",
		"--enable-cli",
		"--enable-phar",
		"--enable-ctype",
		"--enable-tokenizer",
		"--enable-xml",
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

		// PHP 5.3+ has mbstring, but 5.2 and earlier have issues with bundled oniguruma
		if version.Minor >= 3 {
			configureArgs = append(configureArgs, "--enable-mbstring")
		}

		// PHP 5.2 and earlier have configure script compatibility issues with modern shells
		// and libxml2 API changes - disable problematic extensions
		if version.Minor <= 2 {
			// PHP 5.2 and earlier: disable DOM/SimpleXML extensions (libxml2 API incompatibility)
			// Disable filter extension (requires PCRE)
			configureArgs = append(configureArgs, "--without-pcre-regex")
			configureArgs = append(configureArgs, "--enable-ftp")
			configureArgs = append(configureArgs, "--enable-zlib")
			configureArgs = append(configureArgs, "--with-curl")
		}

		// PHP 5.1 and 5.0: use system libxml2 since older versions need autoconf 2.63+ which conflicts with PHP's autoconf 2.59
		// Also disable OpenSSL, Curl and Mbstring extensions since they are incompatible with modern compilers/versions
		if version.Minor <= 1 {
			configureArgs = append(configureArgs, "--with-libxml")
			configureArgs = append(configureArgs, "--without-openssl")
			configureArgs = append(configureArgs, "--without-curl")
			configureArgs = append(configureArgs, "--without-mbstring")
		}
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
		// PHP 5.2 needs PCRE - use system or bundled
		configureArgs = append(configureArgs, "--without-pcre-regex")
		configureArgs = append(configureArgs, "--enable-ftp")
		configureArgs = append(configureArgs, "--enable-zlib")
		// PHP 4's OpenSSL extension is incompatible with modern OpenSSL versions
		configureArgs = append(configureArgs, "--without-openssl")
		// PHP 4 doesn't have oniguruma extension
		configureArgs = append(configureArgs, "--without-oniguruma")
		// PHP 4's Curl extension has issues with modern libcurl
		configureArgs = append(configureArgs, "--without-curl")
	}

	// Add dependency-specific configure flags
	depFlags := s.depService.GetPHPConfigureFlags(version)
	configureArgs = append(configureArgs, depFlags...)

	if err := util.RunCommand(ctx, sourceDir, env, "./configure", configureArgs...); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}
	ui.StopProcessingStep()

	// Step 4: Make
	ui.PrintProcessingStep(4, 6, "Compiling PHP (this may take a while)...")
	if err := util.RunCommand(ctx, sourceDir, env, "make", "-j4"); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}
	ui.StopProcessingStep()

	// Step 5: Make install
	ui.PrintProcessingStep(5, 6, "Installing PHP...")
	if err := util.RunCommand(ctx, sourceDir, env, "make", "install"); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}
	ui.StopProcessingStep()

	// Step 6: Verify installation
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		return fmt.Errorf("PHP binary not found at %s after installation", phpBinary)
	}

	// Step 7: Test the binary
	ui.PrintProcessingStep(6, 6, "Testing PHP binary...")
	if err := util.RunCommand(ctx, sourceDir, nil, phpBinary, "--version"); err != nil {
		return fmt.Errorf("PHP binary test failed: %w", err)
	}
	ui.StopProcessingStep()

	ui.Println()
	ui.PrintBuildComplete("PHP", versionStr, installDir)
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

	// Check if we should use Zig (explicit opt-in)
	if domain.ShouldUseZigToolchain(version) {
		zigVersion := domain.GetZigVersion()
		return fmt.Sprintf("Zig %s (zig cc)", zigVersion.Version)
	}

	// Check if we should use LLVM (explicit opt-in with PHPV_USE_LLVM=1)
	if domain.ShouldUseLLVMToolchain(version) {
		// Get LLVM version for this PHP version
		llvmVersion := domain.GetLLVMVersionForPHP(version)
		return fmt.Sprintf("LLVM %s (clang)", llvmVersion.Version)
	}

	// Default: Use system GCC (works on all systems, no libtinfo issues)
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

	// Check if we should use Zig (explicit opt-in with PHPV_USE_ZIG=1)
	if domain.ShouldUseZigToolchain(version) {
		return "zig"
	}

	// Check if we should use LLVM (explicit opt-in with PHPV_USE_LLVM=1)
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

	// Default: Use system GCC
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

func (s *Service) getEnvVar(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
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
