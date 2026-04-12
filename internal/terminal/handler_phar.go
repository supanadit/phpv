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
const piePharURLTemplate = "https://github.com/php/pie/releases/latest/download/pie.phar"

func (h *TerminalHandler) PharInstall(name string, version string) (*domain.PharResult, error) {
	return h.pharInstallOrUpdate(name, version, false)
}

func (h *TerminalHandler) PharUpdate(name string, version string) (*domain.PharResult, error) {
	return h.pharInstallOrUpdate(name, version, true)
}

func (h *TerminalHandler) pharInstallOrUpdate(name string, version string, isUpdate bool) (*domain.PharResult, error) {
	if name != "composer" && name != "pie" {
		return nil, fmt.Errorf("unsupported phar: %s", name)
	}

	exactVersion := version
	if exactVersion == "" {
		exactVersion = "latest-stable"
	}

	var url string
	var destPath string
	silo, err := h.Silo.GetSilo()
	if err != nil {
		return nil, fmt.Errorf("failed to get silo: %w", err)
	}

	switch name {
	case "composer":
		url = fmt.Sprintf(composerPharURLTemplate, exactVersion)
		destPath = filepath.Join(utils.PharPath(silo), "composer.phar")
	case "pie":
		url = piePharURLTemplate
		destPath = filepath.Join(utils.PharPath(silo), "pie.phar")
	}

	if isUpdate {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("%s is not installed, use 'phpv phar install %s' instead", name, name)
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

	if err := h.regeneratePharShims(silo); err != nil {
		return nil, fmt.Errorf("failed to regenerate shims: %w", err)
	}

	return &domain.PharResult{
		Name:    name,
		Version: exactVersion,
		Path:    destPath,
		Updated: isUpdate,
	}, nil
}

func (h *TerminalHandler) PharRemove(name string) error {
	if name != "composer" && name != "pie" {
		return fmt.Errorf("unsupported phar: %s", name)
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return fmt.Errorf("failed to get silo: %w", err)
	}

	var destPath string
	switch name {
	case "composer":
		destPath = filepath.Join(utils.PharPath(silo), "composer.phar")
	case "pie":
		destPath = filepath.Join(utils.PharPath(silo), "pie.phar")
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return fmt.Errorf("%s is not installed", name)
	}

	if err := os.Remove(destPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", name, err)
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
	if name != "composer" && name != "pie" {
		return "", fmt.Errorf("unsupported phar: %s", name)
	}
	switch name {
	case "composer":
		return shim.DetectComposerPath(), nil
	case "pie":
		return shim.DetectPiePath(), nil
	}
	return "", nil
}

func (h *TerminalHandler) regeneratePharShims(silo *domain.Silo) error {
	shimPath := utils.BinPath(silo)
	composerPath := shim.DetectComposerPath()
	piePath := shim.DetectPiePath()

	return shim.WriteShims(shim.ShimConfig{
		BinPath:      shimPath,
		ComposerPath: composerPath,
		PiePath:      piePath,
	})
}
