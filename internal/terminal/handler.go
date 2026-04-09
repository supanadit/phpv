package terminal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/shim"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/source"
)

type UseResult struct {
	ExactVersion string
	ShimPath     string
	OutputPath   string
}

type TerminalHandler struct {
	BundlerRepo bundler.BundlerRepository
	Silo        silo.SiloRepository
	Source      source.SourceRepository
}

const composerPharURLTemplate = "https://getcomposer.org/download/%s/composer.phar"

func (h *TerminalHandler) PharInstall(name string, version string) (*domain.PharResult, error) {
	return h.pharInstallOrUpdate(name, version, false)
}

func (h *TerminalHandler) PharUpdate(name string, version string) (*domain.PharResult, error) {
	return h.pharInstallOrUpdate(name, version, true)
}

func (h *TerminalHandler) pharInstallOrUpdate(name string, version string, isUpdate bool) (*domain.PharResult, error) {
	if name != "composer" {
		return nil, fmt.Errorf("unsupported phar: %s", name)
	}

	exactVersion := version
	if exactVersion == "" {
		exactVersion = "latest-stable"
	}

	url := fmt.Sprintf(composerPharURLTemplate, exactVersion)

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}

	destPath := filepath.Join(utils.PharPath(silo), "composer.phar")

	if isUpdate {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("composer is not installed, use 'phpv phar install composer' instead")
		}
	}

	pharDir := filepath.Dir(destPath)
	if err := os.MkdirAll(pharDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create phar directory: %w", err)
	}

	tmpPath := destPath + ".tmp"

	out, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write phar: %w", err)
	}
	out.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return nil, fmt.Errorf("failed to move phar to final location: %w", err)
	}

	return &domain.PharResult{
		Name:    name,
		Version: exactVersion,
		Path:    destPath,
		Updated: isUpdate,
	}, nil
}

func (h *TerminalHandler) PharRemove(name string) error {
	if name != "composer" {
		return fmt.Errorf("unsupported phar: %s", name)
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return fmt.Errorf("failed to get silo: %w", err)
	}

	destPath := filepath.Join(utils.PharPath(silo), "composer.phar")

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return fmt.Errorf("composer is not installed")
	}

	if err := os.Remove(destPath); err != nil {
		return fmt.Errorf("failed to remove composer: %w", err)
	}

	return nil
}

func (h *TerminalHandler) PharList() ([]string, error) {
	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}

	binPath := utils.PharPath(silo)
	entries, err := os.ReadDir(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read bin directory: %w", err)
	}

	var phars []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".phar") {
			phars = append(phars, entry.Name())
		}
	}

	return phars, nil
}

func (h *TerminalHandler) PharWhich(name string) (string, error) {
	if name != "composer" {
		return "", fmt.Errorf("unsupported phar: %s", name)
	}
	return shim.DetectComposerPath(), nil
}

func NewHandler(
	bundlerRepo bundler.BundlerRepository,
	siloRepo silo.SiloRepository,
	sourceSvc source.SourceRepository,
) *TerminalHandler {
	return &TerminalHandler{
		BundlerRepo: bundlerRepo,
		Silo:        siloRepo,
		Source:      sourceSvc,
	}
}

func (h *TerminalHandler) Install(version string, compiler string, extensions []string, verbose bool, fresh bool) (domain.Forge, error) {
	return h.BundlerRepo.Install(version, compiler, extensions, fresh)
}

func (h *TerminalHandler) Rebuild(version string, compiler string, extensions []string, verbose bool) (domain.Forge, error) {
	return h.BundlerRepo.Rebuild(version, compiler, extensions)
}

func (h *TerminalHandler) Use(constraint string) (*UseResult, error) {
	exactVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return nil, err
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}

	_ = shim.RemoveSystemMarker(silo.Root)

	outputPath := utils.PHPOutputPath(silo, exactVersion)
	phpBinary := filepath.Join(outputPath, "bin", "php")
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		return nil, fmt.Errorf("PHP %s is not properly installed (binary not found: %s)", exactVersion, phpBinary)
	}

	shimPath := utils.BinPath(silo)
	composerPath := shim.DetectComposerPath()

	if err := shim.WriteShims(shim.ShimConfig{
		BinPath:      shimPath,
		ComposerPath: composerPath,
	}); err != nil {
		return nil, fmt.Errorf("failed to write shims: %w", err)
	}

	return &UseResult{
		ExactVersion: exactVersion,
		ShimPath:     shimPath,
		OutputPath:   outputPath,
	}, nil
}

func (h *TerminalHandler) UseSystem() (*UseResult, error) {
	systemPHP, err := exec.LookPath("php")
	if err != nil {
		return nil, fmt.Errorf("no system PHP found: please install PHP first")
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}

	if err := shim.WriteSystemMarker(silo.Root); err != nil {
		return nil, err
	}

	shimPath := utils.BinPath(silo)
	composerPath := shim.DetectComposerPath()

	if err := shim.WriteShims(shim.ShimConfig{
		BinPath:      shimPath,
		ComposerPath: composerPath,
	}); err != nil {
		return nil, fmt.Errorf("failed to write shims: %w", err)
	}

	return &UseResult{
		ExactVersion: "system",
		ShimPath:     shimPath,
		OutputPath:   filepath.Dir(systemPHP),
	}, nil
}

func (h *TerminalHandler) ShellUse(constraint string) error {
	exactVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return err
	}

	if err := h.Silo.SetDefault(exactVersion); err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	return nil
}

func (h *TerminalHandler) AutoDetect() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	version, err := utils.ParseComposerJSON(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to parse composer.json: %w", err)
	}

	if version == "" {
		return "", fmt.Errorf("no PHP version configured in composer.json")
	}

	return version, nil
}

func (h *TerminalHandler) AutoDetectResolve(constraint string) (string, error) {
	if constraint == "" {
		var err error
		constraint, err = h.AutoDetect()
		if err != nil {
			return "", err
		}
	}

	exactVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return "", fmt.Errorf("PHP %s is not installed: %w", constraint, err)
	}

	return exactVersion, nil
}

func (h *TerminalHandler) SetDefault(constraint string) error {
	exactVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return err
	}

	return h.Silo.SetDefault(exactVersion)
}

func (h *TerminalHandler) GetDefault() (string, error) {
	return h.Silo.GetDefault()
}

func (h *TerminalHandler) ListInstalled() ([]string, error) {
	versions := h.Silo.ListVersions()
	utils.SortVersions(versions)
	return versions, nil
}

func (h *TerminalHandler) ListAvailable() ([]domain.Source, error) {
	sources, err := h.Source.GetVersions()
	if err != nil {
		return nil, err
	}

	var phpSources []domain.Source
	for _, src := range sources {
		if src.Name == "php" {
			phpSources = append(phpSources, src)
		}
	}

	return phpSources, nil
}

func (h *TerminalHandler) Which() (string, error) {
	silo, err := h.Silo.GetSilo()
	if err != nil {
		return "", fmt.Errorf("failed to get silo: %w", err)
	}

	if shim.IsSystemMode(silo.Root) {
		phpvBin := filepath.Join(silo.Root, "bin")
		pathEnv := os.Getenv("PATH")
		var filteredParts []string
		for _, part := range strings.Split(pathEnv, ":") {
			if part != phpvBin && !strings.HasPrefix(part, silo.Root+"/") {
				filteredParts = append(filteredParts, part)
			}
		}
		filteredPath := strings.Join(filteredParts, ":")

		cmd := exec.Command("which", "php")
		cmd.Env = append(os.Environ(), "PATH="+filteredPath)
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}

	defaultVer, err := h.Silo.GetDefault()
	if err != nil {
		return "", err
	}

	if defaultVer == "" {
		return "", nil
	}

	phpPath := filepath.Join(utils.PHPOutputPath(silo, defaultVer), "bin", "php")
	if _, err := os.Stat(phpPath); os.IsNotExist(err) {
		return "", nil
	}
	return phpPath, nil
}

func (h *TerminalHandler) resolveInstalledVersion(constraint string) (string, error) {
	versions := h.Silo.ListVersions()
	return utils.ResolveInstalledVersion(versions, constraint)
}

func (h *TerminalHandler) Uninstall(constraint string) (*UninstallResult, error) {
	exactVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return nil, fmt.Errorf("version not installed: %w", err)
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}

	outputPath := utils.PHPOutputPath(silo, exactVersion)
	phpBinary := filepath.Join(outputPath, "bin", "php")
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		h.Silo.RemovePHPInstallation(exactVersion)
		return nil, fmt.Errorf("PHP %s is not properly installed (binary not found: %s), cleaning up stale data", exactVersion, phpBinary)
	}

	state, err := h.Silo.GetState(exactVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}
	if state == domain.StateNone {
		return nil, fmt.Errorf("version %s is not installed", exactVersion)
	}

	defaultVer, _ := h.Silo.GetDefault()
	wasDefault := defaultVer == exactVersion

	removedTools, err := h.Silo.RemovePHPInstallation(exactVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to uninstall: %w", err)
	}

	shimPath := utils.BinPath(silo)
	for _, name := range []string{"php", "php-cgi", "phpize", "php-config"} {
		shimPath := filepath.Join(shimPath, name)
		_ = os.RemoveAll(shimPath)
	}

	return &UninstallResult{
		Version:      exactVersion,
		RemovedTools: removedTools,
		WasDefault:   wasDefault,
	}, nil
}

func (h *TerminalHandler) CleanBuildTools(dryRun bool) (*CleanBuildToolsResult, error) {
	removed, willRemove, err := h.Silo.RemoveUnusedBuildTools(dryRun)
	if err != nil {
		return nil, err
	}

	return &CleanBuildToolsResult{
		Removed:    removed,
		WillRemove: willRemove,
		DryRun:     dryRun,
	}, nil
}

func (h *TerminalHandler) Upgrade(constraint string) (*UpgradeResult, error) {
	currentVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return nil, fmt.Errorf("no installed version matching %q: %w", constraint, err)
	}

	availableSources, err := h.Source.GetVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to get available versions: %w", err)
	}

	var phpVersions []string
	for _, src := range availableSources {
		if src.Name == "php" {
			phpVersions = append(phpVersions, src.Version)
		}
	}

	currentParsed := utils.ParseVersion(currentVersion)
	var latestMatch string

	for _, v := range phpVersions {
		parsed := utils.ParseVersion(v)
		if parsed.Major == currentParsed.Major && parsed.Minor == currentParsed.Minor {
			if latestMatch == "" || utils.CompareVersions(parsed, utils.ParseVersion(latestMatch)) > 0 {
				latestMatch = v
			}
		}
	}

	if latestMatch == "" || latestMatch == currentVersion {
		return nil, fmt.Errorf("no newer version available for %s (currently at %s)", constraint, currentVersion)
	}

	_, err = h.BundlerRepo.Install(latestMatch, "", nil, true)
	if err != nil {
		return nil, fmt.Errorf("failed to install new version: %w", err)
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}
	outputPath := utils.PHPOutputPath(silo, latestMatch)

	return &UpgradeResult{
		FromVersion: currentVersion,
		ToVersion:   latestMatch,
		Forge: domain.Forge{
			Prefix: outputPath,
			Env:    map[string]string{},
		},
	}, nil
}

func (h *TerminalHandler) Doctor() (*DoctorResult, error) {
	var issues []DoctorIssue
	var warnings []DoctorWarning

	if runtime.GOOS == "linux" {
		requiredCommands := []string{"make", "gcc", "bison", "flex", "re2c", " autoconf", "automake", "libtool", "pkg-config"}
		for _, cmd := range requiredCommands {
			name := strings.TrimPrefix(cmd, " ")
			if name == "autoconf" || name == "automake" || name == "libtool" {
				name = cmd
			}
			if _, err := exec.LookPath(name); err != nil {
				issues = append(issues, DoctorIssue{
					Category: "system",
					Message:  fmt.Sprintf("required command not found: %s", name),
				})
			}
		}

		if _, err := exec.LookPath("xz"); err != nil {
			issues = append(issues, DoctorIssue{
				Category: "system",
				Message:  "xz utility not found (required for extracting .tar.xz archives)",
			})
		}
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		issues = append(issues, DoctorIssue{
			Category: "phpv",
			Message:  fmt.Sprintf("failed to get silo: %v", err),
		})
	} else {
		if _, err := os.Stat(silo.Root); os.IsNotExist(err) {
			issues = append(issues, DoctorIssue{
				Category: "phpv",
				Message:  fmt.Sprintf("PHPV_ROOT does not exist: %s", silo.Root),
			})
		}

		defaultVer, _ := h.Silo.GetDefault()
		if defaultVer != "" {
			phpPath := filepath.Join(utils.PHPOutputPath(silo, defaultVer), "bin", "php")
			if _, err := os.Stat(phpPath); os.IsNotExist(err) {
				warnings = append(warnings, DoctorWarning{
					Category: "phpv",
					Message:  fmt.Sprintf("default PHP version set to %s but binary not found", defaultVer),
				})
			}
		}
	}

	if len(issues) == 0 && len(warnings) == 0 {
		warnings = append(warnings, DoctorWarning{
			Category: "system",
			Message:  "no issues detected",
		})
	}

	return &DoctorResult{
		Issues:   issues,
		Warnings: warnings,
	}, nil
}

func (h *TerminalHandler) GetInitCode(shell string) (string, error) {
	phpvRoot := GetPHPvRoot()
	return GetInitCodeForShell(shell, phpvRoot), nil
}

func (h *TerminalHandler) GetPHPvRoot() string {
	return GetPHPvRoot()
}

func (h *TerminalHandler) ListVersionsFormatted() (*VersionsResult, error) {
	versions, err := h.ListInstalled()
	if err != nil {
		return nil, err
	}

	defaultVer, _ := h.GetDefault()

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, err
	}

	result := &VersionsResult{
		Versions:   make([]VersionInfo, 0, len(versions)+1),
		DefaultVer: defaultVer,
	}

	systemPHPPath, systemPHPVersion := h.detectSystemPHP(silo.Root)
	if systemPHPPath != "" {
		result.Versions = append(result.Versions, VersionInfo{
			Version:    systemPHPVersion,
			IsDefault:  false,
			IsSystem:   true,
			SystemPath: systemPHPPath,
		})
	}

	for _, v := range versions {
		result.Versions = append(result.Versions, VersionInfo{
			Version:   v,
			IsDefault: v == defaultVer,
			IsSystem:  false,
		})
	}

	return result, nil
}

func (h *TerminalHandler) detectSystemPHP(siloRoot string) (path string, version string) {
	phpvBin := filepath.Join(siloRoot, "bin")
	pathEnv := os.Getenv("PATH")
	var filteredParts []string
	for _, part := range strings.Split(pathEnv, ":") {
		if part != phpvBin && !strings.HasPrefix(part, siloRoot+"/") {
			filteredParts = append(filteredParts, part)
		}
	}
	filteredPath := strings.Join(filteredParts, ":")

	cmd := exec.Command("which", "php")
	cmd.Env = append(os.Environ(), "PATH="+filteredPath)
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}
	path = strings.TrimSpace(string(out))

	versionCmd := exec.Command(path, "-r", "echo PHP_VERSION;")
	versionCmd.Env = append(os.Environ(), "PATH="+filteredPath)
	versionOut, err := versionCmd.Output()
	if err != nil {
		return path, ""
	}
	version = strings.TrimSpace(string(versionOut))
	return path, version
}

func (h *TerminalHandler) ListAvailableFormatted() (*ListResult, error) {
	sources, err := h.ListAvailable()
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, src := range sources {
		versions = append(versions, src.Version)
	}

	utils.SortVersions(versions)

	return &ListResult{
		Versions: versions,
	}, nil
}

func (h *TerminalHandler) PECLInstall(archivePath string) (*PECLInstallResult, error) {
	defaultVer, err := h.Silo.GetDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to get default PHP version: %w", err)
	}
	if defaultVer == "" {
		return nil, fmt.Errorf("no default PHP version set. Run 'phpv use <version>' first")
	}

	ext, err := h.BundlerRepo.PECLInstall(archivePath, defaultVer)
	if err != nil {
		return nil, err
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}
	extensionsDir := filepath.Join(utils.PHPOutputPath(silo, defaultVer), "lib", "extensions", ext.Name)

	return &PECLInstallResult{
		Name:       ext.Name,
		Version:    ext.Version,
		InstallDir: extensionsDir,
	}, nil
}

func (h *TerminalHandler) PECLList() ([]string, error) {
	defaultVer, err := h.Silo.GetDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to get default PHP version: %w", err)
	}
	if defaultVer == "" {
		return nil, fmt.Errorf("no default PHP version set. Run 'phpv use <version>' first")
	}

	return h.BundlerRepo.PECLList(defaultVer)
}

func (h *TerminalHandler) PECLUninstall(name string) error {
	defaultVer, err := h.Silo.GetDefault()
	if err != nil {
		return fmt.Errorf("failed to get default PHP version: %w", err)
	}
	if defaultVer == "" {
		return fmt.Errorf("no default PHP version set. Run 'phpv use <version>' first")
	}

	return h.BundlerRepo.PECLUninstall(name, defaultVer)
}
