package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundle"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/graph"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/patcher"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/shim"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/system"
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
	options := []fx.Option{
		fx.NopLogger,
		fx.Provide(
			NewRootCmd,
			fx.Annotate(memory.NewRegistryRepository, fx.As(new(registry.RegistryRepository))),
			fx.Annotate(disk.NewSiloRepository, fx.As(new(silo.SiloRepository))),
			fx.Annotate(disk.NewForgeRepository, fx.As(new(forge.ForgeRepository))),
			fx.Annotate(memory.NewPatcherRepository, fx.As(new(patcher.PatcherRepository))),
			fx.Annotate(memory.NewGraphRepository, fx.As(new(graph.GraphRepository))),
			registry.NewService,
			silo.NewService,
			bundle.NewService,
			system.NewService,
			shim.NewService,
			assembler.NewService,
			forge.NewService,
			patcher.NewService,
			graph.NewService,
		),
		fx.Invoke(
			terminal.NewPHPHandler,
			RegisterRootCmd,
		),
	}

	app := fx.New(options...)

	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := app.Stop(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
