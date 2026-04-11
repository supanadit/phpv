package terminal

import (
	"fmt"
	"path/filepath"

	"github.com/supanadit/phpv/internal/utils"
)

func (h *TerminalHandler) PECLInstall(archivePath string) (*PECLInstallResult, error) {
	activeVer, err := h.resolveActivePHP()
	if err != nil {
		return nil, err
	}

	ext, err := h.BundlerRepo.PECLInstall(archivePath, activeVer)
	if err != nil {
		return nil, err
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}
	extensionsDir := filepath.Join(utils.PHPOutputPath(silo, activeVer), "lib", "extensions", ext.Name)

	return &PECLInstallResult{
		Name:       ext.Name,
		Version:    ext.Version,
		InstallDir: extensionsDir,
	}, nil
}

func (h *TerminalHandler) PECLList() ([]string, error) {
	activeVer, err := h.resolveActivePHP()
	if err != nil {
		return nil, err
	}

	return h.BundlerRepo.PECLList(activeVer)
}

func (h *TerminalHandler) PECLUninstall(name string) error {
	activeVer, err := h.resolveActivePHP()
	if err != nil {
		return err
	}

	return h.BundlerRepo.PECLUninstall(name, activeVer)
}
