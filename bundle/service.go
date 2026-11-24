package bundle

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/supanadit/phpv/domain"
)

// BundleSilo is the storage capability bundle consumes. It is declared here
// so app/main.go can inject the concrete silo service without a module import.
//
//go:generate mockery --name BundleSilo
type BundleSilo interface {
	PackagePrefix(name, version string) string
	MarkComplete(name, version string) error
}

type Service struct {
	silo BundleSilo
}

func NewService(s BundleSilo) *Service {
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

func (s *Service) ImportFromURL(url, phpVersion string) error {
	return importFromURL(s.silo, url, phpVersion)
}

func exportBundle(manifest domain.BundleManifest, prefix, outputPath string) error {
	if outputPath == "" {
		outputPath = fmt.Sprintf("php-%s-%s-%s.tar.gz", manifest.Version, manifest.OS, manifest.Arch)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	if err := addManifest(tw, manifest); err != nil {
		return fmt.Errorf("add manifest: %w", err)
	}

	if err := addDir(tw, prefix, "php"); err != nil {
		return fmt.Errorf("add php dir: %w", err)
	}

	return nil
}

func addManifest(tw *tar.Writer, manifest domain.BundleManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:     "manifest.json",
		Mode:     0644,
		Size:     int64(len(data)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = tw.Write(data)
	return err
}

func addDir(tw *tar.Writer, src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		name := filepath.Join(dst, rel)
		if info.IsDir() {
			hdr := &tar.Header{
				Name:     name + "/",
				Mode:     int64(info.Mode().Perm()),
				Typeflag: tar.TypeDir,
			}
			return tw.WriteHeader(hdr)
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = name
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

func importBundle(svc BundleSilo, bundlePath, phpVersion string) error {
	f, err := os.Open(bundlePath)
	if err != nil {
		return fmt.Errorf("open bundle: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var manifest domain.BundleManifest
	found := false

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		if hdr.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("parse manifest: %w", err)
			}
			found = true
			continue
		}

		target := filepath.Join(svc.PackagePrefix("php", phpVersion), hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
		}

		out, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("create %s: %w", target, err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return fmt.Errorf("write %s: %w", target, err)
		}
		out.Close()

		if hdr.Mode != 0 {
			os.Chmod(target, os.FileMode(hdr.Mode))
		}
	}

	if !found {
		return fmt.Errorf("bundle missing manifest.json")
	}

	if manifest.OS != "" && manifest.OS != "linux" {
		return fmt.Errorf("bundle built for %s, cannot install on linux", manifest.OS)
	}

	if err := svc.MarkComplete("php", phpVersion); err != nil {
		return fmt.Errorf("mark installed: %w", err)
	}

	return nil
}

func importFromURL(svc BundleSilo, url, phpVersion string) error {
	return fmt.Errorf("import from URL not yet implemented")
}
