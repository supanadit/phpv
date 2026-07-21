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

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
)

func importBundle(svc *silo.Service, bundlePath, phpVersion string) error {
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

	// v2+ bundles: enforce arch + libc match.
	if manifest.FormatVersion >= 2 {
		if manifest.Arch != "" && manifest.Arch != runtime.GOARCH {
			return fmt.Errorf("bundle built for %s, cannot install on %s", manifest.Arch, runtime.GOARCH)
		}
		if manifest.Libc != "" {
			hostLibc := detectLibc()
			if manifest.Libc != hostLibc {
				return fmt.Errorf("bundle built for %s libc, host is %s", manifest.Libc, hostLibc)
			}
		}
	}

	if err := svc.MarkComplete("php", phpVersion); err != nil {
		return fmt.Errorf("mark installed: %w", err)
	}

	return nil
}

// detectLibc returns the host libc type ("glibc" or "musl").
func detectLibc() string {
	// Check for /lib/ld-musl-*.so.1 — the musl dynamic linker.
	_, err := os.Stat("/lib/ld-musl-x86_64.so.1")
	if err == nil {
		return "musl"
	}
	_, err = os.Stat("/lib/ld-musl-aarch64.so.1")
	if err == nil {
		return "musl"
	}
	// Also check /usr/lib/ on some distros.
	_, err = os.Stat("/usr/lib/ld-musl-x86_64.so.1")
	if err == nil {
		return "musl"
	}
	return "glibc"
}

func importFromURL(svc *silo.Service, url, phpVersion string) error {
	return fmt.Errorf("import from URL not yet implemented")
}
