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
const wpCliPharURLTemplate = "https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar"

func (h *TerminalHandler) PharInstall(name string, version string) (*domain.PharResult, error) {
	return h.pharInstallOrUpdate(name, version, false)
}

func (h *TerminalHandler) PharUpdate(name string, version string) (*domain.PharResult, error) {
	return h.pharInstallOrUpdate(name, version, true)
}

func (h *TerminalHandler) pharInstallOrUpdate(name string, version string, isUpdate bool) (*domain.PharResult, error) {
	pharName := normalizePharName(name)
	if pharName == "" {
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

	switch pharName {
	case "composer":
		url = fmt.Sprintf(composerPharURLTemplate, exactVersion)
		destPath = filepath.Join(utils.PharPath(silo), "composer.phar")
	case "pie":
		url = piePharURLTemplate
		destPath = filepath.Join(utils.PharPath(silo), "pie.phar")
	case "wp-cli":
		url = wpCliPharURLTemplate
		destPath = filepath.Join(utils.PharPath(silo), "wp-cli.phar")
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
		Name:    pharName,
		Version: exactVersion,
		Path:    destPath,
		Updated: isUpdate,
	}, nil
}

func normalizePharName(name string) string {
	switch name {
	case "composer":
		return "composer"
	case "pie":
		return "pie"
	case "wp", "wp-cli":
		return "wp-cli"
	default:
		return ""
	}
}

func (h *TerminalHandler) PharRemove(name string) error {
	pharName := normalizePharName(name)
	if pharName == "" {
		return fmt.Errorf("unsupported phar: %s", name)
	}

	silo, err := h.Silo.GetSilo()
	if err != nil {
		return fmt.Errorf("failed to get silo: %w", err)
	}

	var destPath string
	switch pharName {
	case "composer":
		destPath = filepath.Join(utils.PharPath(silo), "composer.phar")
	case "pie":
		destPath = filepath.Join(utils.PharPath(silo), "pie.phar")
	case "wp-cli":
		destPath = filepath.Join(utils.PharPath(silo), "wp-cli.phar")
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
	pharName := normalizePharName(name)
	if pharName == "" {
		return "", fmt.Errorf("unsupported phar: %s", name)
	}
	switch pharName {
	case "composer":
		return shim.DetectComposerPath(), nil
	case "pie":
		return shim.DetectPiePath(), nil
	case "wp-cli":
		return shim.DetectWpCliPath(), nil
	}
	return "", nil
}

func (h *TerminalHandler) regeneratePharShims(silo *domain.Silo) error {
	shimPath := utils.BinPath(silo)
	composerPath := shim.DetectComposerPath()
	piePath := shim.DetectPiePath()
	wpCliPath := shim.DetectWpCliPath()

	return shim.WriteShims(shim.ShimConfig{
		BinPath:      shimPath,
		ComposerPath: composerPath,
		PiePath:      piePath,
		WpCliPath:    wpCliPath,
	})
}
