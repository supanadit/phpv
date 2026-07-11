package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

// NewRootCmd provides the root cobra command.
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "phpv",
		Short: "PHP version manager",
	}
}

func main() {
	rootCmd := NewRootCmd()

	// Create Default Options
	options := []fx.Option{
		fx.NopLogger,
		fx.Provide(
			func() *cobra.Command { return rootCmd },
			fx.Annotate(memory.NewRegistryRepository, fx.As(new(registry.RegistryRepository))),
			fx.Annotate(disk.NewSiloRepository, fx.As(new(silo.SiloRepository))),
			silo.NewService,
		),
		fx.Invoke(terminal.NewPHPHandler),
	}

	// Create Inversion of Control
	app := fx.New(options...)

	// Start Context
	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Stop Context
	if err := app.Stop(context.Background()); err != nil {
		log.Fatal(err)
	}
}