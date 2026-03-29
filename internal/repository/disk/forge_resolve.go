package disk

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

func (r *ForgeRepository) resolveURL(name, version string) (string, error) {
	sourceSvc := source.NewService(r.sourceRepo)

	sources, err := sourceSvc.GetVersions()
	if err != nil {
		return "", err
	}

	for _, src := range sources {
		if src.Name == name && src.Version == version {
			return src.URL, nil
		}
	}

	return "", fmt.Errorf("source not found for %s version %s", name, version)
}

func (r *ForgeRepository) ensureSource(name, version, url string) error {
	silo, err := r.siloRepo.GetSilo()
	if err != nil {
		return err
	}

	filename := filepath.Base(url)
	cachePath := filepath.Join(silo.Root, "cache", name, version, filename)

	cacheExists, _ := afero.Exists(r.fs, cachePath)
	if !cacheExists {
		cacheDir := filepath.Dir(cachePath)
		if err := r.fs.MkdirAll(cacheDir, 0o755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}

		downloadSvc := download.NewService(r.downloadRepo)
		if _, err := downloadSvc.Download(url, cachePath); err != nil {
			return fmt.Errorf("failed to download %s: %w", url, err)
		}
		fmt.Println("Downloaded:", cachePath)
	} else {
		fmt.Println("Using cached:", cachePath)
	}

	sourceDir := utils.GetSourceDirPath(silo, name, version)
	sourceExists, _ := afero.Exists(r.fs, sourceDir)

	needsFlatten := false
	extractedFolderName := ""

	if sourceExists {
		entries, _ := afero.ReadDir(r.fs, sourceDir)
		for _, e := range entries {
			if e.IsDir() {
				needsFlatten = true
				extractedFolderName = e.Name()
				break
			}
		}
	}

	if !sourceExists || needsFlatten {
		if !sourceExists {
			sourceBaseDir := filepath.Dir(sourceDir)
			if err := r.fs.MkdirAll(sourceBaseDir, 0o755); err != nil {
				return fmt.Errorf("failed to create source directory: %w", err)
			}
		}

		extractedFolder := ""
		if needsFlatten {
			extractedFolder = filepath.Join(sourceDir, extractedFolderName)
		}

		if needsFlatten && !r.isProperlyFlattened(sourceDir, extractedFolder) {
			if err := r.fs.RemoveAll(sourceDir); err != nil {
				return fmt.Errorf("failed to clean up incomplete source directory: %w", err)
			}
			sourceExists = false
			needsFlatten = false
			extractedFolderName = ""
			if err := r.fs.MkdirAll(sourceDir, 0o755); err != nil {
				return fmt.Errorf("failed to create source directory: %w", err)
			}
		}

		if !needsFlatten || extractedFolder == "" {
			unloadSvc := unload.NewService(r.unloadRepo)
			if _, err := unloadSvc.Unpack(cachePath, sourceDir); err != nil {
				return fmt.Errorf("failed to extract %s: %w", cachePath, err)
			}

			entries, _ := afero.ReadDir(r.fs, sourceDir)
			for _, e := range entries {
				if e.IsDir() {
					extractedFolder = filepath.Join(sourceDir, e.Name())
					extractedFolderName = e.Name()
					break
				}
			}
			fmt.Printf("Extattened to: %s\n", sourceDir)
		}

		if extractedFolder != "" && extractedFolderName != "" {
			if err := r.flattenSource(sourceDir, extractedFolder, extractedFolderName); err != nil {
				return fmt.Errorf("failed to flatten source: %w", err)
			}
		}
	} else {
		fmt.Println("Using cached source:", sourceDir)
	}

	return nil
}

func (r *ForgeRepository) isProperlyFlattened(sourceDir, extractedFolder string) bool {
	if extractedFolder == "" {
		return true
	}
	extractedEntries, err := afero.ReadDir(r.fs, extractedFolder)
	if err != nil {
		return false
	}
	for _, e := range extractedEntries {
		dstPath := filepath.Join(sourceDir, e.Name())
		if _, err := r.fs.Stat(dstPath); err == nil {
			return false
		}
	}
	return true
}

func (r *ForgeRepository) flattenSource(sourceDir, extractedFolder, extractedFolderName string) error {
	extractedEntries, _ := afero.ReadDir(r.fs, extractedFolder)
	for _, f := range extractedEntries {
		src := filepath.Join(extractedFolder, f.Name())
		dst := filepath.Join(sourceDir, f.Name())
		dstInfo, dstErr := r.fs.Stat(dst)
		if f.IsDir() && dstErr == nil && dstInfo.IsDir() {
			if err := r.mergeDirectories(src, dst); err != nil {
				return fmt.Errorf("failed to merge directory %s: %w", f.Name(), err)
			}
		} else if err := r.fs.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move extracted files: %w", err)
		}
	}
	r.fs.Remove(extractedFolder)
	fmt.Printf("Flattened to: %s\n", sourceDir)
	return nil
}

func (r *ForgeRepository) ensureFs() {
	if r.fs == nil {
		r.fs = afero.NewOsFs()
	}
}

func (r *ForgeRepository) mergeDirectories(src, dst string) error {
	entries, err := afero.ReadDir(r.fs, src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			dstInfo, dstErr := r.fs.Stat(dstPath)
			if dstErr == nil && dstInfo.IsDir() {
				if err := r.mergeDirectories(srcPath, dstPath); err != nil {
					return err
				}
			} else {
				if err := r.fs.Rename(srcPath, dstPath); err != nil {
					return err
				}
			}
		} else {
			if err := r.fs.Rename(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return r.fs.Remove(src)
}
