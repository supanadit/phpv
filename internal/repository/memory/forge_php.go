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

type ForgePHP struct {
	downloadRepository download.DownloadRepository
	unloadRepository   unload.UnloadRepository
}

func NewForgePHP() *ForgePHP {
	return &ForgePHP{
		downloadRepository: http.NewDownloadRepository(),
		unloadRepository:   disk.NewUnloadRepository(),
	}
}

func (r *ForgePHP) Build(version string) (domain.Forge, error) {
	sourcePHPRepo := NewSourceRepository()
	sourcePHPSvc := source.NewService(sourcePHPRepo)

	phps, err := sourcePHPSvc.GetVersions()
	if err != nil {
		panic(err)
	}
	for _, php := range phps {
		fmt.Println(php.URL)
	}

	downloadHTTPSvc := download.NewService(r.downloadRepository)
	// unloadSvc := unload.NewService(r.unloadRepository)

	url := fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", version)
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
		unloadSvc := unload.NewService(r.unloadRepository)
		result, err := unloadSvc.Unpack(cachePath, sourcePath)
		if err != nil {
			panic(err)
		}

		// Find the extracted folder and move it to sourceDir
		entries, _ := os.ReadDir(sourcePath)
		extractedFolder := filepath.Join(sourcePath, entries[0].Name())
		os.Rename(extractedFolder, sourceDir)

		fmt.Printf("Extracted %d files to: %s\n", result.Extracted, sourceDir)
	} else {
		fmt.Println("Using cached source:", sourceDir)
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

	//// TODO: We need to know wether it's already configured or not. Multiple time running waste time
	//// Configure Process
	//// 1. Define the command (e.g., ./configure)
	//configure := exec.Command("./configure", fmt.Sprintf("--prefix=%s", versionsPath))
	//
	//// 2. Set the working directory
	//configure.Dir = sourcePath
	//
	//// 4. Pipe output so you can see the build progress
	//configure.Stdout = os.Stdout
	//configure.Stderr = os.Stderr
	//
	//// 5. Run it
	//fmt.Println("Starting configure...")
	//if err := configure.Run(); err != nil {
	//	fmt.Printf("Error: %s\n", err)
	//}

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
