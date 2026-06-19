package terminal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	silopkg "github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/shim"
)

const composerPharURLTemplate = "https://getcomposer.org/download/%s/composer.phar"
const piePharURLTemplate = "https://github.com/php/pie/releases/latest/download/pie.phar"
const wpCliPharURLTemplate = "https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar"

// composerVersionForPHP returns the highest Composer version compatible with the given PHP version.
// SHA-256/SHA-512 phar support is now enabled for all PHP versions via dependency-sorted configure flags.
func composerVersionForPHP(phpVersion string, requestedVersion string) string {
	if requestedVersion != "" && requestedVersion != "latest-stable" {
		return requestedVersion
	}
	return "latest-stable"
}

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

	activePHP := h.detectActivePHPVersion(silo)
	if activePHP == "" {
		return nil, fmt.Errorf("no active PHP version. Run 'phpv use <version>' first")
	}

	// Auto-detect compatible version for composer based on active PHP version
	if pharName == "composer" && exactVersion == "latest-stable" {
		mapped := composerVersionForPHP(activePHP, exactVersion)
		if mapped != exactVersion {
			fmt.Printf("Auto-selected composer %s for PHP %s\n", mapped, activePHP)
		}
		exactVersion = mapped
	}

	pharDir := silopkg.VersionPharPath(silo, activePHP)

	switch pharName {
	case "composer":
		url = fmt.Sprintf(composerPharURLTemplate, exactVersion)
		destPath = filepath.Join(pharDir, "composer.phar")
	case "pie":
		url = piePharURLTemplate
		destPath = filepath.Join(pharDir, "pie.phar")
	case "wp-cli":
		url = wpCliPharURLTemplate
		destPath = filepath.Join(pharDir, "wp-cli.phar")
	}

	if isUpdate {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("%s is not installed, use 'phpv phar install %s' instead", name, name)
		}
	}

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

	activePHP := h.detectActivePHPVersion(silo)
	if activePHP == "" {
		return fmt.Errorf("no active PHP version. Run 'phpv use <version>' first")
	}

	pharDir := silopkg.VersionPharPath(silo, activePHP)
	var destPath string
	switch pharName {
	case "composer":
		destPath = filepath.Join(pharDir, "composer.phar")
	case "pie":
		destPath = filepath.Join(pharDir, "pie.phar")
	case "wp-cli":
		destPath = filepath.Join(pharDir, "wp-cli.phar")
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

	activePHP := h.detectActivePHPVersion(silo)
	if activePHP == "" {
		return nil, fmt.Errorf("no active PHP version. Run 'phpv use <version>' first")
	}

	pharDir := silopkg.VersionPharPath(silo, activePHP)
	entries, err := os.ReadDir(pharDir)
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
	silo, err := h.Silo.GetSilo()
	if err != nil {
		return "", nil
	}
	activePHP := h.detectActivePHPVersion(silo)
	if activePHP == "" {
		return "", fmt.Errorf("no active PHP version")
	}
	return filepath.Join(silopkg.VersionPharPath(silo, activePHP), pharName+".phar"), nil
}

// detectActivePHPVersion returns the currently active PHP version by checking
// environment variable, default file, or scanning installed versions.
func (h *TerminalHandler) detectActivePHPVersion(silo *domain.Silo) string {
	// 1. Check PHPV_CURRENT env var
	if v := os.Getenv("PHPV_CURRENT"); v != "" {
		return v
	}
	// 2. Check default marker
	defaultFile := filepath.Join(silo.Root, "default")
	if data, err := os.ReadFile(defaultFile); err == nil {
		return strings.TrimSpace(string(data))
	}
	// 3. Find latest installed version
	versionsDir := filepath.Join(silo.Root, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return ""
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].IsDir() {
			return entries[i].Name()
		}
	}
	return ""
}

func (h *TerminalHandler) regeneratePharShims(silo *domain.Silo) error {
	return shim.WriteShims(shim.ShimConfig{
		BinPath: silopkg.BinPath(silo),
	})
}

// regenerateAllShims regenerates all shims (php, composer, pie, wp) for the current installation.
// Called by 'phpv init' to ensure shims always match the current binary.
func (h *TerminalHandler) regenerateAllShims() {
	silo, err := h.Silo.GetSilo()
	if err != nil {
		return
	}
	h.regeneratePharShims(silo)
}
