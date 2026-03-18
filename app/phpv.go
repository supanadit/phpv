package main

import (
	"fmt"

	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/http"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

func main() {
	sourcePHPRepo := memory.NewPHPRepository()
	sourcePHPSvc := source.NewService(sourcePHPRepo)
	if phps, err := sourcePHPSvc.GetVersions(); err == nil {
		for _, php := range phps {
			fmt.Println(php.URL)
		}
	}

	downloadHTTPRepo := http.NewDownloadRepository()
	downloadHTTPSvc := download.NewService(downloadHTTPRepo)
	archivePath := "/home/supanadit/.phpv/cache/php-4.4.0.tar.gz"
	if _, err := downloadHTTPSvc.Download("https://museum.php.net/php4/php-4.4.0.tar.gz", archivePath); err != nil {
		panic(err)
	}
	fmt.Println("Download completed")

	unloadRepo := disk.NewUnloadRepository()
	unloadSvc := unload.NewService(unloadRepo)
	extractDir := "/home/supanadit/.phpv/extract/php-4.4.0"
	result, err := unloadSvc.Unpack(archivePath, extractDir)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Extracted %d files to %s\n", result.Extracted, result.Destination)
}
