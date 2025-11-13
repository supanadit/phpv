package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
)

// PHPVersionRepository represent the PHP version's repository contract
type PHPVersionRepository interface {
	GetAvailableVersions(ctx context.Context) ([]domain.PHPVersion, error)
	GetVersionByString(ctx context.Context, version string) (domain.PHPVersion, error)
	SaveVersion(ctx context.Context, version domain.PHPVersion) error
}

// InstallationRepository represent the installation's repository contract
type InstallationRepository interface {
	GetAllInstallations(ctx context.Context) ([]domain.Installation, error)
	GetInstallationByVersion(ctx context.Context, version domain.PHPVersion) (domain.Installation, error)
	GetActiveInstallation(ctx context.Context) (domain.Installation, error)
	SaveInstallation(ctx context.Context, installation domain.Installation) error
	SetActiveInstallation(ctx context.Context, installation domain.Installation) error
	DeleteInstallation(ctx context.Context, version domain.PHPVersion) error
}

// Downloader represent the downloader contract for downloading PHP source
type Downloader interface {
	DownloadSource(ctx context.Context, version domain.PHPVersion, destPath string) error
}

// Builder represent the builder contract for building PHP from source
type Builder interface {
	Build(ctx context.Context, sourcePath string, installPath string, config map[string]string) error
	GetBuildStrategy() domain.BuildStrategy
}

// FileSystem represents the filesystem operations contract
type FileSystem interface {
	CreateDirectory(path string) error
	RemoveDirectory(path string) error
	FileExists(path string) bool
	DirectoryExists(path string) bool
}

// DockerBuilder builds PHP using Docker containers
type DockerBuilder struct {
	baseBuilder Builder
	dockerImage string
}

// NewDockerBuilder creates a new Docker-based builder
func NewDockerBuilder(baseBuilder Builder, dockerImage string) *DockerBuilder {
	return &DockerBuilder{
		baseBuilder: baseBuilder,
		dockerImage: dockerImage,
	}
}

// Build builds PHP using Docker
func (b *DockerBuilder) Build(ctx context.Context, sourcePath string, installPath string, config map[string]string) error {
	fmt.Printf("🐳 Building PHP in Docker container (%s)...\n", b.dockerImage)

	// Create a temporary directory for the build context
	buildContext := fmt.Sprintf("/tmp/phpv-build-%d", time.Now().Unix())
	if err := os.MkdirAll(buildContext, 0755); err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer os.RemoveAll(buildContext)

	// Create a Dockerfile
	dockerfile := `FROM ` + b.dockerImage + `

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    wget \
    tar \
    gzip \
    libxml2-dev \
    libssl-dev \
    libcurl4-openssl-dev \
    libpng-dev \
    libjpeg-dev \
    libfreetype6-dev \
    libbz2-dev \
    libreadline-dev \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy source code
COPY . /build/source

# Configure and build
RUN cd /build/source && \
    ./configure --prefix=/build/install --enable-shared=no --enable-static=yes --disable-all --enable-cli && \
    make -j$(nproc) && \
    make install

# Copy built binaries back to host
CMD ["cp", "-r", "/build/install", "/output"]
`

	if err := os.WriteFile(filepath.Join(buildContext, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to create Dockerfile: %w", err)
	}

	// Copy source code to build context
	if err := b.copySourceToContext(sourcePath, buildContext); err != nil {
		return fmt.Errorf("failed to copy source: %w", err)
	}

	// Build Docker image
	imageName := fmt.Sprintf("phpv-build-%d", time.Now().Unix())
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", imageName, buildContext)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	fmt.Printf("🏗️  Building Docker image %s...\n", imageName)
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(installPath, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Run container to extract built binaries
	runCmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-v", fmt.Sprintf("%s:/output", installPath), imageName)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	fmt.Printf("📦 Extracting built PHP to %s...\n", installPath)
	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("failed to run Docker container: %w", err)
	}

	fmt.Printf("✅ Successfully built PHP using Docker!\n")
	return nil
}

// copySourceToContext copies the PHP source code to the Docker build context
func (b *DockerBuilder) copySourceToContext(sourcePath string, buildContext string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source path: %w", err)
	}

	if sourceInfo.IsDir() {
		// Copy directory
		return b.copyDir(sourcePath, filepath.Join(buildContext, "source"))
	} else {
		// Copy file (though this shouldn't happen for PHP source)
		return b.copyFile(sourcePath, filepath.Join(buildContext, "source"))
	}
}

// copyDir recursively copies a directory
func (b *DockerBuilder) copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		} else {
			return b.copyFile(path, targetPath)
		}
	})
}

// copyFile copies a single file
func (b *DockerBuilder) copyFile(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// GetBuildStrategy returns the build strategy
func (b *DockerBuilder) GetBuildStrategy() domain.BuildStrategy {
	return domain.BuildStrategyDocker
}

// NativeGCCBuilder builds PHP with specific GCC versions and dependencies on the host machine
type NativeGCCBuilder struct {
	baseBuilder   Builder
	gccVersion    string
	phpVersion    domain.PHPVersion
	dependencyMgr *BuildDependencyManager
}

// NewNativeGCCBuilder creates a new native GCC-based builder
func NewNativeGCCBuilder(baseBuilder Builder, gccVersion string, phpVersion domain.PHPVersion) *NativeGCCBuilder {
	return &NativeGCCBuilder{
		baseBuilder:   baseBuilder,
		gccVersion:    gccVersion,
		phpVersion:    phpVersion,
		dependencyMgr: NewBuildDependencyManager(),
	}
}

// Build builds PHP with specific GCC version and dependencies
func (b *NativeGCCBuilder) Build(ctx context.Context, sourcePath string, installPath string, config map[string]string) error {
	fmt.Printf("🔧 Building PHP %s with GCC %s on native system...\n", b.phpVersion.Version, b.gccVersion)

	// Step 1: Install build dependencies
	if err := b.dependencyMgr.InstallDependencies(ctx, b.phpVersion); err != nil {
		fmt.Printf("⚠️  Failed to install dependencies: %v\n", err)
		fmt.Println("Continuing with build anyway...")
	}

	// Step 2: Ensure GCC version is available
	if err := b.ensureGCCVersion(ctx); err != nil {
		// Check if this is a GCC installation failure that should trigger Docker fallback
		if err.Error() == "gcc_installation_failed_fallback_to_docker" {
			fmt.Printf("🐳 Falling back to Docker build due to GCC installation failure\n")
			return b.fallbackToDocker(ctx, sourcePath, installPath, config)
		}
		fmt.Printf("⚠️  Failed to set up GCC %s: %v\n", b.gccVersion, err)
		fmt.Println("Falling back to system GCC...")
	}

	// Step 3: Build with the configured environment
	if err := b.buildWithEnvironment(ctx, sourcePath, installPath, config); err != nil {
		// If build fails and we should fall back to Docker, try that
		if b.shouldFallbackToDocker() {
			fmt.Printf("⚠️  Native build failed, falling back to Docker build: %v\n", err)
			return b.fallbackToDocker(ctx, sourcePath, installPath, config)
		}
		return err
	}

	return nil
}

// ensureGCCVersion ensures the specified GCC version is available
func (b *NativeGCCBuilder) ensureGCCVersion(ctx context.Context) error {
	gccMgr := NewGCCManager()

	// Check if GCC version is already available
	if gccMgr.IsGCCVersionAvailable(b.gccVersion) {
		fmt.Printf("✅ GCC %s is already available\n", b.gccVersion)
		return nil
	}

	// Try to install GCC version
	fmt.Printf("📦 Installing GCC %s...\n", b.gccVersion)
	actualVersion, err := gccMgr.InstallGCCVersion(ctx, b.gccVersion)
	if err != nil {
		// If GCC installation fails, check if we should fall back to Docker
		if b.shouldFallbackToDocker() {
			fmt.Printf("⚠️  GCC installation failed, falling back to Docker build\n")
			return fmt.Errorf("gcc_installation_failed_fallback_to_docker")
		}
		return fmt.Errorf("failed to install GCC and no fallback available: %w", err)
	}

	// Update the GCC version to the one that was actually installed
	if actualVersion != b.gccVersion {
		fmt.Printf("ℹ️  Using GCC %s instead of requested GCC %s\n", actualVersion, b.gccVersion)
		b.gccVersion = actualVersion
	}

	return nil
}

// shouldFallbackToDocker determines if we should fall back to Docker for this PHP version
func (b *NativeGCCBuilder) shouldFallbackToDocker() bool {
	// For very old PHP versions, Docker fallback is recommended
	return b.phpVersion.Major <= 7
}

// fallbackToDocker creates a Docker builder and delegates the build
func (b *NativeGCCBuilder) fallbackToDocker(ctx context.Context, sourcePath string, installPath string, config map[string]string) error {
	dockerImage := b.phpVersion.GetRecommendedDockerImage()
	fmt.Printf("🐳 Falling back to Docker build with image: %s\n", dockerImage)

	dockerBuilder := NewDockerBuilder(b.baseBuilder, dockerImage)
	return dockerBuilder.Build(ctx, sourcePath, installPath, config)
}

// buildWithEnvironment builds PHP with the correct environment variables set
func (b *NativeGCCBuilder) buildWithEnvironment(ctx context.Context, sourcePath string, installPath string, config map[string]string) error {
	// Set environment variables for the specific GCC version
	env := os.Environ()

	if b.gccVersion != "" {
		gccBin := fmt.Sprintf("gcc-%s", b.gccVersion)
		gxxBin := fmt.Sprintf("g++-%s", b.gccVersion)

		// Check if the GCC binaries exist
		if _, err := exec.LookPath(gccBin); err == nil {
			fmt.Printf("🔧 Setting CC=%s and CXX=%s\n", gccBin, gxxBin)
			env = append(env, fmt.Sprintf("CC=%s", gccBin))
			env = append(env, fmt.Sprintf("CXX=%s", gxxBin))
		} else {
			fmt.Printf("⚠️  GCC binary %s not found in PATH\n", gccBin)
		}
	}

	// Create a modified builder that uses our environment
	envBuilder := &EnvironmentBuilder{
		baseBuilder: b.baseBuilder,
		env:         env,
	}

	return envBuilder.Build(ctx, sourcePath, installPath, config)
}

// GetBuildStrategy returns the build strategy
func (b *NativeGCCBuilder) GetBuildStrategy() domain.BuildStrategy {
	return domain.BuildStrategySpecificGCC
}

// EnvironmentBuilder wraps a builder with custom environment variables
type EnvironmentBuilder struct {
	baseBuilder Builder
	env         []string
}

// Build builds with custom environment
func (b *EnvironmentBuilder) Build(ctx context.Context, sourcePath string, installPath string, config map[string]string) error {
	// Store the environment in the config so the SourceBuilder can use it
	config["_ENV_"] = strings.Join(b.env, "\n")
	return b.baseBuilder.Build(ctx, sourcePath, installPath, config)
}

// GetBuildStrategy returns the build strategy
func (b *EnvironmentBuilder) GetBuildStrategy() domain.BuildStrategy {
	return b.baseBuilder.GetBuildStrategy()
}

// BuildDependencyManager handles installation of build dependencies for different PHP versions
type BuildDependencyManager struct{}

// NewBuildDependencyManager creates a new dependency manager
func NewBuildDependencyManager() *BuildDependencyManager {
	return &BuildDependencyManager{}
}

// InstallDependencies installs build dependencies for the given PHP version
func (m *BuildDependencyManager) InstallDependencies(ctx context.Context, version domain.PHPVersion) error {
	fmt.Printf("📦 Installing build dependencies for PHP %s...\n", version.Version)

	// Detect package manager
	pkgMgr := m.detectPackageManager()
	if pkgMgr == "" {
		return fmt.Errorf("no supported package manager found")
	}

	// Get dependencies for this PHP version
	deps := m.getDependenciesForVersion(version)

	// Install dependencies
	return m.installPackages(ctx, pkgMgr, deps)
}

// detectPackageManager detects the available package manager
func (m *BuildDependencyManager) detectPackageManager() string {
	managers := []string{"apt-get", "yum", "dnf", "pacman", "zypper"}

	for _, mgr := range managers {
		if _, err := exec.LookPath(mgr); err == nil {
			return mgr
		}
	}

	return ""
}

// getDependenciesForVersion returns the build dependencies for a PHP version
func (m *BuildDependencyManager) getDependenciesForVersion(version domain.PHPVersion) []string {
	// Base dependencies for all PHP versions
	baseDeps := []string{
		"build-essential", "wget", "tar", "gzip",
		"libxml2-dev", "libssl-dev", "libcurl4-openssl-dev",
		"libpng-dev", "libjpeg-dev", "libfreetype6-dev",
		"libbz2-dev", "libreadline-dev",
	}

	// Version-specific dependencies
	switch version.Major {
	case 4, 5:
		// Older PHP versions may need additional libraries
		baseDeps = append(baseDeps, "libmcrypt-dev", "libmysqlclient-dev")
	case 7:
		if version.Minor < 4 {
			baseDeps = append(baseDeps, "libmcrypt-dev")
		}
	}

	return baseDeps
}

// installPackages installs packages using the detected package manager
func (m *BuildDependencyManager) installPackages(ctx context.Context, pkgMgr string, packages []string) error {
	var cmd *exec.Cmd

	switch pkgMgr {
	case "apt-get":
		// For apt-get, we need to run update first, then install
		updateCmd := exec.CommandContext(ctx, "sudo", "apt-get", "update")
		if err := updateCmd.Run(); err != nil {
			fmt.Printf("⚠️  Failed to update package list: %v\n", err)
		}
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"apt-get", "install", "-y"}, packages...)...)
	case "yum":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"yum", "install", "-y"}, packages...)...)
	case "dnf":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"dnf", "install", "-y"}, packages...)...)
	case "pacman":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"pacman", "-S", "--noconfirm"}, packages...)...)
	case "zypper":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"zypper", "install", "-y"}, packages...)...)
	default:
		return fmt.Errorf("unsupported package manager: %s", pkgMgr)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GCCManager handles GCC version installation and management
type GCCManager struct{}

// NewGCCManager creates a new GCC manager
func NewGCCManager() *GCCManager {
	return &GCCManager{}
}

// IsGCCVersionAvailable checks if a specific GCC version is available
func (m *GCCManager) IsGCCVersionAvailable(version string) bool {
	gccBin := fmt.Sprintf("gcc-%s", version)
	_, err := exec.LookPath(gccBin)
	return err == nil
}

// InstallGCCVersion installs a specific GCC version with fallback options
// Returns the actual version that was installed (which may differ from requested version)
func (m *GCCManager) InstallGCCVersion(ctx context.Context, version string) (string, error) {
	fmt.Printf("📦 Installing GCC %s...\n", version)

	// Detect package manager
	pkgMgr := m.detectPackageManager()
	if pkgMgr == "" {
		return "", fmt.Errorf("no supported package manager found for GCC installation")
	}

	// For very old GCC versions on modern systems, try available versions
	versionsToTry := []string{version}

	// Add fallback versions based on the requested version
	switch version {
	case "4.8", "5", "6":
		// For very old GCC versions on modern distros, try what's available
		// Ubuntu 24.04+ typically has GCC 11, 12, 13, 14, 15
		versionsToTry = append(versionsToTry, "11", "12", "13", "14", "15")
	case "7", "8", "9":
		// For moderately old versions, try adjacent versions
		versionsToTry = append(versionsToTry, "10", "11", "12", "13", "14", "15")
	case "10", "11", "12":
		// For newer versions, try adjacent versions
		versionsToTry = append(versionsToTry, "13", "14", "15")
	}

	var lastErr error
	for _, gccVer := range versionsToTry {
		fmt.Printf("🔄 Trying GCC %s...\n", gccVer)
		packages := []string{fmt.Sprintf("gcc-%s", gccVer), fmt.Sprintf("g++-%s", gccVer)}

		if err := m.installPackages(ctx, pkgMgr, packages); err == nil {
			fmt.Printf("✅ Successfully installed GCC %s\n", gccVer)
			return gccVer, nil
		} else {
			fmt.Printf("⚠️  Failed to install GCC %s: %v\n", gccVer, err)
			lastErr = err
		}
	}

	// If all GCC installations failed, this indicates we're on a very modern distro
	// where old GCC versions aren't available. We should recommend Docker instead.
	return "", fmt.Errorf("no compatible GCC version could be installed. This often happens on modern Linux distributions (Ubuntu 24.04+, etc.) where old GCC versions are not available. Consider using Docker build strategy instead: %w", lastErr)
}

// detectPackageManager detects the available package manager (duplicate from BuildDependencyManager, could be refactored)
func (m *GCCManager) detectPackageManager() string {
	managers := []string{"apt-get", "yum", "dnf", "pacman", "zypper"}

	for _, mgr := range managers {
		if _, err := exec.LookPath(mgr); err == nil {
			return mgr
		}
	}

	return ""
}

// installPackages installs packages (duplicate from BuildDependencyManager, could be refactored)
func (m *GCCManager) installPackages(ctx context.Context, pkgMgr string, packages []string) error {
	var cmd *exec.Cmd

	switch pkgMgr {
	case "apt-get":
		updateCmd := exec.CommandContext(ctx, "sudo", "apt-get", "update")
		if err := updateCmd.Run(); err != nil {
			fmt.Printf("⚠️  Failed to update package list: %v\n", err)
		}
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"apt-get", "install", "-y"}, packages...)...)
	case "yum":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"yum", "install", "-y"}, packages...)...)
	case "dnf":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"dnf", "install", "-y"}, packages...)...)
	case "pacman":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"pacman", "-S", "--noconfirm"}, packages...)...)
	case "zypper":
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"zypper", "install", "-y"}, packages...)...)
	default:
		return fmt.Errorf("unsupported package manager: %s", pkgMgr)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
