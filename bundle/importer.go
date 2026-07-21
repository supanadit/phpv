package bundle

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/silo"
)

// readManifest reads only the manifest.json from a bundle without extracting
// the full archive. It returns the parsed manifest.
func readManifest(bundlePath string) (*domain.BundleManifest, error) {
	f, err := os.Open(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if hdr.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read manifest: %w", err)
			}
			var manifest domain.BundleManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("parse manifest: %w", err)
			}
			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("bundle missing manifest.json")
}

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

		// Strip the leading "php/" prefix that the exporter adds.
		name := hdr.Name
		name = strings.TrimPrefix(name, "php/")

		target := filepath.Join(svc.PackagePrefix("php", phpVersion), name)
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
			switch manifest.Libc {
			case "musl":
				// musl-static bundles run on any Linux (glibc or musl).
			case "glibc":
				if hostLibc == "musl" {
					return fmt.Errorf("bundle built for glibc, cannot install on musl (Alpine)")
				}
				// Check glibc version compatibility.
				if manifest.GlibcVersion != "" {
					hostVer := detectGlibcVersion()
					if !glibcCompatible(hostVer, manifest.GlibcVersion) {
						return fmt.Errorf("bundle requires glibc >= %s, host has %s", manifest.GlibcVersion, hostVer)
					}
				}
			}
		}
	}

	// Seed extension manifest from ExtPool (v2+).
	if len(manifest.ExtPool) > 0 {
		extManifest := &domain.ExtensionManifest{
			PHPVersion: phpVersion,
		}
		for _, ext := range manifest.ExtPool {
			extManifest.Extensions = append(extManifest.Extensions, domain.ExtensionState{
				Name:          ext.Name,
				Type:          domain.ExtensionTypeBuiltin,
				Version:       ext.Version,
				InstalledAt:   time.Now(),
				SoPath:        filepath.Join("exts", ext.SOFile),
				Prebuilt:      true,
				PhpApiVersion: ext.PhpApiVersion,
			})
		}
		if err := svc.SaveExtensionManifest(phpVersion, extManifest); err != nil {
			return fmt.Errorf("save extension manifest: %w", err)
		}
	}

	// Write toolchain.json for on-demand toolchain download.
	if manifest.Toolchain.URL != "" {
		tcDir := svc.ToolchainPath(manifest.Toolchain.Arch)
		if err := os.MkdirAll(tcDir, 0755); err != nil {
			return fmt.Errorf("create toolchain dir: %w", err)
		}
		tcData, err := json.MarshalIndent(manifest.Toolchain, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal toolchain: %w", err)
		}
		if err := os.WriteFile(filepath.Join(tcDir, "toolchain.json"), tcData, 0644); err != nil {
			return fmt.Errorf("write toolchain.json: %w", err)
		}
	}

	// Write bundle metadata for InstallExtension fast path.
	meta := domain.BundleMeta{
		Libc:          manifest.Libc,
		PhpApiVersion: manifest.PhpApiVersion,
	}
	metaPath := filepath.Join(svc.PackagePrefix("php", phpVersion), ".bundle_meta.json")
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bundle meta: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("write bundle meta: %w", err)
	}

	if err := svc.MarkComplete("php", phpVersion); err != nil {
		return fmt.Errorf("mark installed: %w", err)
	}

	return nil
}

// detectLibc returns the host libc type ("glibc" or "musl").
func detectLibc() string {
	_, err := os.Stat("/lib/ld-musl-x86_64.so.1")
	if err == nil {
		return "musl"
	}
	_, err = os.Stat("/lib/ld-musl-aarch64.so.1")
	if err == nil {
		return "musl"
	}
	_, err = os.Stat("/usr/lib/ld-musl-x86_64.so.1")
	if err == nil {
		return "musl"
	}
	return "glibc"
}

// detectGlibcVersion returns the host glibc version as "major.minor" or "".
func detectGlibcVersion() string {
	out, err := exec.Command("ldd", "--version").Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			out = ee.Stderr
		} else {
			return ""
		}
	}
	return parseGlibcVersion(string(out))
}

// parseGlibcVersion extracts "major.minor" from ldd --version output.
func parseGlibcVersion(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "GLIBC") || strings.Contains(line, "GNU libc") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if matched, _ := regexp.MatchString(`^\d+\.\d+`, p); matched {
					return p
				}
			}
		}
	}
	return ""
}

// glibcCompatible checks if host glibc >= bundle glibc.
func glibcCompatible(hostVer, bundleVer string) bool {
	if hostVer == "" || bundleVer == "" {
		return true
	}
	hMajor, hMinor := splitGlibcVersion(hostVer)
	bMajor, bMinor := splitGlibcVersion(bundleVer)
	if hMajor == 0 && hMinor == 0 {
		return true
	}
	if bMajor == 0 && bMinor == 0 {
		return true
	}
	if hMajor > bMajor {
		return true
	}
	if hMajor == bMajor && hMinor >= bMinor {
		return true
	}
	return false
}

func splitGlibcVersion(v string) (int, int) {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	return major, minor
}

func importFromURL(svc *silo.Service, url, phpVersion string) error {
	return fmt.Errorf("import from URL not yet implemented")
}
