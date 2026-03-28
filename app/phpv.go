package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/repository/memory"
)

func main() {
	// Configure viper to read environment variables
	viper.AutomaticEnv()
	// Set default PHPV_ROOT to $HOME/.phpv, respecting OS
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("You must have a home directory set for phpv to work")
	}
	viper.SetDefault("PHPV_ROOT", filepath.Join(homeDir, ".phpv"))

	repo := memory.NewForgeRepository()
	svc := forge.NewService(repo)

	// svc.Build(domain.ForgeConfig{Name: "php", Version: "8.5.4"})
	// svc.Build(domain.ForgeConfig{Name: "php", Version: "7.4.33"})
	svc.Build(domain.ForgeConfig{Name: "php", Version: "8.0.0"})
}
