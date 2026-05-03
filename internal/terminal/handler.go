package terminal

import (
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/extension"
	"github.com/supanadit/phpv/internal/compiler"
	"github.com/supanadit/phpv/internal/config"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/source"
)

type UseResult struct {
	ExactVersion string
	ShimPath     string
	OutputPath   string
}

type ExtensionsResult struct {
	Extensions []domain.ExtensionInfo
	PHPVersion string
}

type TerminalHandler struct {
	BundlerRepo   bundler.BundlerRepository
	Silo          silo.SiloRepository
	Source        source.SourceRepository
	ExtensionRepo extension.Repository
	Compiler      *compiler.CompilerService
}

func NewHandler(
	bundlerRepo bundler.BundlerRepository,
	siloRepo silo.SiloRepository,
	sourceSvc source.SourceRepository,
	extRepo extension.Repository,
) *TerminalHandler {
	return &TerminalHandler{
		BundlerRepo:   bundlerRepo,
		Silo:          siloRepo,
		Source:        sourceSvc,
		ExtensionRepo: extRepo,
		Compiler:      compiler.NewCompilerService(config.Get().RootDir()),
	}
}
