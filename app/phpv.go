package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/http"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type silentLogger struct{}

func (s *silentLogger) LogEvent(event fxevent.Event) {}

func main() {
	debugMode := flag.Bool("x", false, "verbose fx logging")
	flag.Parse()

	viper.AutomaticEnv()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("You must have a home directory set for phpv to work")
	}
	viper.SetDefault("PHPV_ROOT", filepath.Join(homeDir, ".phpv"))

	opts := []fx.Option{
		fx.Provide(
			NewSiloRepository,
			NewSourceRepository,
			NewDownloadRepository,
			NewUnloadRepository,
			NewAdvisorRepository,
			NewAssemblerRepository,
			NewForgeRepository,
			NewBundlerServiceConfig,
		),
		fx.Invoke(run),
	}

	if !*debugMode {
		opts = append(opts, fx.WithLogger(func() fxevent.Logger { return &silentLogger{} }))
	}

	app := fx.New(opts...)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting app: %v\n", err)
		os.Exit(1)
	}

	<-app.Done()
}

type Silos struct {
	fx.In
	Silo *disk.SiloRepository
}

type Sources struct {
	fx.In
	Source source.SourceRepository
}

type Downloads struct {
	fx.In
	Download download.DownloadRepository
}

type Unloads struct {
	fx.In
	Unload unload.UnloadRepository
}

type Advisors struct {
	fx.In
	Advisor advisor.AdvisorRepository
}

type Assemblers struct {
	fx.In
	Assembler assembler.AssemblerRepository
}

type Forges struct {
	fx.In
	Forge forge.ForgeRepository
}

func NewSiloRepository() (*disk.SiloRepository, error) {
	return disk.NewSiloRepository()
}

func NewSourceRepository() source.SourceRepository {
	return memory.NewSourceRepository()
}

func NewDownloadRepository() download.DownloadRepository {
	return http.NewDownloadRepository()
}

func NewUnloadRepository() unload.UnloadRepository {
	return disk.NewUnloadRepository()
}

func NewAdvisorRepository() advisor.AdvisorRepository {
	return disk.NewAdvisorRepository()
}

func NewAssemblerRepository() assembler.AssemblerRepository {
	return memory.NewMemoryAssemblerRepository()
}

func NewForgeRepository(dl download.DownloadRepository, ul unload.UnloadRepository, sil *disk.SiloRepository, src source.SourceRepository) forge.ForgeRepository {
	return disk.NewForgeRepository(dl, ul, sil, src)
}

func NewBundlerServiceConfig(
	sil *disk.SiloRepository,
	asm assembler.AssemblerRepository,
	adv advisor.AdvisorRepository,
	fg forge.ForgeRepository,
	dl download.DownloadRepository,
	ul unload.UnloadRepository,
	src source.SourceRepository,
) (bundler.BundlerServiceConfig, error) {
	silo, err := sil.GetSilo()
	if err != nil {
		return bundler.BundlerServiceConfig{}, err
	}

	return bundler.BundlerServiceConfig{
		Assembler: asm,
		Advisor:   adv,
		Forge:     fg,
		Download:  dl,
		Unload:    ul,
		Source:    src,
		Silo:      silo,
	}, nil
}

func run(
	lifecycle fx.Lifecycle,
	sil *disk.SiloRepository,
	cfg bundler.BundlerServiceConfig,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := sil.EnsurePaths(); err != nil {
				return fmt.Errorf("failed to ensure paths: %w", err)
			}

			bundlerRepo := disk.NewBundlerRepository(cfg)
			bundlerSvc := bundlerRepo

			version := "8.4.0"
			if len(os.Args) > 1 {
				version = os.Args[1]
			}

			fmt.Printf("Installing PHP %s...\n", version)
			forge, err := bundlerSvc.Install(version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return nil
			}

			fmt.Printf("\n✅ PHP installed successfully!\n")
			fmt.Printf("   Prefix: %s\n", forge.Prefix)
			for k, v := range forge.Env {
				fmt.Printf("   %s: %s\n", k, v)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
}
