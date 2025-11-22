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
	"github.com/supanadit/phpv/internal/util"
)

type Service struct {
	httpClient *http.Client
	phpvRoot   string
	toolchain  *domain.ToolchainConfig
}

func NewService(phpvRoot string) *Service {
	return NewServiceWithToolchain(phpvRoot, nil)
}

// NewServiceWithToolchain allows providing an optional toolchain configuration.
func NewServiceWithToolchain(phpvRoot string, toolchain *domain.ToolchainConfig) *Service {
	return &Service{
		httpClient: &http.Client{},
		phpvRoot:   phpvRoot,
		toolchain:  toolchain,
	}
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

// IsDependencyBuilt checks if a dependency is already built
func (s *Service) IsDependencyBuilt(phpVersion domain.Version, dep domain.Dependency) bool {
	installDir := s.GetDependencyInstallDir(phpVersion, dep.Name)
	if dep.Name == "cmake" {
		// For cmake, check if bin/cmake exists
		binPath := filepath.Join(installDir, "bin", "cmake")
		if _, err := os.Stat(binPath); err == nil {
			return true
		}
		return false
	}
	// Check if lib directory exists with some files
	libDir := filepath.Join(installDir, "lib")
	if stat, err := os.Stat(libDir); err == nil && stat.IsDir() {
		// Check if there are any files in lib directory
		entries, err := os.ReadDir(libDir)
		return err == nil && len(entries) > 0
	}
	return false
}

// BuildDependencies builds all dependencies for a PHP version
func (s *Service) BuildDependencies(ctx context.Context, phpVersion domain.Version) error {
	deps := GetDependenciesForVersion(phpVersion)

	fmt.Printf("\n=== Building Dependencies for PHP %d.%d.%d ===\n\n",
		phpVersion.Major, phpVersion.Minor, phpVersion.Patch)

	// Build dependencies in order (respecting transitive dependencies)
	builtDeps := make(map[string]bool)

	for _, dep := range deps {
		if err := s.buildDependencyWithDeps(ctx, phpVersion, dep, deps, builtDeps); err != nil {
			return err
		}
	}

	fmt.Printf("\n✓ All dependencies built successfully\n\n")
	return nil
}

// buildDependencyWithDeps recursively builds a dependency and its dependencies
func (s *Service) buildDependencyWithDeps(ctx context.Context, phpVersion domain.Version, dep domain.Dependency, allDeps []domain.Dependency, built map[string]bool) error {
	// Skip if already built
	if built[dep.Name] {
		return nil
	}

	// Check if already installed
	if s.IsDependencyBuilt(phpVersion, dep) {
		fmt.Printf("→ %s %s already built, skipping\n", dep.Name, dep.Version)
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
	sourceDir := filepath.Join(s.phpvRoot, "dependencies-src", dep.Name+"-"+dep.Version)

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
		}

		// Only remove autotools-generated files for autotools-based projects
		// For zlib (uses CMake), don't remove configure-related files
		if dep.Name != "zlib" {
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
		// Check for autogen.sh first
		if _, err := os.Stat(autogenPath); err == nil {
			fmt.Printf("Running autogen.sh to regenerate configure script...\n")
			if err := util.RunCommand(ctx, sourceDir, env, "./autogen.sh"); err != nil {
				return fmt.Errorf("autogen.sh failed for %s: %w", dep.Name, err)
			}
		} else {
			// Check for buildconf (used by curl) or configure.ac (used by most autotools projects)
			// Projects recommend using autoreconf -fi for configure regeneration
			buildconfPath := filepath.Join(sourceDir, "buildconf")
			configureAcPath := filepath.Join(sourceDir, "configure.ac")

			if _, err := os.Stat(buildconfPath); err == nil {
				fmt.Printf("Running autoreconf -fi to regenerate configure script...\n")
				if err := util.RunCommand(ctx, sourceDir, env, "autoreconf", "-fi"); err != nil {
					return fmt.Errorf("autoreconf failed for %s: %w", dep.Name, err)
				}
			} else if _, err := os.Stat(configureAcPath); err == nil {
				fmt.Printf("Running autoreconf -fi to regenerate configure script...\n")
				if err := util.RunCommand(ctx, sourceDir, env, "autoreconf", "-fi"); err != nil {
					return fmt.Errorf("autoreconf failed for %s: %w", dep.Name, err)
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
	env := os.Environ()
	env = s.applyCompilerEnv(env)

	// Add cmake to PATH if available
	cmakeBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "cmake"), "bin")
	if _, err := os.Stat(filepath.Join(cmakeBin, "cmake")); err == nil {
		env = setOrReplaceEnv(env, "PATH", cmakeBin+":"+getEnvValue(env, "PATH"))
	}

	// Add perl to PATH if available
	perlBin := filepath.Join(s.GetDependencyInstallDir(phpVersion, "perl"), "bin")
	if _, err := os.Stat(filepath.Join(perlBin, "perl")); err == nil {
		env = setOrReplaceEnv(env, "PATH", perlBin+":"+getEnvValue(env, "PATH"))
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

func (s *Service) applyCompilerEnv(env []string) []string {
	cc := "clang"
	cxx := "clang++"
	if s.toolchain != nil {
		if s.toolchain.CC != "" {
			cc = s.toolchain.CC
		}
		if s.toolchain.CXX != "" {
			cxx = s.toolchain.CXX
		}
	}

	env = setOrReplaceEnv(env, "CC", cc)
	env = setOrReplaceEnv(env, "CXX", cxx)

	if s.toolchain != nil {
		env = s.applyToolchainPath(env)
		if s.toolchain.Sysroot != "" {
			env = setOrReplaceEnv(env, "PKG_CONFIG_SYSROOT_DIR", s.toolchain.Sysroot)
		}
	}

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

// downloadAndExtract downloads and extracts a tarball
func (s *Service) downloadAndExtract(ctx context.Context, url, destDir string) error {
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

	// Determine if it's gzip or xz based on URL
	if strings.HasSuffix(url, ".tar.xz") {
		return s.extractTarXz(resp.Body, destDir)
	}
	return s.extractTarGz(resp.Body, destDir)
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

	deps := GetDependenciesForVersion(phpVersion)
	for _, dep := range deps {
		depDir := filepath.Join(depsDir, dep.Name)

		// Add specific flags for each dependency
		// PHP 7.0-7.3 uses different flag names than PHP 7.4+ and PHP 8.x
		isPHP7Old := phpVersion.Major == 7 && phpVersion.Minor < 4

		switch dep.Name {
		case "libxml2":
			if isPHP7Old {
				// PHP 7.0-7.3 uses --with-libxml-dir
				flags = append(flags, fmt.Sprintf("--with-libxml-dir=%s", depDir))
			} else {
				// PHP 7.4+ and PHP 8.x use --with-libxml
				flags = append(flags, fmt.Sprintf("--with-libxml=%s", depDir))
			}
		case "openssl":
			if isPHP7Old {
				// PHP 7.0-7.3 uses --with-openssl-dir
				flags = append(flags, fmt.Sprintf("--with-openssl-dir=%s", depDir))
			} else {
				// PHP 7.4+ and PHP 8.x use --with-openssl
				flags = append(flags, fmt.Sprintf("--with-openssl=%s", depDir))
			}
		case "curl":
			// All versions use --with-curl
			flags = append(flags, fmt.Sprintf("--with-curl=%s", depDir))
		case "zlib":
			if isPHP7Old {
				// PHP 7.0-7.3 uses --with-zlib-dir
				flags = append(flags, fmt.Sprintf("--with-zlib-dir=%s", depDir))
			} else {
				// PHP 7.4+ and PHP 8.x use --with-zlib
				flags = append(flags, fmt.Sprintf("--with-zlib=%s", depDir))
			}
		case "oniguruma":
			// All versions use --with-onig
			flags = append(flags, fmt.Sprintf("--with-onig=%s", depDir))
		}
	}

	return flags
}

// GetPHPEnvironment returns environment variables for PHP build
func (s *Service) GetPHPEnvironment(phpVersion domain.Version) []string {
	env := os.Environ()
	depsDir := s.GetDependenciesDir(phpVersion)

	var pkgConfigPath []string
	var ldflags []string
	var cppflags []string
	var cflags []string

	deps := GetDependenciesForVersion(phpVersion)
	for _, dep := range deps {
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
	// Add more version-specific flags here as needed
	// if phpVersion.Major == X && phpVersion.Minor == Y {
	//     cflags = append(cflags, "additional-flag")
	// }

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
