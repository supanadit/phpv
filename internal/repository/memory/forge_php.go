package memory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type ForgePHP struct {
	downloadRepository download.DownloadRepository
	unloadRepository   unload.UnloadRepository
}

func NewForgePHP() *ForgePHP {
	return &ForgePHP{}
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
	//sourceDir := filepath.Join(viper.GetString("PHPV_ROOT"), "source")

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		panic(err)
	}

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if _, err := downloadHTTPSvc.Download(url, cachePath); err != nil {
			panic(err)
		}
		fmt.Println("Download completed:", cachePath)
	} else {
		fmt.Println("Using cached:", cachePath)
	}
	return domain.Forge{Prefix: "Haha"}, nil
}
