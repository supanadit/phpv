package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
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

type verboseKey struct{}

const (
	minBoxWidth = 64
)

func printBox(width int, lines []string) {
	border := "+" + strings.Repeat("=", width) + "+"
	middle := "+" + strings.Repeat("=", width) + "+"
	bottom := "+" + strings.Repeat("=", width) + "+"

	fmt.Println(border)
	for i, line := range lines {
		if i == 1 {
			fmt.Println(middle)
		}
		padding := width - len(line)
		if padding < 0 {
			padding = 0
		}
		fmt.Println("|" + line + strings.Repeat(" ", padding) + "|")
	}
	fmt.Println(bottom)
}

func main() {
	debugMode := flag.Bool("x", false, "verbose fx logging")
	flag.Bool("v", false, "verbose output (show compile logs)")
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
			NewFlagResolverRepository,
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

func NewFlagResolverRepository() domain.FlagResolverRepository {
	return memory.NewFlagResolverRepository()
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

	verbose := false
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
			break
		}
	}

	return bundler.BundlerServiceConfig{
		Assembler: asm,
		Advisor:   adv,
		Forge:     fg,
		Download:  dl,
		Unload:    ul,
		Source:    src,
		Silo:      silo,
		Verbose:   verbose,
	}, nil
}

func run(
	shutdowner fx.Shutdowner,
	sil *disk.SiloRepository,
	cfg bundler.BundlerServiceConfig,
	flagResolverRepo domain.FlagResolverRepository,
) {
	if err := sil.EnsurePaths(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		shutdowner.Shutdown(fx.ExitCode(1))
		return
	}

	bundlerRepo := disk.NewBundlerRepository(cfg, flagResolverRepo)
	bundlerSvc := bundlerRepo

	version := "8.4.0"
	args := flag.Args()
	if len(args) > 0 {
		version = args[0]
	}

	forge, err := bundlerSvc.Install(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		shutdowner.Shutdown(fx.ExitCode(1))
		return
	}

	binaryPath := forge.Prefix + "/bin/php"
	header := "                    PHP Installation Summary"
	labelVersion := "Version:"
	labelBinary := "Binary:"

	boxWidth := minBoxWidth

	contentWidth := boxWidth - 2

	headerWidth := len(labelVersion) + 1 + len(version)
	binaryWidth := len(labelBinary) + 1 + len(binaryPath)

	if binaryWidth > contentWidth {
		boxWidth = binaryWidth + 2
		contentWidth = boxWidth - 2
	}
	if headerWidth > contentWidth {
		boxWidth = headerWidth + 2
		contentWidth = boxWidth - 2
	}

	displayBinaryPath := binaryPath
	availableBinaryContent := contentWidth - len(labelBinary) - 1
	if len(binaryPath) > availableBinaryContent {
		displayBinaryPath = "..." + binaryPath[len(binaryPath)-availableBinaryContent+3:]
	}

	versionContent := fmt.Sprintf("%s %s", labelVersion, version)
	binaryContent := fmt.Sprintf("%s %s", labelBinary, displayBinaryPath)

	fmt.Println()
	printBox(boxWidth, []string{
		"",
		header,
		versionContent,
		binaryContent,
	})

	shutdowner.Shutdown()
}
