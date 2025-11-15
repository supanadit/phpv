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
}

func NewService(phpvRoot string) *Service {
	return &Service{
		httpClient: &http.Client{},
		phpvRoot:   phpvRoot,
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
		// Remove autotools-generated files that cause regeneration issues
		filesToRemove := []string{
			"Makefile",
			"Makefile.in",
			"config.status",
			"config.log",
			"config.h",
			"config.h.in",
			"configure",
			"aclocal.m4",
			"autom4te.cache",
			"libtool",
			"stamp-h1",
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
		if _, err := os.Stat(autogenPath); err == nil {
			fmt.Printf("Running autogen.sh to regenerate configure script...\n")
			if err := util.RunCommand(ctx, sourceDir, env, "./autogen.sh"); err != nil {
				return fmt.Errorf("autogen.sh failed for %s: %w", dep.Name, err)
			}
		}
	}

	// Configure
	configureCmd := "./configure"
	configureArgs := append([]string{fmt.Sprintf("--prefix=%s", installDir)}, dep.ConfigureFlags...)

	// Special handling for OpenSSL which uses ./config
	if len(dep.BuildCommands) > 0 && strings.Contains(dep.BuildCommands[0], "config") {
		configureCmd = dep.BuildCommands[0]
		configureArgs = append([]string{fmt.Sprintf("--prefix=%s", installDir)}, dep.ConfigureFlags...)
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
	env = append(env, "CC=clang", "CXX=clang++")

	// Add dependency paths for transitive dependencies
	var pkgConfigPath []string
	var ldflags []string
	var cppflags []string

	for _, depName := range dep.Dependencies {
		depInstallDir := s.GetDependencyInstallDir(phpVersion, depName)
		pkgConfigPath = append(pkgConfigPath, filepath.Join(depInstallDir, "lib", "pkgconfig"))
		ldflags = append(ldflags, fmt.Sprintf("-L%s/lib", depInstallDir))
		cppflags = append(cppflags, fmt.Sprintf("-I%s/include", depInstallDir))
	}

	if len(pkgConfigPath) > 0 {
		env = append(env, "PKG_CONFIG_PATH="+strings.Join(pkgConfigPath, ":"))
	}
	if len(ldflags) > 0 {
		env = append(env, "LDFLAGS="+strings.Join(ldflags, " "))
	}
	if len(cppflags) > 0 {
		env = append(env, "CPPFLAGS="+strings.Join(cppflags, " "))
	}

	return env
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
		switch dep.Name {
		case "libxml2":
			flags = append(flags, fmt.Sprintf("--with-libxml=%s", depDir))
		case "openssl":
			flags = append(flags, fmt.Sprintf("--with-openssl=%s", depDir))
		case "curl":
			flags = append(flags, fmt.Sprintf("--with-curl=%s", depDir))
		case "zlib":
			flags = append(flags, fmt.Sprintf("--with-zlib=%s", depDir))
		case "oniguruma":
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

	deps := GetDependenciesForVersion(phpVersion)
	for _, dep := range deps {
		depDir := filepath.Join(depsDir, dep.Name)
		pkgConfigPath = append(pkgConfigPath, filepath.Join(depDir, "lib", "pkgconfig"))
		ldflags = append(ldflags, fmt.Sprintf("-L%s/lib", depDir))
		cppflags = append(cppflags, fmt.Sprintf("-I%s/include", depDir))
	}

	env = append(env, "CC=clang", "CXX=clang++")

	if len(pkgConfigPath) > 0 {
		env = append(env, "PKG_CONFIG_PATH="+strings.Join(pkgConfigPath, ":"))
	}
	if len(ldflags) > 0 {
		env = append(env, "LDFLAGS="+strings.Join(ldflags, " "))
	}
	if len(cppflags) > 0 {
		env = append(env, "CPPFLAGS="+strings.Join(cppflags, " "))
	}

	return env
}
