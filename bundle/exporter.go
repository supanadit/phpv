package bundle

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
)

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
