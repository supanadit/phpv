package main

import (
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/silo"
)

func main() {
	registryRepo := memory.NewRegistryRepository()
	siloRepo := disk.NewSiloRepository()

	_ = silo.NewService(siloRepo, registryRepo)
}
