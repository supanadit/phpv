package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "phpv",
		Short: "PHP version manager",
	}

	app := fx.New(
		fx.NopLogger,
		fx.Provide(
			func() *cobra.Command { return rootCmd },
			fx.Annotate(memory.NewRegistryRepository, fx.As(new(registry.RegistryRepository))),
			fx.Annotate(disk.NewSiloRepository, fx.As(new(silo.SiloRepository))),
			fx.Annotate(silo.NewService, fx.As(new(terminal.PHPService))),
		),
		fx.Invoke(terminal.NewPHPHandler),
	)

	if err := app.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}