package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/patcher"
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

// RegisterRootCmd executes the root command when the app starts.
func RegisterRootCmd(rootCmd *cobra.Command, lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return rootCmd.Execute()
		},
	})
}

func main() {
	// Create Default Options
	options := []fx.Option{
		fx.NopLogger,
		fx.Provide(
			NewRootCmd,
			fx.Annotate(memory.NewRegistryRepository, fx.As(new(registry.RegistryRepository))),
			fx.Annotate(disk.NewSiloRepository, fx.As(new(silo.SiloRepository))),
			fx.Annotate(memory.NewAssemblerRepository, fx.As(new(assembler.AssemblerRepository))),
			fx.Annotate(disk.NewForgeRepository, fx.As(new(forge.ForgeRepository))),
			fx.Annotate(memory.NewPatcherRepository, fx.As(new(patcher.PatcherRepository))),
			silo.NewService,
			assembler.NewService,
		),
		fx.Invoke(
			terminal.NewPHPHandler,
			RegisterRootCmd,
		),
	}

	// Create Inversion of Control
	app := fx.New(options...)

	// Start Context
	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Stop Context
	if err := app.Stop(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}