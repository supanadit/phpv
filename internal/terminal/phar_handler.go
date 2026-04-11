package terminal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/shim"
)

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
