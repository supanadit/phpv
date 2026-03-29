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
	var extractedFolderName string
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

		if needsFlatten {
			extractedFolder := filepath.Join(sourceDir, extractedFolderName)
			extractedEntries, _ := afero.ReadDir(r.fs, extractedFolder)
			for _, f := range extractedEntries {
				src := filepath.Join(extractedFolder, f.Name())
				dst := filepath.Join(sourceDir, f.Name())
				if err := r.fs.Rename(src, dst); err != nil {
					return fmt.Errorf("failed to move extracted files: %w", err)
				}
			}
			r.fs.Remove(extractedFolder)
			fmt.Printf("Flattened to: %s\n", sourceDir)
		} else {
			unloadSvc := unload.NewService(r.unloadRepo)
			if _, err := unloadSvc.Unpack(cachePath, sourceDir); err != nil {
				return fmt.Errorf("failed to extract %s: %w", cachePath, err)
			}

			entries, _ := afero.ReadDir(r.fs, sourceDir)
			for _, e := range entries {
				if e.IsDir() {
					extractedFolder := filepath.Join(sourceDir, e.Name())
					extractedEntries, _ := afero.ReadDir(r.fs, extractedFolder)
					for _, f := range extractedEntries {
						src := filepath.Join(extractedFolder, f.Name())
						dst := filepath.Join(sourceDir, f.Name())
						if err := r.fs.Rename(src, dst); err != nil {
							return fmt.Errorf("failed to move extracted files: %w", err)
						}
					}
					r.fs.Remove(extractedFolder)
					break
				}
			}
			fmt.Printf("Extracted to: %s\n", sourceDir)
		}
	} else {
		fmt.Println("Using cached source:", sourceDir)
	}

	return nil
}

func (r *ForgeRepository) ensureFs() {
	if r.fs == nil {
		r.fs = afero.NewOsFs()
	}
}
