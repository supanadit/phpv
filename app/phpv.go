package main

import (
	"fmt"

	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/internal/repository/http"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/source"
)

func main() {
	sourcePHPRepo := memory.NewSourceRepository()
	sourcePHPSvc := source.NewService(sourcePHPRepo)
	if phps, err := sourcePHPSvc.GetVersions(); err == nil {
		for _, php := range phps {
			fmt.Println(php.URL)
		}
	}
	downloadHTTPRepo := http.NewDownloadRepository()
	downloadHTTPSvc := download.NewService(downloadHTTPRepo)
	df, de := downloadHTTPSvc.Download("https://museum.php.net/php4/php-4.4.0.tar.gz", "/home/supanadit/.phpv/cache/php-4.4.0.tar.gz")
	if de != nil {
		panic(de)
	}
	fmt.Println(df.Status)
}
