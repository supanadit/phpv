package main

import (
	"context"
	"fmt"
	"os"

	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/extension"
	"github.com/supanadit/phpv/flagresolver"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/config"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/http"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type silentLogger struct{}

func (s *silentLogger) LogEvent(event fxevent.Event) {}

func main() {
	// Initialize config - reads PHPV_ROOT from env or uses default
	_ = config.Get()

	opts := []fx.Option{
		fx.WithLogger(func() fxevent.Logger { return &silentLogger{} }),
		fx.Provide(
			NewSiloRepository,
			NewSourceRepository,
			NewDownloadRepository,
			NewUnloadRepository,
			NewAdvisorRepository,
			NewAssemblerRepository,
			NewForgeRepository,
			NewExtensionRepository,
			NewFlagRepository,
			NewBundlerServiceConfig,
			NewPatternRepository,
		),
		fx.Invoke(run),
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

func NewAdvisorRepository(asm assembler.AssemblerRepository, extRepo extension.Repository) advisor.AdvisorRepository {
	return disk.NewAdvisorRepository(asm, extRepo)
}

func NewAssemblerRepository() assembler.AssemblerRepository {
	return memory.NewMemoryAssemblerRepository()
}

func NewExtensionRepository() extension.Repository {
	return memory.NewExtensionRepository()
}

func NewFlagRepository(extRepo extension.Repository) flagresolver.Repository {
	return memory.NewFlagRepository(extRepo)
}

func NewPatternRepository() pattern.PatternRepository {
	return memory.NewPatternRepository()
}

func NewForgeRepository(dl download.DownloadRepository, ul unload.UnloadRepository, sil *disk.SiloRepository, src source.SourceRepository) forge.ForgeRepository {
	return disk.NewForgeRepository(dl, ul, sil, src, nil)
}

func NewBundlerServiceConfig(
	sil *disk.SiloRepository,
	asm assembler.AssemblerRepository,
	adv advisor.AdvisorRepository,
	fg forge.ForgeRepository,
	dl download.DownloadRepository,
	ul unload.UnloadRepository,
	flagRepo flagresolver.Repository,
	src source.SourceRepository,
) (bundler.BundlerServiceConfig, error) {
	silo, err := sil.GetSilo()
	if err != nil {
		return bundler.BundlerServiceConfig{}, err
	}

	verbose := false
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
			break
		}
	}

	var logger utils.Logger
	logger = utils.NewLogger(utils.LogLevelInfo)

	return bundler.BundlerServiceConfig{
		Assembler:        asm,
		Advisor:          adv,
		Forge:            fg,
		Download:         dl,
		Unload:           ul,
		FlagResolverRepo: flagRepo,
		Source:           src,
		Silo:             silo,
		SiloRepo:         sil,
		Verbose:          verbose,
		Logger:           logger,
	}, nil
}

func run(
	shutdowner fx.Shutdowner,
	sil *disk.SiloRepository,
	cfg bundler.BundlerServiceConfig,
	pattern pattern.PatternRepository,
	src source.SourceRepository,
	ext extension.Repository,
) {
	if err := sil.EnsurePaths(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		shutdowner.Shutdown(fx.ExitCode(1))
		return
	}

	bundlerRepo := disk.NewBundlerRepository(cfg, pattern)
	handler := terminal.NewHandler(bundlerRepo, sil, src, ext)

	if err := terminal.ExecuteCobra(handler, shutdowner); err != nil {
		return
	}

	shutdowner.Shutdown()
}
