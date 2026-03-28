package memory

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/http"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type ForgeRepository struct {
	downloadRepository download.DownloadRepository
	unloadRepository   unload.UnloadRepository
}

func NewForgeRepository() *ForgeRepository {
	return &ForgeRepository{
		downloadRepository: http.NewDownloadRepository(),
		unloadRepository:   disk.NewUnloadRepository(),
	}
}

func (r *ForgeRepository) Build(version string) (domain.Forge, error) {
	sourceRepository := NewSourceRepository()
	sourceService := source.NewService(sourceRepository)

	phps, err := sourceService.GetVersions()
	if err != nil {
		panic(err)
	}

	downloadHTTPSvc := download.NewService(r.downloadRepository)
	unloadSvc := unload.NewService(r.unloadRepository)

	var url string
	for _, src := range phps {
		if src.Name == "php" && src.Version == version {
			url = src.URL
			break
		}
	}
	if url == "" {
		return domain.Forge{}, fmt.Errorf("source not found for version %s", version)
	}
	cacheDir := filepath.Join(viper.GetString("PHPV_ROOT"), "cache")
	cachePath := filepath.Join(cacheDir, fmt.Sprintf("php-%s.tar.gz", version))
	sourceDir := filepath.Join(viper.GetString("PHPV_ROOT"), "sources")
	sourcePath := filepath.Join(sourceDir, version)

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		panic(err)
	}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		if _, err := downloadHTTPSvc.Download(url, cachePath); err != nil {
			panic(err)
		}
		fmt.Println("Download completed:", cachePath)
	} else {
		fmt.Println("Using cached:", cachePath)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		result, err := unloadSvc.Unpack(cachePath, sourcePath)
		if err != nil {
			panic(err)
		}

		entries, _ := os.ReadDir(sourcePath)
		if len(entries) == 1 && entries[0].IsDir() {
			extractedFolder := filepath.Join(sourcePath, entries[0].Name())
			files, _ := os.ReadDir(extractedFolder)
			for _, f := range files {
				os.Rename(filepath.Join(extractedFolder, f.Name()), filepath.Join(sourcePath, f.Name()))
			}
			os.RemoveAll(extractedFolder)
		}

		fmt.Printf("Extracted %d files to: %s\n", result.Extracted, sourcePath)
	} else {
		fmt.Println("Using cached source:", sourcePath)
	}

	versionsDir := filepath.Join(viper.GetString("PHPV_ROOT"), "versions")
	versionsPath := filepath.Join(versionsDir, version)
	if err := os.MkdirAll(versionsDir, 0o755); err != nil {
		panic(err)
	}

	err = os.Chmod(filepath.Join(sourcePath, "configure"), 0o755)
	if err != nil {
		log.Fatal(err)
	}

	configure := exec.Command("./configure", fmt.Sprintf("--prefix=%s", versionsPath))
	configure.Dir = sourcePath
	configure.Stdout = os.Stdout
	configure.Stderr = os.Stderr

	fmt.Println("Starting configure...")
	if err := configure.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
	}

	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "build")).Run()
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "ext")).Run()

	fmt.Println("Path Version", versionsPath)
	// Make Process
	// 1. Define the command (e.g., ./configure)
	mk1 := exec.Command("/usr/bin/make", "-j16")

	// 2. Set the working directory
	mk1.Dir = sourcePath

	// 4. Pipe output so you can see the build progress
	mk1.Stdout = os.Stdout
	mk1.Stderr = os.Stderr

	// 5. Run it
	fmt.Println("Starting make...")
	if err := mk1.Run(); err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	// Make Install
	// 1. Define the command (e.g., ./configure)
	mk2 := exec.Command("/usr/bin/make", "-j16", "install")

	// 2. Set the working directory
	mk2.Dir = sourcePath

	// 4. Pipe output so you can see the build progress
	mk2.Stdout = os.Stdout
	mk2.Stderr = os.Stderr

	// 5. Run it
	fmt.Println("Starting make install...")
	if err := mk2.Run(); err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	return domain.Forge{Prefix: versionsPath}, nil
}
