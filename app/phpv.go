package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/http"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

func phpvPath(parts ...string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = "~"
	}
	return filepath.Join(append([]string{home, ".phpv"}, parts...)...)
}

func main() {
	sourcePHPRepo := memory.NewPHPRepository()
	sourcePHPSvc := source.NewService(sourcePHPRepo)

	phps, err := sourcePHPSvc.GetVersions()
	if err != nil {
		panic(err)
	}
	for _, php := range phps {
		fmt.Println(php.URL)
	}

	downloadHTTPRepo := http.NewDownloadRepository()
	downloadHTTPSvc := download.NewService(downloadHTTPRepo)

	unloadRepo := disk.NewUnloadRepository()
	unloadSvc := unload.NewService(unloadRepo)

	forgeRepo := disk.NewForgeRepository()
	buildRepo := disk.NewBuildRepository()
	forgeSvc := forge.NewService(forgeRepo, buildRepo)

	version := "8.2.0"
	url := fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", version)
	cachePath := phpvPath("cache", fmt.Sprintf("php-%s.tar.gz", version))
	sourcePath := phpvPath("sources", version, "php")
	buildPrefix := forgeSvc.GetBuildPrefix(version)

	if err := os.MkdirAll(phpvPath("cache"), 0o755); err != nil {
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
		if err := os.MkdirAll(sourcePath, 0o755); err != nil {
			panic(err)
		}
		result, err := unloadSvc.Unpack(cachePath, sourcePath)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Extracted %d files to %s\n", result.Extracted, result.Destination)
	} else {
		fmt.Println("Using existing source:", sourcePath)
	}

	flags, ok := forgeSvc.ExpandConfigureFlags(version)
	if !ok {
		panic(fmt.Sprintf("no configure flags for version %s", version))
	}

	fmt.Println("\n=== Building PHP", version, "===")
	fmt.Printf("Source: %s\n", sourcePath)
	fmt.Printf("Prefix: %s\n", buildPrefix)
	fmt.Printf("Flags: %v\n", flags)

	fmt.Println("\n--- Distclean (cleanup) ---")
	forgeSvc.Distclean(sourcePath)

	fmt.Println("\n--- Configure ---")
	if err := forgeSvc.Configure(sourcePath, flags); err != nil {
		panic(fmt.Errorf("configure failed: %w", err))
	}

	fmt.Println("\n--- Make ---")
	jobs := runtime.NumCPU()
	if err := forgeSvc.Make(sourcePath, jobs); err != nil {
		panic(fmt.Errorf("make failed: %w", err))
	}

	fmt.Println("\n--- Install ---")
	if err := forgeSvc.Install(sourcePath); err != nil {
		panic(fmt.Errorf("install failed: %w", err))
	}

	fmt.Println("\n=== Build completed ===")
	fmt.Printf("PHP installed to: %s\n", buildPrefix)
}
