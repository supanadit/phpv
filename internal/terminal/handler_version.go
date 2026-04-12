package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/shim"
)

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

func (h *TerminalHandler) resolveActivePHP() (string, error) {
	if phpvCurrent := os.Getenv("PHPV_CURRENT"); phpvCurrent != "" {
		return h.resolveInstalledVersion(phpvCurrent)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	if _, version, err := utils.FindPhpvrcFromPath(cwd); err == nil && version != "" {
		return h.resolveInstalledVersion(version)
	}

	if _, version, err := utils.FindComposerJSONFromPath(cwd); err == nil && version != "" {
		return h.resolveInstalledVersion(version)
	}

	defaultVer, err := h.Silo.GetDefault()
	if err != nil {
		return "", err
	}
	if defaultVer != "" {
		return h.resolveInstalledVersion(defaultVer)
	}

	return "", fmt.Errorf("no active PHP version. Set default with 'phpv default <version>' or run 'phpv use <version>'")
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

func (h *TerminalHandler) resolveInstalledVersion(constraint string) (string, error) {
	versions := h.Silo.ListVersions()
	return utils.ResolveInstalledVersion(versions, constraint)
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
