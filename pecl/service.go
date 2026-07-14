package pecl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
)

type InstallResult struct {
	Name       string
	Version    string
	InstallDir string
}

type Service struct {
	silo       *silo.Service
	downloader Downloader
}

func NewService(siloSvc *silo.Service) *Service {
	return &Service{
		silo:       siloSvc,
		downloader: newPECLDownloader(),
	}
}

func (s *Service) Install(ctx context.Context, source, phpVersion string, jobs int) (*InstallResult, error) {
	prefix := s.silo.PackagePrefix("php", phpVersion)
	phpBin := filepath.Join(prefix, "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("PHP %s is not installed at %s", phpVersion, prefix)
	}

	archivePath := source
	extName := ""
	extVersion := ""

	if isLocalArchive(source) {
		var err error
		extName, extVersion, err = parseNameVersion(source)
		if err != nil {
			return nil, fmt.Errorf("parse archive name: %w", err)
		}
	} else {
		pkg, err := s.downloader.ResolveLatest(source)
		if err != nil {
			return nil, fmt.Errorf("resolve pecl package %s: %w", source, err)
		}
		extName = pkg.Name
		extVersion = pkg.Version

		cacheDir := filepath.Dir(s.silo.PECLArchivePath(extName, extVersion))
		archivePath, err = s.downloader.Download(pkg, cacheDir)
		if err != nil {
			return nil, fmt.Errorf("download pecl package %s: %w", source, err)
		}
	}

	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return nil, fmt.Errorf("get extension manifest: %w", err)
	}
	for _, e := range manifest.Extensions {
		if e.Name == extName {
			return nil, fmt.Errorf("extension %s is already installed for PHP %s", extName, phpVersion)
		}
	}

	extractDir := filepath.Join(prefix, "lib", "pecl", extName)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("create pecl extract dir: %w", err)
	}

	statePath := filepath.Join(extractDir, ".state")
	os.WriteFile(statePath, []byte("in_progress"), 0644)

	cleanup := func() {
		os.WriteFile(statePath, []byte("failed"), 0644)
		os.RemoveAll(extractDir)
	}

	if err := extractArchive(archivePath, extractDir); err != nil {
		cleanup()
		return nil, fmt.Errorf("extract pecl archive: %w", err)
	}

	sourceDir := findSourceDir(extractDir, extName)
	if sourceDir == "" {
		cleanup()
		return nil, fmt.Errorf("could not find extension source in %s (no config.m4 found)", extractDir)
	}

	phpize := filepath.Join(prefix, "bin", "phpize")
	phpConfig := filepath.Join(prefix, "bin", "php-config")

	cmd := exec.CommandContext(ctx, phpize)
	cmd.Dir = sourceDir
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("phpize %s: %w\n%s", extName, err, out)
	}

	configure := exec.CommandContext(ctx, "./configure", "--with-php-config="+phpConfig)
	configure.Dir = sourceDir
	if out, err := configure.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("configure %s: %w\n%s", extName, err, out)
	}

	make := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	make.Dir = sourceDir
	if out, err := make.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("make %s: %w\n%s", extName, err, out)
	}

	install := exec.CommandContext(ctx, "make", "install")
	install.Dir = sourceDir
	if out, err := install.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("make install %s: %w\n%s", extName, err, out)
	}

	iniDir := filepath.Join(prefix, "etc")
	if err := os.MkdirAll(iniDir, 0755); err != nil {
		cleanup()
		return nil, fmt.Errorf("create ini dir: %w", err)
	}
	iniPath := filepath.Join(iniDir, "php.ini")
	entry := "extension=" + extName + ".so\n"
	f, err := os.OpenFile(iniPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("open php.ini: %w", err)
	}
	if _, err := f.WriteString(entry); err != nil {
		f.Close()
		cleanup()
		return nil, fmt.Errorf("write php.ini: %w", err)
	}
	f.Close()

	os.WriteFile(statePath, []byte("installed"), 0644)

	manifest.Extensions = append(manifest.Extensions, domain.ExtensionState{
		Name:        extName,
		Type:        domain.ExtensionTypePECL,
		Version:     extVersion,
		InstalledAt: time.Now(),
	})
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return nil, fmt.Errorf("save extension manifest: %w", err)
	}

	return &InstallResult{
		Name:       extName,
		Version:    extVersion,
		InstallDir: extractDir,
	}, nil
}

func (s *Service) List(phpVersion string) ([]domain.ExtensionState, error) {
	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return nil, err
	}
	var peclExts []domain.ExtensionState
	for _, e := range manifest.Extensions {
		if e.Type == domain.ExtensionTypePECL {
			peclExts = append(peclExts, e)
		}
	}
	return peclExts, nil
}

func (s *Service) Uninstall(name, phpVersion string) error {
	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}

	found := false
	for _, e := range manifest.Extensions {
		if e.Name == name && e.Type == domain.ExtensionTypePECL {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("PECL extension %q is not installed for PHP %s", name, phpVersion)
	}

	prefix := s.silo.PackagePrefix("php", phpVersion)

	extDir := filepath.Join(prefix, "lib", "php", "extensions")
	removed := false
	filepath.Walk(extDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == name+".so" {
			os.Remove(path)
			removed = true
		}
		return nil
	})

	iniPath := filepath.Join(prefix, "etc", "php.ini")
	if data, err := os.ReadFile(iniPath); err == nil {
		lines := strings.Split(string(data), "\n")
		var kept []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "extension="+name+".so" {
				kept = append(kept, line)
			}
		}
		os.WriteFile(iniPath, []byte(strings.Join(kept, "\n")), 0644)
	}

	peclDir := filepath.Join(prefix, "lib", "pecl", name)
	os.RemoveAll(peclDir)

	var remaining []domain.ExtensionState
	for _, e := range manifest.Extensions {
		if e.Name != name {
			remaining = append(remaining, e)
		}
	}
	manifest.Extensions = remaining
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return fmt.Errorf("save extension manifest: %w", err)
	}

	if !removed {
		return fmt.Errorf("extension %q .so file not found at %s", name, extDir)
	}
	return nil
}

func isLocalArchive(path string) bool {
	return strings.HasSuffix(path, ".tgz") ||
		strings.HasSuffix(path, ".tar.gz") ||
		strings.HasSuffix(path, ".tar.bz2")
}
