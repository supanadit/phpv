package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/version"
)

func main() {
	ctx := context.Background()

	// Configure viper to read environment variables
	viper.AutomaticEnv()
	// Set default PHPV_ROOT to $HOME/.phpv, respecting OS
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("You must have a home directory set for phpv to work")
	}
	viper.SetDefault("PHPV_ROOT", filepath.Join(homeDir, ".phpv"))
	viper.SetDefault("PHP_SOURCE", "github")

	// Register and check help flag
	pflag.BoolP("help", "h", false, "Show help")
	pflag.Parse()
	help, _ := pflag.CommandLine.GetBool("help")
	h, _ := pflag.CommandLine.GetBool("h")
	if help || h {
		terminal.NewNothingHandler()
		return
	}

	versionRepo := memory.NewVersionRepository()
	versionSvc := version.NewService(versionRepo)
	downloadSvc := download.NewService()

	if !terminal.NewDownloadHandler(ctx, versionSvc, downloadSvc) {
		if !terminal.NewVersionHandler(ctx, versionSvc) {
			terminal.NewNothingHandler()
		}
	}
}
