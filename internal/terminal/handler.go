package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/shim"
	"github.com/supanadit/phpv/source"
)

type UseResult struct {
	ExactVersion string
	ShimPath     string
	OutputPath   string
}

type TerminalHandler struct {
	BundlerRepo bundler.BundlerRepository
	Silo        *disk.SiloRepository
	Source      source.SourceRepository
}

func NewHandler(
	bundlerRepo bundler.BundlerRepository,
	silo *disk.SiloRepository,
	sourceSvc source.SourceRepository,
) *TerminalHandler {
	return &TerminalHandler{
		BundlerRepo: bundlerRepo,
		Silo:        silo,
		Source:      sourceSvc,
	}
}

func (h *TerminalHandler) Install(version string, compiler string, verbose bool, fresh bool) (domain.Forge, error) {
	return h.BundlerRepo.Install(version, compiler, fresh)
}

func (h *TerminalHandler) Use(constraint string) (*UseResult, error) {
	exactVersion, err := h.resolveInstalledVersion(constraint)
	if err != nil {
		return nil, err
	}

	silo, _ := h.Silo.GetSilo()
	shimPath := utils.BinPath(silo)

	if err := shim.WriteShims(shimPath); err != nil {
		return nil, fmt.Errorf("failed to write shims: %w", err)
	}

	if err := h.Silo.SetDefault(exactVersion); err != nil {
		return nil, fmt.Errorf("failed to set default: %w", err)
	}

	return &UseResult{
		ExactVersion: exactVersion,
		ShimPath:     shimPath,
		OutputPath:   utils.PHPOutputPath(silo, exactVersion),
	}, nil
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
	defaultVer, err := h.Silo.GetDefault()
	if err != nil {
		return "", err
	}

	if defaultVer == "" {
		return "", nil
	}

	silo, _ := h.Silo.GetSilo()
	phpPath := filepath.Join(utils.PHPOutputPath(silo, defaultVer), "bin", "php")
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

	silo, _ := h.Silo.GetSilo()

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

	fmt.Printf("Upgrading PHP %s to %s...\n", currentVersion, latestMatch)

	_, err = h.BundlerRepo.Install(latestMatch, "", true)
	if err != nil {
		return nil, fmt.Errorf("failed to install new version: %w", err)
	}

	silo, _ := h.Silo.GetSilo()
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
