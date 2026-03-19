package memory

import (
	"fmt"
	"os"
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
	sourcePHPRepo := NewPHPRepository()
	sourcePHPSvc := source.NewService(sourcePHPRepo)

	phps, err := sourcePHPSvc.GetVersions()
	if err != nil {
		panic(err)
	}
	for _, php := range phps {
		fmt.Println(php.URL)
	}

	downloadHTTPSvc := download.NewService(r.downloadRepository)
	//unloadSvc := unload.NewService(r.unloadRepository)

	url := fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", version)
	cacheDir := filepath.Join(viper.GetString("PHPV_ROOT"), "cache")
	cachePath := filepath.Join(cacheDir, fmt.Sprintf("php-%s.tar.gz", version))
	sourceDir := filepath.Join(viper.GetString("PHPV_ROOT"), "sources")
	sourcePath := filepath.Join(sourceDir, version)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
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
		//os.RemoveAll(sourcePath)

		fmt.Printf("Extracted %d files to: %s\n", result.Extracted, sourceDir)
	} else {
		fmt.Println("Using cached source:", sourceDir)
	}

	return domain.Forge{Prefix: "Haha"}, nil
}
