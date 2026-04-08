package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (s *bundlerRepository) PECLInstall(archivePath string, phpVersion string) (*domain.Extension, error) {
	if err := s.validateArchivePath(archivePath); err != nil {
		return nil, err
	}

	phpBin := filepath.Join(utils.PHPOutputPath(s.silo, phpVersion), "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("PHP %s is not installed at %s", phpVersion, phpBin)
	}

	phpizeBin := filepath.Join(utils.PHPOutputPath(s.silo, phpVersion), "bin", "phpize")
	phpConfigBin := filepath.Join(utils.PHPOutputPath(s.silo, phpVersion), "bin", "php-config")

	extName, extVersion, err := s.extractExtensionInfo(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract extension info from archive: %w", err)
	}

	extBaseDir := filepath.Join(utils.PHPOutputPath(s.silo, phpVersion), "lib", "extensions")
	if err := os.MkdirAll(extBaseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create extensions directory: %w", err)
	}

	extractDir := filepath.Join(extBaseDir, extName)
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create extension directory: %w", err)
	}

	if _, err := s.unloadSvc.Unpack(archivePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract PECL archive: %w", err)
	}

	sourceDir := s.findExtensionSourceDir(extractDir, extName)
	if sourceDir == "" {
		return nil, fmt.Errorf("could not find extension source directory in %s", extractDir)
	}

	s.logInfo("Building PECL extension %s...", extName)

	phpizeCmd := exec.Command(phpizeBin)
	phpizeCmd.Dir = sourceDir
	if output, err := phpizeCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("phpize failed: %s, %w", string(output), err)
	}

	configureCmd := exec.Command(
		"./configure",
		"--with-php-config="+phpConfigBin,
	)
	configureCmd.Dir = sourceDir
	configureCmd.Env = append(os.Environ(),
		"CFLAGS=-O2",
	)
	if output, err := configureCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("configure failed: %s, %w", string(output), err)
	}

	makeCmd := exec.Command("make", "-j", fmt.Sprintf("%d", s.jobs))
	makeCmd.Dir = sourceDir
	if output, err := makeCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("make failed: %s, %w", string(output), err)
	}

	makeInstallCmd := exec.Command("make", "install")
	makeInstallCmd.Dir = sourceDir
	if output, err := makeInstallCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("make install failed: %s, %w", string(output), err)
	}

	s.logInfo("✓ PECL extension %s installed successfully", extName)

	return &domain.Extension{
		Name:    extName,
		Type:    domain.ExtensionTypePECL,
		Version: extVersion,
	}, nil
}

func (s *bundlerRepository) PECLList(phpVersion string) ([]string, error) {
	extensionsDir := filepath.Join(utils.PHPOutputPath(s.silo, phpVersion), "lib", "extensions")

	if _, err := os.Stat(extensionsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(extensionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read extensions directory: %w", err)
	}

	var extensions []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "" && entry.Name() != "." && entry.Name() != ".." {
			extensions = append(extensions, entry.Name())
		}
	}

	return extensions, nil
}

func (s *bundlerRepository) PECLUninstall(name string, phpVersion string) error {
	extensionsDir := filepath.Join(utils.PHPOutputPath(s.silo, phpVersion), "lib", "extensions")
	extDir := filepath.Join(extensionsDir, name)

	if _, err := os.Stat(extDir); os.IsNotExist(err) {
		return fmt.Errorf("extension %s is not installed", name)
	}

	if err := os.RemoveAll(extDir); err != nil {
		return fmt.Errorf("failed to remove extension directory: %w", err)
	}

	s.logInfo("✓ PECL extension %s uninstalled", name)
	return nil
}

func (s *bundlerRepository) validateArchivePath(archivePath string) error {
	if archivePath == "" {
		return fmt.Errorf("archive path cannot be empty")
	}

	info, err := os.Stat(archivePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("archive does not exist: %s", archivePath)
	}
	if err != nil {
		return fmt.Errorf("failed to stat archive: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("archive path must be a file, not a directory")
	}

	if !strings.HasSuffix(archivePath, ".tgz") && !strings.HasSuffix(archivePath, ".tar.gz") && !strings.HasSuffix(archivePath, ".tar.bz2") {
		return fmt.Errorf("unsupported archive format: %s (expected .tgz, .tar.gz, or .tar.bz2)", archivePath)
	}

	return nil
}

var peclNameVersionRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)-(\d+\.\d+\.\d+(?:[a-zA-Z0-9._-]*)?)`)

func (s *bundlerRepository) extractExtensionInfo(archivePath string) (name string, version string, err error) {
	baseName := filepath.Base(archivePath)
	baseName = strings.TrimSuffix(baseName, ".tar.gz")
	baseName = strings.TrimSuffix(baseName, ".tar.bz2")
	baseName = strings.TrimSuffix(baseName, ".tgz")

	matches := peclNameVersionRegex.FindStringSubmatch(baseName)
	if len(matches) >= 3 {
		return matches[1], matches[2], nil
	}

	parts := strings.Split(baseName, "-")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}

	return baseName, "unknown", nil
}

func (s *bundlerRepository) findExtensionSourceDir(baseDir string, extName string) string {
	configM4Exists := func(dir string) bool {
		_, err := os.Stat(filepath.Join(dir, "config.m4"))
		return err == nil
	}

	if configM4Exists(baseDir) {
		return baseDir
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() && (entry.Name() == extName || strings.Contains(entry.Name(), extName)) {
			subDir := filepath.Join(baseDir, entry.Name())
			if configM4Exists(subDir) {
				return subDir
			}

			nested := s.findExtensionSourceDir(subDir, extName)
			if nested != "" {
				return nested
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(baseDir, entry.Name())
			if configM4Exists(subDir) {
				return subDir
			}
		}
	}

	return ""
}
