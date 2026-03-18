package main

import (
	"fmt"

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
}
