package dependency

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/ui"
	"github.com/supanadit/phpv/internal/util"
)

type Service struct {
	httpClient       *http.Client
	phpvRoot         string
	toolchain        *domain.ToolchainConfig
	toolchainService *ToolchainService
	phpVersion       *domain.Version // Used to determine which LLVM toolchain to use
}

func NewService(phpvRoot string) *Service {
	return NewServiceWithToolchain(phpvRoot, nil)
}

// NewServiceWithToolchain allows providing an optional toolchain configuration.
func NewServiceWithToolchain(phpvRoot string, toolchain *domain.ToolchainConfig) *Service {
	return &Service{
		httpClient:       &http.Client{},
		phpvRoot:         phpvRoot,
		toolchain:        toolchain,
		toolchainService: NewToolchainService(phpvRoot),
		phpVersion:       nil, // Will be set when building dependencies
	}
}

// GetCacheDir returns the cache directory for downloaded archives
func (s *Service) GetCacheDir() string {
	return filepath.Join(s.phpvRoot, "cache", "sources")
}

// GetDependenciesDir returns the dependencies directory for a PHP version
func (s *Service) GetDependenciesDir(phpVersion domain.Version) string {
	versionStr := fmt.Sprintf("%d.%d.%d", phpVersion.Major, phpVersion.Minor, phpVersion.Patch)
	return filepath.Join(s.phpvRoot, "dependencies", versionStr)
}

// GetDependencyInstallDir returns the install directory for a specific dependency
func (s *Service) GetDependencyInstallDir(phpVersion domain.Version, depName string) string {
	return filepath.Join(s.GetDependenciesDir(phpVersion), depName)
}

// GetDependencySourceDir returns the source directory for a dependency (isolated per PHP version)
func (s *Service) GetDependencySourceDir(phpVersion domain.Version, dep domain.Dependency) string {
	versionStr := fmt.Sprintf("%d.%d.%d", phpVersion.Major, phpVersion.Minor, phpVersion.Patch)
	return filepath.Join(s.phpvRoot, "dependencies-src", versionStr, dep.Name+"-"+dep.Version)
}

// IsDependencyBuilt checks if a dependency is already built
func (s *Service) IsDependencyBuilt(phpVersion domain.Version, dep domain.Dependency) bool {
	// LLVM is managed by the toolchain service
	if dep.Name == "llvm" {
		return s.toolchainService.IsLLVMInstalled(dep.Version)
	}

	// Check if system dependency is available (for PHP 8.3+)
	if s.isSystemDependencyAvailable(dep.Name) {
		return true
	}

	installDir := s.GetDependencyInstallDir(phpVersion, dep.Name)

	// Special checks for tool dependencies that install binaries, not libraries
	toolDeps := []string{"cmake", "perl", "m4", "autoconf", "automake", "libtool", "re2c", "flex", "bison"}
	for _, tool := range toolDeps {
		if dep.Name == tool {
			binPath := filepath.Join(installDir, "bin", dep.Name)
			if _, err := os.Stat(binPath); err == nil {
				return true
			}
			return false
		}
	}

	// For library dependencies, check if lib directory exists with some files
	libDir := filepath.Join(installDir, "lib")
	if stat, err := os.Stat(libDir); err == nil && stat.IsDir() {
		// Check if there are any files in lib directory
		entries, err := os.ReadDir(libDir)
		return err == nil && len(entries) > 0
	}
	return false
}

// isSystemDependencyAvailable checks if a system dependency is available
// This is used for PHP 8.3+ which can use system libraries
func (s *Service) isSystemDependencyAvailable(depName string) bool {
	// Only use system deps for PHP 8.3+ (or when not using LLVM)
	if s.phpVersion == nil {
		return false
	}
	if !domain.ShouldUseLLVMToolchain(*s.phpVersion) {
		// Use the new version-aware system dependency checking
		result := domain.CheckSystemDependency(depName, *s.phpVersion)
		return result.CanUse
	}
	return false
}

// GetSystemDependencyPath returns the system library path for a dependency
func (s *Service) GetSystemDependencyPath(depName string) (string, string, string, bool) {
	// Map dependency names to pkg-config names and default paths
	pkgConfigMap := map[string]struct {
		pkg        string
		defaultLib string
		defaultInc string
	}{
		"zlib":      {"zlib", "/usr/lib", "/usr/include"},
		"libxml2":   {"libxml-2.0", "/usr/lib", "/usr/include/libxml2"},
		"openssl":   {"openssl", "/usr/lib", "/usr/include/openssl"},
		"curl":      {"libcurl", "/usr/lib", "/usr/include"},
		"oniguruma": {"oniguruma", "/usr/lib", "/usr/include"},
	}

	info, ok := pkgConfigMap[depName]
	if !ok {
		return "", "", "", false
	}

	// Check if pkg-config can find it
	cmd := exec.Command("pkg-config", "--exists", info.pkg)
	if err := cmd.Run(); err != nil {
		return "", "", "", false
	}

	// Get the library and include paths
	libCmd := exec.Command("pkg-config", "--libs", "-L", info.pkg)
	libOut, _ := libCmd.Output()
	incCmd := exec.Command("pkg-config", "--cflags", info.pkg)
	incOut, _ := incCmd.Output()

	libPath := strings.TrimSpace(string(libOut))
	incPath := strings.TrimSpace(string(incOut))

	// Parse -I prefix from cflags
	incPath = strings.TrimPrefix(incPath, "-I")

	return libPath, incPath, info.pkg, true
}

// BuildDependencies builds all dependencies for a PHP version
func (s *Service) BuildDependencies(ctx context.Context, phpVersion domain.Version) error {
	// Store the PHP version to ensure we use the correct LLVM toolchain
	s.phpVersion = &phpVersion

	ui := ui.GetUI()

	deps := GetDependenciesForVersion(phpVersion)

	ui.PrintSection(fmt.Sprintf("Building Dependencies for PHP %d.%d.%d", phpVersion.Major, phpVersion.Minor, phpVersion.Patch))

	// Check if we should use Zig, LLVM, or system toolchain
	useZig := domain.ShouldUseZigToolchain(phpVersion)
	useLLVM := domain.ShouldUseLLVMToolchain(phpVersion)

	// Check and report system dependencies for PHP 8.3+
	if !useLLVM && !useZig {
		s.checkAndReportSystemDeps(phpVersion)
	}

	// First, ensure Zig is installed if requested
	if useZig {
		ui.PrintInfo("Using Zig compiler (PHPV_USE_ZIG=1)")
		if err := s.toolchainService.DownloadAndInstallZig(ctx); err != nil {
			return fmt.Errorf("failed to install Zig: %w", err)
		}
		// Update toolchain configuration to use Zig
		if s.toolchain == nil || s.toolchain.IsEmpty() {
			s.toolchain = s.toolchainService.GetZigToolchainConfig()
		}
	}

	// First, ensure LLVM is installed if needed (for PHP < 8.3 or PHPV_USE_LLVM=1)
	if useLLVM && !useZig {
		for _, dep := range deps {
			if dep.Name == "llvm" {
				if err := s.toolchainService.DownloadAndInstallLLVM(ctx, phpVersion); err != nil {
					return fmt.Errorf("failed to install LLVM: %w", err)
				}
				// Update toolchain configuration to use the downloaded LLVM
				if s.toolchain == nil || s.toolchain.IsEmpty() {
					s.toolchain = s.toolchainService.GetToolchainConfig(phpVersion)
				}
				break
			}
		}
	} else if !useZig {
		ui.PrintInfo("Using system GCC (no LLVM needed)")
	}

	// Build dependencies in order (respecting transitive dependencies)
	builtDeps := make(map[string]bool)

	for _, dep := range deps {
		if err := s.buildDependencyWithDeps(ctx, phpVersion, dep, deps, builtDeps); err != nil {
			return err
		}
	}

	ui.PrintSuccess("All dependencies built successfully")
	ui.Println()
	return nil
}

// checkAndReportSystemDeps checks system dependencies and prints a report
func (s *Service) checkAndReportSystemDeps(phpVersion domain.Version) {
	ui := ui.GetUI()
	ui.PrintInfo("Checking system dependencies...")

	toolDeps := []string{"autoconf", "automake", "libtool", "cmake", "perl", "m4", "re2c", "flex", "bison"}
	libDeps := []string{"zlib", "libxml2", "openssl", "curl", "oniguruma"}

	ui.PrintSubheader("Build Tools:")
	for _, depName := range toolDeps {
		result := domain.CheckSystemDependency(depName, phpVersion)
		if result.Found {
			if result.CanUse {
				ui.PrintDependencyStatus(result.Name, result.Version, result.MinVersion, true)
			} else {
				ui.PrintAction("Building", fmt.Sprintf("%s (need ≥%s)", result.Name, result.MinVersion))
			}
		} else {
			ui.PrintAction("Building", fmt.Sprintf("%s (not found)", depName))
		}
	}

	ui.PrintSubheader("Libraries:")
	for _, depName := range libDeps {
		result := domain.CheckSystemDependency(depName, phpVersion)
		if result.Found {
			if result.CanUse {
				ui.PrintDependencyStatus(result.Name, result.Version, "", true)
			} else {
				ui.PrintAction("Building", fmt.Sprintf("%s (too old)", depName))
			}
		} else {
			ui.PrintAction("Building", fmt.Sprintf("%s (not found)", depName))
		}
	}

	ui.Println()

	s.validateDependencyConstraints(phpVersion)
}

func (s *Service) validateDependencyConstraints(phpVersion domain.Version) {
	ui := ui.GetUI()

	config := getConfigForVersion(phpVersion)

	specs := map[string]domain.DependencyVersionSpec{
		"perl":      config.Perl,
		"m4":        config.M4,
		"autoconf":  config.Autoconf,
		"automake":  config.Automake,
		"libtool":   config.Libtool,
		"re2c":      config.Re2c,
		"flex":      config.Flex,
		"bison":     config.Bison,
		"zlib":      config.Zlib,
		"libxml2":   config.Libxml2,
		"openssl":   config.OpenSSL,
		"curl":      config.Curl,
		"oniguruma": config.Oniguruma,
	}

	currentVersions := make(map[string]string)
	depNames := []string{"perl", "m4", "autoconf", "automake", "libtool", "re2c", "flex", "bison", "zlib", "libxml2", "openssl", "curl", "oniguruma"}

	for _, depName := range depNames {
		result := domain.CheckSystemDependency(depName, phpVersion)
		if result.Found {
			currentVersions[depName] = result.Version
		}
	}

	warnings := domain.ValidateAllDependencies(specs, currentVersions)

	if len(warnings) > 0 {
		ui.PrintSubheader("Dependency Constraint Warnings:")
		for _, w := range warnings {
			ui.PrintWarning(w.Message)
		}
		ui.Println()
	}
}

// buildDependencyWithDeps recursively builds a dependency and its dependencies
func (s *Service) buildDependencyWithDeps(ctx context.Context, phpVersion domain.Version, dep domain.Dependency, allDeps []domain.Dependency, built map[string]bool) error {
	ui := ui.GetUI()

	// Skip if already built
	if built[dep.Name] {
		return nil
	}

	// Check if already installed
	if s.IsDependencyBuilt(phpVersion, dep) {
		ui.PrintAlreadyBuilt(dep.Name, dep.Version)
		built[dep.Name] = true
		return nil
	}

	// Build transitive dependencies first
	for _, depName := range dep.Dependencies {
		var transDep *domain.Dependency
		for i := range allDeps {
			if allDeps[i].Name == depName {
				transDep = &allDeps[i]
				break
			}
		}
		if transDep == nil {
			return fmt.Errorf("transitive dependency %s not found", depName)
		}
		if err := s.buildDependencyWithDeps(ctx, phpVersion, *transDep, allDeps, built); err != nil {
			return err
		}
	}

	// Build this dependency
	if err := s.BuildDependency(ctx, phpVersion, dep); err != nil {
		return err
	}

	built[dep.Name] = true
	return nil
}

// BuildDependency downloads and builds a single dependency
func (s *Service) BuildDependency(ctx context.Context, phpVersion domain.Version, dep domain.Dependency) error {
	fmt.Printf("\n--- Building %s %s ---\n", dep.Name, dep.Version)

	installDir := s.GetDependencyInstallDir(phpVersion, dep.Name)
	sourceDir := s.GetDependencySourceDir(phpVersion, dep)

	// LLVM is handled specially by the toolchain service
	if dep.Name == "llvm" {
		// Already installed by BuildDependencies
		fmt.Printf("✓ %s %s already installed by toolchain service\n", dep.Name, dep.Version)
		return nil
	}

	// For prebuilt dependencies, download directly to installDir
	if len(dep.BuildCommands) > 0 && dep.BuildCommands[0] == "prebuilt" {
		if _, err := os.Stat(installDir); os.IsNotExist(err) {
			fmt.Printf("Downloading %s...\n", dep.Name)
			if err := s.downloadAndExtract(ctx, dep.DownloadURL, installDir); err != nil {
				return fmt.Errorf("failed to download %s: %w", dep.Name, err)
			}
		} else {
			fmt.Printf("%s already installed: %s\n", dep.Name, installDir)
		}
		fmt.Printf("✓ %s %s installed successfully\n", dep.Name, dep.Version)
		return nil
	}

	// Download if not exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Printf("Downloading %s...\n", dep.Name)
		if err := s.downloadAndExtract(ctx, dep.DownloadURL, sourceDir); err != nil {
			return fmt.Errorf("failed to download %s: %w", dep.Name, err)
		}
	} else {
		fmt.Printf("Source already downloaded: %s\n", sourceDir)
	}

	// Prepare environment with dependency paths
	env := s.buildEnvironment(phpVersion, dep)

	// Clean any previous build artifacts to avoid automake regeneration issues
	makefilePath := filepath.Join(sourceDir, "Makefile")
	configurePath := filepath.Join(sourceDir, "configure")
	autogenPath := filepath.Join(sourceDir, "autogen.sh")

	if _, err := os.Stat(makefilePath); err == nil {
		fmt.Printf("Cleaning previous build artifacts...\n")

		// Determine which files to remove based on the dependency
		filesToRemove := []string{
			"Makefile",
			"config.status",
			"config.log",
			"config.h",
			"config.cache",
		}

		// Don't remove configure and related files for packages that ship with them
		// These include: zlib (CMake), m4, autoconf, automake, libtool (stable GNU packages)
		// For curl with ./buildconf or ./reconf, keep the shipped configure script
		useBuildconf := false
		useReconf := false
		for _, cmd := range dep.BuildCommands {
			if cmd == "./buildconf" {
				useBuildconf = true
				break
			}
			if cmd == "./reconf" {
				useReconf = true
				break
			}
		}
		shouldKeepConfigure := dep.Name == "zlib" || dep.Name == "m4" ||
			dep.Name == "autoconf" || dep.Name == "automake" || dep.Name == "libtool" ||
			dep.Name == "bison" ||
			useBuildconf || useReconf

		if !shouldKeepConfigure {
			filesToRemove = append(filesToRemove,
				"Makefile.in",
				"config.h.in",
				"configure",
				"aclocal.m4",
				"autom4te.cache",
				"libtool",
				"stamp-h1",
			)
		}

		for _, file := range filesToRemove {
			path := filepath.Join(sourceDir, file)
			if _, err := os.Stat(path); err == nil {
				os.RemoveAll(path)
			}
		}
		// Also remove any .deps directories
		filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() && info.Name() == ".deps" {
				os.RemoveAll(path)
			}
			return nil
		})
	}

	// Regenerate configure script if needed
	if _, err := os.Stat(configurePath); os.IsNotExist(err) {
		// Handle bootstrap build for autotools that don't ship with configure
		if len(dep.BuildCommands) > 0 && dep.BuildCommands[0] == "bootstrap" {
			// Check if there's a bootstrap script
			bootstrapPath := filepath.Join(sourceDir, "bootstrap")
			if _, err := os.Stat(bootstrapPath); err == nil {
				fmt.Printf("Running bootstrap script for %s...\n", dep.Name)
				if err := util.RunCommand(ctx, sourceDir, env, "./bootstrap"); err != nil {
					return fmt.Errorf("bootstrap failed for %s: %w", dep.Name, err)
				}
			} else {
				fmt.Printf("Generating configure script for %s using autoreconf...\n", dep.Name)
				// Fallback to autoreconf if no bootstrap script exists
				autoreconfPath := filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin", "autoreconf")
				if err := util.RunCommand(ctx, sourceDir, env, autoreconfPath, "-fi"); err != nil {
					return fmt.Errorf("autoreconf failed for %s (system autoreconf required for bootstrapping): %w", dep.Name, err)
				}
			}
		} else {
			// Check for autogen.sh first
			if _, err := os.Stat(autogenPath); err == nil {
				fmt.Printf("Running autogen.sh to regenerate configure script...\n")
				if err := util.RunCommand(ctx, sourceDir, env, "./autogen.sh"); err != nil {
					return fmt.Errorf("autogen.sh failed for %s: %w", dep.Name, err)
				}
			} else {
				// Check for buildconf (used by curl) or configure.ac (used by most autotools projects)
				buildconfPath := filepath.Join(sourceDir, "buildconf")
				reconfPath := filepath.Join(sourceDir, "reconf")
				configureAcPath := filepath.Join(sourceDir, "configure.ac")

				// If BuildCommands explicitly specifies "./buildconf" or "./reconf", use it directly
				useBuildconf := false
				useReconf := false
				for _, cmd := range dep.BuildCommands {
					if cmd == "./buildconf" {
						useBuildconf = true
						break
					}
					if cmd == "./reconf" {
						useReconf = true
						break
					}
				}

				if useBuildconf {
					configurePath := filepath.Join(sourceDir, "configure")
					if _, err := os.Stat(configurePath); err == nil {
						fmt.Printf("configure already exists, skipping buildconf\n")
					} else {
						// Use per-version autoconf for regenerating configure
						autoconfBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin")
						depsBin := s.getDependencyBinPath(phpVersion)
						env = setOrReplaceEnv(env, "PATH", autoconfBin+":"+depsBin+":"+getEnvValue(env, "PATH"))
						env = setOrReplaceEnv(env, "AUTOCONF", filepath.Join(autoconfBin, "autoconf"))
						env = setOrReplaceEnv(env, "AUTOMAKE", filepath.Join(autoconfBin, "automake"))
						env = setOrReplaceEnv(env, "ACLOCAL", filepath.Join(autoconfBin, "aclocal"))
						fmt.Printf("Running buildconf to regenerate configure script...\n")
						if err := util.RunCommand(ctx, sourceDir, env, "./buildconf"); err != nil {
							return fmt.Errorf("buildconf failed for %s: %w", dep.Name, err)
						}
					}
				} else if useReconf {
					if _, err := os.Stat(reconfPath); err == nil {
						// For old curl with reconf, copy all libtool macros to source directory
						// because old aclocal (1.4-p6) doesn't support ACLOCAL_PATH
						libtoolShare := filepath.Join(s.GetDependencyInstallDir(phpVersion, "libtool"), "share", "aclocal")
						if _, err := os.Stat(libtoolShare); err == nil {
							// Copy all libtool .m4 files to current directory
							m4Files := []string{"libtool.m4", "ltoptions.m4", "ltsugar.m4", "ltversion.m4", "lt~obsolete.m4"}
							for _, m4File := range m4Files {
								src := filepath.Join(libtoolShare, m4File)
								dst := filepath.Join(sourceDir, m4File)
								if srcData, err := os.ReadFile(src); err == nil {
									os.WriteFile(dst, srcData, 0644)
								}
							}
						}

						// Old automake 1.4 expects configure.in instead of configure.ac
						configureAcPath := filepath.Join(sourceDir, "configure.ac")
						configureInPath := filepath.Join(sourceDir, "configure.in")
						if _, err := os.Stat(configureAcPath); err == nil {
							if _, err := os.Stat(configureInPath); os.IsNotExist(err) {
								// Create symlink from configure.in to configure.ac
								os.Symlink("configure.ac", configureInPath)
							}
						}

						fmt.Printf("Running reconf to regenerate configure script...\n")
						if err := util.RunCommand(ctx, sourceDir, env, "./reconf"); err != nil {
							return fmt.Errorf("reconf failed for %s: %w", dep.Name, err)
						}
					}
				} else if _, err := os.Stat(buildconfPath); err == nil {
					// Use buildconf for modern curl versions
					// Use per-version autoconf for regenerating configure
					autoconfBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin")
					depsBin := s.getDependencyBinPath(phpVersion)
					env = setOrReplaceEnv(env, "PATH", autoconfBin+":"+depsBin+":"+getEnvValue(env, "PATH"))
					env = setOrReplaceEnv(env, "AUTOCONF", filepath.Join(autoconfBin, "autoconf"))
					env = setOrReplaceEnv(env, "AUTOMAKE", filepath.Join(autoconfBin, "automake"))
					env = setOrReplaceEnv(env, "ACLOCAL", filepath.Join(autoconfBin, "aclocal"))
					fmt.Printf("Running buildconf to regenerate configure script...\n")
					if err := util.RunCommand(ctx, sourceDir, env, "./buildconf"); err != nil {
						return fmt.Errorf("buildconf failed for %s: %w", dep.Name, err)
					}
				} else if _, err := os.Stat(configureAcPath); err == nil {
					autoreconfPath := s.getAutoreconfPath(phpVersion)
					if autoreconfPath == "" {
						return fmt.Errorf("autoreconf not available for %s", dep.Name)
					}
					if err := util.RunCommand(ctx, sourceDir, env, autoreconfPath, "-fi"); err != nil {
						return fmt.Errorf("autoreconf failed for %s: %w", dep.Name, err)
					}
				}
			}
		}
	}

	// Configure
	configureCmd := "./configure"
	configureArgs := append([]string{fmt.Sprintf("--prefix=%s", installDir)}, dep.ConfigureFlags...)

	// Special handling for Perl which uses ./Configure (capitalized)
	if len(dep.BuildCommands) > 0 && dep.BuildCommands[0] == "./Configure" {
		configureCmd = "./Configure"
		configureArgs = append([]string{fmt.Sprintf("-Dprefix=%s", installDir)}, dep.ConfigureFlags...)

		// For Perl, explicitly set the compiler to ensure it uses LLVM clang
		if cc := getEnvValue(env, "CC"); cc != "" {
			configureArgs = append(configureArgs, fmt.Sprintf("-Dcc=%s", cc))
		}
	}

	// Special handling for OpenSSL which uses ./config
	if len(dep.BuildCommands) > 0 && strings.Contains(dep.BuildCommands[0], "config") {
		configureCmd = dep.BuildCommands[0]
		configureArgs = append([]string{fmt.Sprintf("--prefix=%s", installDir)}, dep.ConfigureFlags...)
	}

	// Special handling for CMake-based builds
	if len(dep.BuildCommands) > 0 && dep.BuildCommands[0] == "cmake" {
		cmakeBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "cmake"), "bin")
		configureCmd = filepath.Join(cmakeBin, "cmake")
		configureArgs = append([]string{"."}, dep.ConfigureFlags...)
		// Replace %s placeholder with actual installDir
		for i, arg := range configureArgs {
			configureArgs[i] = strings.ReplaceAll(arg, "%s", installDir)
		}
	}

	fmt.Printf("Configuring %s...\n", dep.Name)
	if err := util.RunCommand(ctx, sourceDir, env, configureCmd, configureArgs...); err != nil {
		return fmt.Errorf("configure failed for %s: %w", dep.Name, err)
	}

	// Make
	fmt.Printf("Compiling %s...\n", dep.Name)
	if err := util.RunCommand(ctx, sourceDir, env, "make", "-j4"); err != nil {
		return fmt.Errorf("make failed for %s: %w", dep.Name, err)
	}

	// Install
	fmt.Printf("Installing %s to %s...\n", dep.Name, installDir)
	if err := util.RunCommand(ctx, sourceDir, env, "make", "install"); err != nil {
		return fmt.Errorf("make install failed for %s: %w", dep.Name, err)
	}

	fmt.Printf("✓ %s %s built successfully\n", dep.Name, dep.Version)
	return nil
}

// buildEnvironment creates environment variables for building dependencies
func (s *Service) buildEnvironment(phpVersion domain.Version, dep domain.Dependency) []string {
	env := s.getCleanBaseEnv()

	// Use LLVM toolchain or system GCC based on PHP version
	env = s.applyCompilerEnv(env)

	// Check if using system dependencies
	useSystemDeps := !domain.ShouldUseLLVMToolchain(phpVersion)

	// Get system PATH first - we want system tools to take precedence
	systemPath := "/usr/local/bin:/usr/bin:/bin"

	if useSystemDeps {
		// Add system tool paths first (but keep our PATH clean)
		// Only add built tool paths if system version is not available

		// Add cmake to PATH only if system cmake is not available or too old
		cmakeResult := domain.CheckSystemDependency("cmake", phpVersion)
		if !cmakeResult.CanUse {
			cmakeBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "cmake"), "bin")
			if _, err := os.Stat(filepath.Join(cmakeBin, "cmake")); err == nil {
				systemPath = cmakeBin + ":" + systemPath
			}
		}

		// Add perl to PATH only if system perl is not available or too old
		perlResult := domain.CheckSystemDependency("perl", phpVersion)
		if !perlResult.CanUse {
			perlBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "perl"), "bin")
			if _, err := os.Stat(filepath.Join(perlBin, "perl")); err == nil {
				systemPath = perlBin + ":" + systemPath
			}
		}

		// Add m4 to PATH only if system m4 is not available or too old
		m4Result := domain.CheckSystemDependency("m4", phpVersion)
		if !m4Result.CanUse {
			m4Bin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "m4"), "bin")
			if _, err := os.Stat(filepath.Join(m4Bin, "m4")); err == nil {
				systemPath = m4Bin + ":" + systemPath
			}
		}

		// Add autoconf to PATH only if system autoconf is not available or too old
		autoconfResult := domain.CheckSystemDependency("autoconf", phpVersion)
		if !autoconfResult.CanUse {
			autoconfBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin")
			if _, err := os.Stat(filepath.Join(autoconfBin, "autoconf")); err == nil {
				systemPath = autoconfBin + ":" + systemPath
			}
		}

		// Add automake to PATH only if system automake is not available or too old
		automakeResult := domain.CheckSystemDependency("automake", phpVersion)
		if !automakeResult.CanUse {
			automakeBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "automake"), "bin")
			if _, err := os.Stat(filepath.Join(automakeBin, "automake")); err == nil {
				systemPath = automakeBin + ":" + systemPath
			}
		}

		// Add libtool to PATH only if system libtool is not available or too old
		libtoolResult := domain.CheckSystemDependency("libtool", phpVersion)
		if !libtoolResult.CanUse {
			libtoolBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "libtool"), "bin")
			if _, err := os.Stat(filepath.Join(libtoolBin, "libtoolize")); err == nil {
				systemPath = libtoolBin + ":" + systemPath
			}
		}

		// Add re2c to PATH only if system re2c is not available or too old
		re2cResult := domain.CheckSystemDependency("re2c", phpVersion)
		if !re2cResult.CanUse {
			re2cBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "re2c"), "bin")
			if _, err := os.Stat(filepath.Join(re2cBin, "re2c")); err == nil {
				systemPath = re2cBin + ":" + systemPath
			}
		}

		// Add bison to PATH only if system bison is not available or too old
		bisonResult := domain.CheckSystemDependency("bison", phpVersion)
		if !bisonResult.CanUse {
			bisonBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "bison"), "bin")
			if _, err := os.Stat(filepath.Join(bisonBin, "bison")); err == nil {
				systemPath = bisonBin + ":" + systemPath
			}
		}

		// Add flex to PATH only if system flex is not available or too old
		flexResult := domain.CheckSystemDependency("flex", phpVersion)
		if !flexResult.CanUse {
			flexBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "flex"), "bin")
			if _, err := os.Stat(filepath.Join(flexBin, "flex")); err == nil {
				systemPath = flexBin + ":" + systemPath
			}
		}
	} else {
		// Using LLVM - add all built tool paths
		// Add cmake to PATH if available
		cmakeBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "cmake"), "bin")
		if _, err := os.Stat(filepath.Join(cmakeBin, "cmake")); err == nil {
			systemPath = cmakeBin + ":" + systemPath
		}

		// Add perl to PATH if available
		perlBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "perl"), "bin")
		if _, err := os.Stat(filepath.Join(perlBin, "perl")); err == nil {
			systemPath = perlBin + ":" + systemPath
		}

		// Add m4 to PATH if available
		m4Bin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "m4"), "bin")
		if _, err := os.Stat(filepath.Join(m4Bin, "m4")); err == nil {
			systemPath = m4Bin + ":" + systemPath
		}

		// Add autoconf to PATH if available
		autoconfBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin")
		if _, err := os.Stat(filepath.Join(autoconfBin, "autoconf")); err == nil {
			systemPath = autoconfBin + ":" + systemPath
		}

		// Add automake to PATH if available
		automakeBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "automake"), "bin")
		if _, err := os.Stat(filepath.Join(automakeBin, "automake")); err == nil {
			systemPath = automakeBin + ":" + systemPath
		}

		// Add libtool to PATH if available
		libtoolBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "libtool"), "bin")
		if _, err := os.Stat(filepath.Join(libtoolBin, "libtoolize")); err == nil {
			systemPath = libtoolBin + ":" + systemPath
		}

		// Add re2c to PATH if available
		re2cBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "re2c"), "bin")
		if _, err := os.Stat(filepath.Join(re2cBin, "re2c")); err == nil {
			systemPath = re2cBin + ":" + systemPath
		}

		// Add bison to PATH if available
		bisonBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "bison"), "bin")
		if _, err := os.Stat(filepath.Join(bisonBin, "bison")); err == nil {
			systemPath = bisonBin + ":" + systemPath
		}

		// Add flex to PATH if available
		flexBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "flex"), "bin")
		if _, err := os.Stat(filepath.Join(flexBin, "flex")); err == nil {
			systemPath = flexBin + ":" + systemPath
		}
	}

	env = setOrReplaceEnv(env, "PATH", systemPath)

	if os.Getenv("PHPV_DEBUG") == "1" {
		fmt.Printf("[DEBUG] Environment for %s %s:\n", dep.Name, dep.Version)
		for _, e := range env {
			if strings.HasPrefix(e, "PATH=") {
				fmt.Printf("  %s\n", e)
			}
		}
	}

	// Add libtool m4 macros to ACLOCAL_PATH for autoreconf
	// Check if using system or built libtool
	var aclocalPath []string

	libtoolResult := domain.CheckSystemDependency("libtool", phpVersion)
	if !libtoolResult.CanUse {
		// Use built libtool
		libtoolShare := filepath.Join(s.GetDependencyInstallDir(phpVersion, "libtool"), "share", "aclocal")
		if _, err := os.Stat(libtoolShare); err == nil {
			aclocalPath = append(aclocalPath, libtoolShare)
		}
	}

	automakeResult := domain.CheckSystemDependency("automake", phpVersion)
	if !automakeResult.CanUse {
		// Use built automake
		automakeShare := filepath.Join(s.GetDependencyInstallDir(phpVersion, "automake"), "share", "aclocal")
		if _, err := os.Stat(automakeShare); err == nil {
			aclocalPath = append(aclocalPath, automakeShare)
		}
	}
	if len(aclocalPath) > 0 {
		currentAclocal := getEnvValue(env, "ACLOCAL_PATH")
		if currentAclocal != "" {
			aclocalPath = append(aclocalPath, currentAclocal)
		}
		env = setOrReplaceEnv(env, "ACLOCAL_PATH", strings.Join(aclocalPath, ":"))
	}

	// Add dependency paths for transitive dependencies
	var pkgConfigPath []string
	var ldflags []string
	var cppflags []string
	var cflags []string

	for _, depName := range dep.Dependencies {
		depInstallDir := s.GetDependencyInstallDir(phpVersion, depName)
		pkgConfigPath = append(pkgConfigPath, filepath.Join(depInstallDir, "lib", "pkgconfig"))
		ldflags = append(ldflags, fmt.Sprintf("-L%s/lib", depInstallDir))
		cppflags = append(cppflags, fmt.Sprintf("-I%s/include", depInstallDir))

		// Add bin directory to PATH for transitive dependencies
		depBinDir := filepath.Join(depInstallDir, "bin")
		if stat, err := os.Stat(depBinDir); err == nil && stat.IsDir() {
			env = setOrReplaceEnv(env, "PATH", depBinDir+":"+getEnvValue(env, "PATH"))
		}
	}

	cflags, cppflags, ldflags = s.applyToolchainFlags(cflags, cppflags, ldflags)

	if len(pkgConfigPath) > 0 {
		env = setOrReplaceEnv(env, "PKG_CONFIG_PATH", strings.Join(pkgConfigPath, ":"))
	}
	if len(ldflags) > 0 {
		env = setOrReplaceEnv(env, "LDFLAGS", strings.Join(ldflags, " "))
	}
	if len(cppflags) > 0 {
		env = setOrReplaceEnv(env, "CPPFLAGS", strings.Join(cppflags, " "))
	}
	if len(cflags) > 0 {
		env = setOrReplaceEnv(env, "CFLAGS", strings.Join(cflags, " "))
	}

	return env
}

var essentialVars = []string{"HOME", "USER", "TMPDIR", "TMP", "TERM", "SHELL", "SSH_AUTH_SOCK"}
var localeVars = []string{"LANG", "LC_ALL", "LC_CTYPE", "LC_MESSAGES"}
var toolchainEnvVars = []string{
	"PHPV_TOOLCHAIN_CC", "PHPV_TOOLCHAIN_CXX", "PHPV_TOOLCHAIN_SYSROOT",
	"PHPV_TOOLCHAIN_PATH", "PHPV_TOOLCHAIN_CFLAGS", "PHPV_TOOLCHAIN_CPPFLAGS",
	"PHPV_TOOLCHAIN_LDFLAGS",
}

func (s *Service) getCleanBaseEnv() []string {
	env := []string{}

	for _, key := range essentialVars {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	for _, key := range localeVars {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	for _, key := range toolchainEnvVars {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	env = append(env, "PATH=/usr/bin:/bin")

	if os.Getenv("PHPV_DEBUG") == "1" {
		fmt.Printf("[DEBUG] Base environment:\n")
		for _, e := range env {
			if strings.HasPrefix(e, "PATH=") {
				fmt.Printf("  %s\n", e)
			}
		}
	}

	return env
}

func (s *Service) getAutoreconfPath(phpVersion domain.Version) string {
	// First check if we should use system autoconf
	if !domain.ShouldUseLLVMToolchain(phpVersion) {
		result := domain.CheckSystemDependency("autoconf", phpVersion)
		if result.CanUse {
			// Check system autoconf first
			if _, err := exec.LookPath("autoconf"); err == nil {
				return "autoconf"
			}
			if _, err := exec.LookPath("autoreconf"); err == nil {
				return "autoreconf"
			}
		}
	}

	// Fall back to built autoconf
	autoconfPath := filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin", "autoreconf")
	if _, err := os.Stat(autoconfPath); err == nil {
		return autoconfPath
	}

	// Try autoconf itself
	autoconfPath = filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin", "autoconf")
	if _, err := os.Stat(autoconfPath); err == nil {
		return autoconfPath
	}

	return ""
}

func (s *Service) getDependencyBinPath(phpVersion domain.Version) string {
	var bins []string
	useSystem := !domain.ShouldUseLLVMToolchain(phpVersion)

	// Check which tools are available on the system
	if useSystem {
		systemTools := []string{"autoconf", "automake", "libtool", "m4", "perl", "cmake"}
		for _, tool := range systemTools {
			result := domain.CheckSystemDependency(tool, phpVersion)
			if result.CanUse {
				// Tool is available on system, no need to add built path
				continue
			}
			// Tool not available or too old, add built path
			binPath := filepath.Join(s.GetDependenciesDir(phpVersion), tool, "bin")
			if _, err := os.Stat(binPath); err == nil {
				bins = append(bins, binPath)
			}
		}
	} else {
		// Using LLVM, add all built tool paths
		deps := []string{"autoconf", "automake", "libtool", "m4", "perl", "cmake"}
		for _, dep := range deps {
			binPath := filepath.Join(s.GetDependenciesDir(phpVersion), dep, "bin")
			if _, err := os.Stat(binPath); err == nil {
				bins = append(bins, binPath)
			}
		}
	}

	return strings.Join(bins, ":")
}

func (s *Service) applyCompilerEnv(env []string) []string {
	// Check if we should use Zig, LLVM or system GCC
	useZig := s.phpVersion != nil && domain.ShouldUseZigToolchain(*s.phpVersion)
	useLLVM := s.phpVersion != nil && domain.ShouldUseLLVMToolchain(*s.phpVersion)

	// Use custom toolchain if provided
	if s.toolchain != nil && s.toolchain.CC != "" {
		env = setOrReplaceEnv(env, "CC", s.toolchain.CC)
		if s.toolchain.CXX != "" {
			env = setOrReplaceEnv(env, "CXX", s.toolchain.CXX)
		}
		env = s.applyToolchainPath(env)
		if s.toolchain.Sysroot != "" {
			env = setOrReplaceEnv(env, "PKG_CONFIG_SYSROOT_DIR", s.toolchain.Sysroot)
		}
		return env
	}

	// Use system GCC if not using LLVM or Zig
	if !useLLVM && !useZig {
		env = setOrReplaceEnv(env, "CC", "gcc")
		env = setOrReplaceEnv(env, "CXX", "g++")
		// Use system ar, ranlib, nm
		env = setOrReplaceEnv(env, "AR", "ar")
		env = setOrReplaceEnv(env, "RANLIB", "ranlib")
		env = setOrReplaceEnv(env, "NM", "nm")
		env = setOrReplaceEnv(env, "LD", "ld")
		return env
	}

	// Use Zig compiler if requested
	if useZig {
		zigInstallDir := s.toolchainService.GetZigInstallDir()
		zigBinDir := filepath.Join(zigInstallDir, "bin")

		cc := fmt.Sprintf("zig cc -target x86_64-linux-gnu")
		cxx := fmt.Sprintf("zig c++ -target x86_64-linux-gnu")

		env = setOrReplaceEnv(env, "CC", cc)
		env = setOrReplaceEnv(env, "CXX", cxx)
		env = setOrReplaceEnv(env, "AR", "zig ar")
		env = setOrReplaceEnv(env, "RANLIB", "zig ranlib")
		env = setOrReplaceEnv(env, "NM", "zig nm")

		// Add Zig bin directory to PATH
		currentPath := getEnvValue(env, "PATH")
		if currentPath != "" {
			env = setOrReplaceEnv(env, "PATH", zigBinDir+":"+currentPath)
		} else {
			env = setOrReplaceEnv(env, "PATH", zigBinDir)
		}

		// Add target flag to CFLAGS and CXXFLAGS
		currentCflags := getEnvValue(env, "CFLAGS")
		env = setOrReplaceEnv(env, "CFLAGS", "-target x86_64-linux-gnu "+currentCflags)
		currentCppflags := getEnvValue(env, "CPPFLAGS")
		env = setOrReplaceEnv(env, "CPPFLAGS", "-target x86_64-linux-gnu "+currentCppflags)
		currentLdflags := getEnvValue(env, "LDFLAGS")
		env = setOrReplaceEnv(env, "LDFLAGS", "-target x86_64-linux-gnu "+currentLdflags)

		return env
	}

	// Use version-specific LLVM toolchain if we have a PHP version
	if s.phpVersion != nil {
		llvmConfig := domain.GetLLVMVersionForPHP(*s.phpVersion)
		llvmInstallDir := s.toolchainService.GetLLVMInstallDir(llvmConfig.Version)
		llvmBinDir := filepath.Join(llvmInstallDir, "bin")

		cc := filepath.Join(llvmBinDir, "clang")
		cxx := filepath.Join(llvmBinDir, "clang++")
		ar := filepath.Join(llvmBinDir, "llvm-ar")
		ranlib := filepath.Join(llvmBinDir, "llvm-ranlib")
		nm := filepath.Join(llvmBinDir, "llvm-nm")
		ld := filepath.Join(llvmBinDir, "ld.lld")

		env = setOrReplaceEnv(env, "CC", cc)
		env = setOrReplaceEnv(env, "CXX", cxx)
		env = setOrReplaceEnv(env, "AR", ar)
		env = setOrReplaceEnv(env, "RANLIB", ranlib)
		env = setOrReplaceEnv(env, "NM", nm)
		env = setOrReplaceEnv(env, "LD", ld)

		// Add LLVM bin directory to PATH
		currentPath := getEnvValue(env, "PATH")
		if currentPath != "" {
			env = setOrReplaceEnv(env, "PATH", llvmBinDir+":"+currentPath)
		} else {
			env = setOrReplaceEnv(env, "PATH", llvmBinDir)
		}
		return env
	}

	// Fallback to system clang (shouldn't happen in normal usage)
	env = setOrReplaceEnv(env, "CC", "clang")
	env = setOrReplaceEnv(env, "CXX", "clang++")
	return env
}

func (s *Service) applyToolchainPath(env []string) []string {
	if s.toolchain == nil || len(s.toolchain.Path) == 0 {
		return env
	}
	var cleaned []string
	for _, segment := range s.toolchain.Path {
		segment = strings.TrimSpace(segment)
		if segment != "" {
			cleaned = append(cleaned, segment)
		}
	}
	if len(cleaned) == 0 {
		return env
	}
	current := getEnvValue(env, "PATH")
	if current != "" {
		cleaned = append(cleaned, current)
	}
	return setOrReplaceEnv(env, "PATH", strings.Join(cleaned, string(os.PathListSeparator)))
}

func (s *Service) applyToolchainFlags(cflags, cppflags, ldflags []string) ([]string, []string, []string) {
	if s.toolchain == nil {
		return cflags, cppflags, ldflags
	}
	if s.toolchain.Sysroot != "" {
		sysrootFlag := fmt.Sprintf("--sysroot=%s", s.toolchain.Sysroot)
		cflags = append(cflags, sysrootFlag)
		cppflags = append(cppflags, sysrootFlag)
		ldflags = append(ldflags, sysrootFlag)
	}
	cflags = append(cflags, s.toolchain.CFlags...)
	cppflags = append(cppflags, s.toolchain.CPPFlags...)
	ldflags = append(ldflags, s.toolchain.LDFlags...)
	return cflags, cppflags, ldflags
}

func setOrReplaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func getEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

// getCachedArchivePath returns the path where a dependency archive should be cached
func (s *Service) getCachedArchivePath(url string) string {
	// Extract filename from URL
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	return filepath.Join(s.GetCacheDir(), filename)
}

// downloadAndExtract downloads and extracts a tarball using cache
func (s *Service) downloadAndExtract(ctx context.Context, url, destDir string) error {
	cachePath := s.getCachedArchivePath(url)

	// Check if already cached
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		// Download to cache
		if err := s.downloadToCache(ctx, url, cachePath); err != nil {
			return fmt.Errorf("failed to download: %w", err)
		}
		fmt.Printf("Downloaded and cached: %s\n", filepath.Base(cachePath))
	} else {
		fmt.Printf("Using cached archive: %s\n", filepath.Base(cachePath))
	}

	// Extract from cache
	return s.extractFromCache(cachePath, destDir)
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

// extractFromCache extracts an archive from the cache
func (s *Service) extractFromCache(cachePath, destDir string) error {
	file, err := os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cached file: %w", err)
	}
	defer file.Close()

	// Determine if it's gzip or xz based on filename
	if strings.HasSuffix(cachePath, ".tar.xz") {
		return s.extractTarXz(file, destDir)
	}
	return s.extractTarGz(file, destDir)
}

// extractTarGz extracts a tar.gz archive
func (s *Service) extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return s.extractTar(tar.NewReader(gzr), destDir)
}

// extractTarXz extracts a tar.xz archive using external xz command
func (s *Service) extractTarXz(r io.Reader, destDir string) error {
	// Save to temp file first
	tmpFile, err := os.CreateTemp("", "dep-*.tar.xz")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, r); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Use xz command to decompress
	cmd := exec.Command("xz", "-dc", tmpPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	err = s.extractTar(tar.NewReader(stdout), destDir)
	cmd.Wait()
	return err
}

// extractTar extracts a tar archive
func (s *Service) extractTar(tr *tar.Reader, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip the first directory component
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}

		topLevel := parts[0]
		target := filepath.Join(destDir, parts[1])

		atime := header.AccessTime
		if atime.IsZero() {
			atime = header.ModTime
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
			_ = os.Chtimes(target, atime, header.ModTime)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Chmod(os.FileMode(header.Mode)); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			_ = os.Chtimes(target, atime, header.ModTime)
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				if !os.IsExist(err) {
					return err
				}
			}
		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			linkParts := strings.SplitN(header.Linkname, "/", 2)
			var linkRel string
			if len(linkParts) == 1 {
				linkRel = linkParts[0]
			} else if linkParts[0] == topLevel {
				linkRel = linkParts[1]
			} else {
				linkRel = header.Linkname
			}
			linkTarget := filepath.Join(destDir, linkRel)
			if err := os.Link(linkTarget, target); err != nil {
				if !os.IsExist(err) {
					return err
				}
			}
		}
	}

	return nil
}

// GetPHPConfigureFlags returns configure flags for PHP to use built dependencies
func (s *Service) GetPHPConfigureFlags(phpVersion domain.Version) []string {
	depsDir := s.GetDependenciesDir(phpVersion)

	var flags []string

	// Check if we should use system dependencies (for PHP 8.3+)
	useSystemDeps := domain.ShouldUseLLVMToolchain(phpVersion) == false

	deps := GetDependenciesForVersion(phpVersion)
	for _, dep := range deps {
		// Skip LLVM - it's a toolchain, not a PHP dependency
		if dep.Name == "llvm" {
			continue
		}

		// If using system deps, check if this dep is available on the system
		if useSystemDeps {
			if s.isSystemDependencyAvailable(dep.Name) {
				// Use system library
				flags = append(flags, s.getSystemDepConfigureFlag(dep.Name, phpVersion)...)
				continue
			}
		}

		depDir := filepath.Join(depsDir, dep.Name)

		// PHP 4.x has different flag names
		isPHP4 := phpVersion.Major == 4
		// PHP 7.0-7.3 uses different flag names than PHP 7.4+ and PHP 8.x
		isPHP7Old := phpVersion.Major == 7 && phpVersion.Minor < 4

		switch dep.Name {
		case "libxml2":
			if isPHP4 || isPHP7Old {
				// PHP 4 and PHP 7.0-7.3 uses --with-libxml-dir
				flags = append(flags, fmt.Sprintf("--with-libxml-dir=%s", depDir))
			} else {
				// PHP 7.4+ and PHP 8.x use --with-libxml
				flags = append(flags, fmt.Sprintf("--with-libxml=%s", depDir))
			}
		case "openssl":
			// PHP 4's OpenSSL extension is incompatible with modern OpenSSL
			// Skip adding OpenSSL support for PHP 4 and PHP 5.1/5.0
			if isPHP4 || (phpVersion.Major == 5 && phpVersion.Minor <= 1) {
				continue
			}
			if isPHP7Old {
				// PHP 7.0-7.3 uses --with-openssl-dir
				flags = append(flags, fmt.Sprintf("--with-openssl-dir=%s", depDir))
			} else {
				// PHP 7.4+ and PHP 8.x use --with-openssl
				flags = append(flags, fmt.Sprintf("--with-openssl=%s", depDir))
			}
		case "curl":
			// PHP 5.1/5.0 and PHP 4 Curl extension is incompatible with modern libcurl
			// Skip adding Curl support for PHP 5.1/5.0 and PHP 4
			if isPHP4 || (phpVersion.Major == 5 && phpVersion.Minor <= 1) {
				continue
			}
			// All other versions use --with-curl
			flags = append(flags, fmt.Sprintf("--with-curl=%s", depDir))
		case "zlib":
			if isPHP4 || isPHP7Old {
				// PHP 4 and PHP 7.0-7.3 uses --with-zlib-dir
				flags = append(flags, fmt.Sprintf("--with-zlib-dir=%s", depDir))
			} else {
				// PHP 7.4+ and PHP 8.x use --with-zlib
				flags = append(flags, fmt.Sprintf("--with-zlib=%s", depDir))
			}
		case "oniguruma":
			// PHP 4 doesn't have oniguruma extension, skip it
			if isPHP4 {
				continue
			}
			// PHP 5+ uses --with-onig
			flags = append(flags, fmt.Sprintf("--with-onig=%s", depDir))
		}
	}

	return flags
}

// getSystemDepConfigureFlag returns configure flags for system dependencies
func (s *Service) getSystemDepConfigureFlag(depName string, phpVersion domain.Version) []string {
	isPHP7Old := phpVersion.Major == 7 && phpVersion.Minor < 4
	isPHP8Plus := phpVersion.Major >= 8

	switch depName {
	case "zlib":
		if isPHP7Old {
			return []string{"--with-zlib-dir=/usr"}
		}
		return []string{"--with-zlib"}
	case "libxml2":
		if isPHP7Old {
			return []string{"--with-libxml-dir=/usr"}
		}
		return []string{"--with-libxml"}
	case "openssl":
		if isPHP7Old {
			return []string{"--with-openssl-dir=/usr"}
		}
		if isPHP8Plus {
			return []string{"--with-openssl"}
		}
		return []string{"--with-openssl"}
	case "curl":
		return []string{"--with-curl"}
	case "oniguruma":
		// PHP 8.5+ removed the --with-oniguruma option (built-in)
		if phpVersion.Major >= 8 && phpVersion.Minor >= 5 {
			return []string{}
		}
		if isPHP8Plus {
			return []string{"--with-oniguruma"}
		}
		return []string{"--with-onig"}
	case "libzip":
		return []string{"--with-libzip"}
	case "brotli":
		return []string{"--with-brotli"}
	case "zstd":
		return []string{"--with-zstd"}
	case "libedit":
		return []string{"--with-libedit"}
	case "libxslt":
		return []string{"--with-xsl"}
	case "libgd":
		return []string{"--with-gd"}
	case "freetype":
		return []string{"--with-freetype"}
	case "jpeg":
		return []string{"--with-jpeg"}
	case "png":
		return []string{"--with-png"}
	case "webp":
		return []string{"--with-webp"}
	case "avif":
		return []string{"--with-avif"}
	}
	return []string{}
}

// GetPHPEnvironment returns environment variables for PHP build
func (s *Service) GetPHPEnvironment(phpVersion domain.Version) []string {
	// Store the PHP version to ensure we use the correct LLVM toolchain
	s.phpVersion = &phpVersion

	env := s.getCleanBaseEnv()
	depsDir := s.GetDependenciesDir(phpVersion)

	// Check if we should use system dependencies (for PHP 8.3+)
	useSystemDeps := domain.ShouldUseLLVMToolchain(phpVersion) == false

	var pkgConfigPath []string
	var ldflags []string
	var cppflags []string
	var cflags []string

	deps := GetDependenciesForVersion(phpVersion)
	for _, dep := range deps {
		// Skip LLVM - it's a toolchain, not a PHP dependency
		if dep.Name == "llvm" {
			continue
		}

		// For system dependencies, add system pkg-config path
		if useSystemDeps && s.isSystemDependencyAvailable(dep.Name) {
			// Get system library path via pkg-config
			libPath, incPath, _, found := s.GetSystemDependencyPath(dep.Name)
			if found {
				// Add system pkg-config path
				pkgConfigPath = append(pkgConfigPath, "/usr/lib/pkgconfig", "/usr/local/lib/pkgconfig")
				// Add system library path
				if libPath != "" {
					// Extract just the -L path from pkg-config output
					if !strings.Contains(libPath, "/") {
						// It's just -L without path, use default
						ldflags = append(ldflags, "-L/usr/lib", "-L/usr/local/lib")
					}
				}
				// Add system include path
				if incPath != "" {
					cppflags = append(cppflags, fmt.Sprintf("-I%s", incPath))
				} else {
					// Use default include paths
					switch dep.Name {
					case "libxml2":
						cppflags = append(cppflags, "-I/usr/include/libxml2")
					case "openssl":
						cppflags = append(cppflags, "-I/usr/include/openssl")
					case "curl":
						cppflags = append(cppflags, "-I/usr/include")
					case "zlib":
						cppflags = append(cppflags, "-I/usr/include")
					case "oniguruma":
						cppflags = append(cppflags, "-I/usr/include")
					}
				}
			}
			continue
		}

		depDir := filepath.Join(depsDir, dep.Name)
		pkgConfigPath = append(pkgConfigPath, filepath.Join(depDir, "lib", "pkgconfig"))
		ldflags = append(ldflags, fmt.Sprintf("-L%s/lib", depDir))
		cppflags = append(cppflags, fmt.Sprintf("-I%s/include", depDir))
	}

	env = s.applyCompilerEnv(env)

	// Add version-specific CFLAGS
	if phpVersion.Major == 7 && phpVersion.Minor == 2 {
		// PHP 7.2 needs specific feature test macros defined and to suppress deprecated declarations warnings
		cflags = append(cflags, "-D_GNU_SOURCE")
		cflags = append(cflags, "-D_DEFAULT_SOURCE")
		cflags = append(cflags, "-Wno-deprecated-declarations")
		// Fix for readdir_r and stream cast errors on modern systems
		cflags = append(cflags, "-D_LARGEFILE_SOURCE")
		cflags = append(cflags, "-D_FILE_OFFSET_BITS=64")
		cflags = append(cflags, "-D_POSIX_C_SOURCE=200809L")
	}

	// PHP 4.x needs special flags to handle multiple scanner definitions
	if phpVersion.Major == 4 {
		// Fix for yytext multiple definition error between zend_language_scanner and zend_ini_scanner
		// The pre-generated scanner files have conflicting global yytext variables
		// -fcommon allows multiple definitions (like older GCC behavior)
		cflags = append(cflags, "-fcommon")
		// Suppress warnings about implicit function declarations (PHP 4 uses old-style declarations)
		cflags = append(cflags, "-Wno-implicit-function-declaration")
		// Fix for LONG_MAX comparison issues in zend_operators.c
		cflags = append(cflags, "-Wno-implicit-const-int-float-conversion")
		// Fix for abs() function with long argument
		cflags = append(cflags, "-Wno-absolute-value")
	}

	cflags, cppflags, ldflags = s.applyToolchainFlags(cflags, cppflags, ldflags)

	if len(pkgConfigPath) > 0 {
		env = setOrReplaceEnv(env, "PKG_CONFIG_PATH", strings.Join(pkgConfigPath, ":"))
	}
	if len(ldflags) > 0 {
		env = setOrReplaceEnv(env, "LDFLAGS", strings.Join(ldflags, " "))
	}
	if len(cppflags) > 0 {
		env = setOrReplaceEnv(env, "CPPFLAGS", strings.Join(cppflags, " "))
	}
	if len(cflags) > 0 {
		env = setOrReplaceEnv(env, "CFLAGS", strings.Join(cflags, " "))
	}

	// Add re2c to PATH if available
	re2cBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "re2c"), "bin")
	if _, err := os.Stat(filepath.Join(re2cBin, "re2c")); err == nil {
		env = setOrReplaceEnv(env, "PATH", re2cBin+":"+getEnvValue(env, "PATH"))
	}

	// Add autoconf to PATH if available
	autoconfBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "autoconf"), "bin")
	if _, err := os.Stat(filepath.Join(autoconfBin, "autoconf")); err == nil {
		env = setOrReplaceEnv(env, "PATH", autoconfBin+":"+getEnvValue(env, "PATH"))
	}

	// Add automake to PATH if available
	automakeBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "automake"), "bin")
	if _, err := os.Stat(filepath.Join(automakeBin, "automake")); err == nil {
		env = setOrReplaceEnv(env, "PATH", automakeBin+":"+getEnvValue(env, "PATH"))
	}

	// Add libtool to PATH if available
	libtoolBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "libtool"), "bin")
	if _, err := os.Stat(filepath.Join(libtoolBin, "libtool")); err == nil {
		env = setOrReplaceEnv(env, "PATH", libtoolBin+":"+getEnvValue(env, "PATH"))
	}

	// Add m4 to PATH if available (bison depends on it)
	m4Bin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "m4"), "bin")
	if _, err := os.Stat(filepath.Join(m4Bin, "m4")); err == nil {
		env = setOrReplaceEnv(env, "PATH", m4Bin+":"+getEnvValue(env, "PATH"))
	}

	// For PHP 5+, add bison and flex to PATH for parser regeneration
	// For PHP 4.x, skip these to use pre-generated scanner files
	if phpVersion.Major >= 5 {
		bisonBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "bison"), "bin")
		if _, err := os.Stat(filepath.Join(bisonBin, "bison")); err == nil {
			env = setOrReplaceEnv(env, "PATH", bisonBin+":"+getEnvValue(env, "PATH"))
		}

		flexBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "flex"), "bin")
		if _, err := os.Stat(filepath.Join(flexBin, "flex")); err == nil {
			env = setOrReplaceEnv(env, "PATH", flexBin+":"+getEnvValue(env, "PATH"))
		}
	}

	if os.Getenv("PHPV_DEBUG") == "1" {
		fmt.Printf("[DEBUG] PHP environment:\n")
		for _, e := range env {
			if strings.HasPrefix(e, "PATH=") {
				fmt.Printf("  %s\n", e)
			}
		}
	}

	return env
}

func (s *Service) Clean(phpVersion domain.Version, depName string) error {
	versionStr := fmt.Sprintf("%d.%d.%d", phpVersion.Major, phpVersion.Minor, phpVersion.Patch)

	if depName == "" {
		depsDir := filepath.Join(s.phpvRoot, "dependencies", versionStr)
		if _, err := os.Stat(depsDir); err == nil {
			if err := os.RemoveAll(depsDir); err != nil {
				return fmt.Errorf("failed to remove dependencies directory: %w", err)
			}
			fmt.Printf("Removed dependencies directory: %s\n", depsDir)
		}

		depsSrcDir := filepath.Join(s.phpvRoot, "dependencies-src", versionStr)
		if _, err := os.Stat(depsSrcDir); err == nil {
			if err := os.RemoveAll(depsSrcDir); err != nil {
				return fmt.Errorf("failed to remove dependencies-src directory: %w", err)
			}
			fmt.Printf("Removed dependencies-src directory: %s\n", depsSrcDir)
		}

		fmt.Printf("✓ Cleaned all dependencies for PHP %s\n", versionStr)
		return nil
	}

	installDir := filepath.Join(s.phpvRoot, "dependencies", versionStr, depName)
	if _, err := os.Stat(installDir); err == nil {
		if err := os.RemoveAll(installDir); err != nil {
			return fmt.Errorf("failed to remove dependency %s: %w", depName, err)
		}
		fmt.Printf("Removed: %s\n", installDir)
	}

	depsSrcDir := filepath.Join(s.phpvRoot, "dependencies-src", versionStr)
	if _, err := os.Stat(depsSrcDir); err == nil {
		entries, err := os.ReadDir(depsSrcDir)
		if err == nil {
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), depName+"-") {
					srcPath := filepath.Join(depsSrcDir, entry.Name())
					if err := os.RemoveAll(srcPath); err != nil {
						return fmt.Errorf("failed to remove dependency source %s: %w", entry.Name(), err)
					}
					fmt.Printf("Removed: %s\n", srcPath)
				}
			}
		}
	}

	fmt.Printf("✓ Cleaned dependency '%s' for PHP %s\n", depName, versionStr)
	return nil
}
