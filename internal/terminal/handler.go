package terminal

import (
	"fmt"
	"os"
	"path/filepath"

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
	outputPath := utils.PHPOutputPath(silo, exactVersion)

	if err := shim.WriteShims(shimPath, exactVersion, outputPath); err != nil {
		return nil, fmt.Errorf("failed to write shims: %w", err)
	}

	return &UseResult{
		ExactVersion: exactVersion,
		ShimPath:     shimPath,
		OutputPath:   outputPath,
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
