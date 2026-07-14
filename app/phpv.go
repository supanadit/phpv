package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundle"
	"github.com/supanadit/phpv/config"
	"github.com/supanadit/phpv/doctor"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/graph"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/shutdown"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/patcher"
	"github.com/supanadit/phpv/pecl"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/shim"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/system"
	"github.com/supanadit/phpv/update"
)

var Version = "dev"

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "phpv",
		Short: "PHP version manager",
	}
}

func RegisterRootCmd(rootCmd *cobra.Command, lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return rootCmd.Execute()
		},
	})
}

func main() {
	mgr := shutdown.New(shutdown.DefaultSignals...)
	shutdownCtx := mgr.Context()

	go func() {
		sig := mgr.Wait()
		shutdown.PrintInterrupted(sig)
	}()

	options := []fx.Option{
		fx.NopLogger,
		fx.Supply(Version),
		fx.Supply(shutdownCtx),
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
			pecl.NewService,
			fx.Annotate(disk.NewConfigRepository, fx.As(new(config.ConfigRepository))),
			config.NewService,
			fx.Annotate(disk.NewDoctorRepository, fx.As(new(doctor.Repository))),
			doctor.NewService,
			fx.Annotate(disk.NewUpdateRepository, fx.As(new(update.Repository))),
			func(repo update.Repository) *update.Service {
				return update.NewService(repo, Version)
			},
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
		mgr.Stop()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mgr.Stop()

	if err := app.Stop(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
