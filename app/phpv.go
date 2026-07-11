package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/silo"
)

func main() {
	// Prepare repositories
	registryRepo := memory.NewRegistryRepository()
	siloRepo := disk.NewSiloRepository()

	// Build service layer
	siloService := silo.NewService(siloRepo, registryRepo)

	// Prepare delivery layer (cobra)
	rootCmd := &cobra.Command{
		Use:   "phpv",
		Short: "PHP version manager",
	}
	terminal.NewPHPHandler(rootCmd, siloService)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}