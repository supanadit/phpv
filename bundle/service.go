package bundle

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
)

type Service struct {
	silo *silo.Service
}

func NewService(s *silo.Service) *Service {
	return &Service{silo: s}
}

func (s *Service) Export(phpVersion, outputPath string) error {
	prefix := s.silo.PackagePrefix("php", phpVersion)
	info, err := os.Stat(filepath.Join(prefix, "bin", "php"))
	if err != nil {
		return err
	}

	manifest := domain.BundleManifest{
		FormatVersion: 1,
		Package:       "php",
		Version:       phpVersion,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		BuildDate:     time.Now(),
		Builder: domain.BundleBuilder{
			PHPVVersion: "1.0.0",
			Compiler:    "gcc",
			Static:      false,
			Libc:        "glibc",
		},
		TotalSize: info.Size(),
	}

	return exportBundle(manifest, prefix, outputPath)
}

func (s *Service) Import(bundlePath, phpVersion string) error {
	return importBundle(s.silo, bundlePath, phpVersion)
}

// ImportFromPath installs a bundle, reading the PHP version from the manifest.
func (s *Service) ImportFromPath(bundlePath string) error {
	manifest, err := readManifest(bundlePath)
	if err != nil {
		return err
	}
	return importBundle(s.silo, bundlePath, manifest.Version)
}

func (s *Service) ImportFromURL(url, phpVersion string) error {
	return importFromURL(s.silo, url, phpVersion)
}
