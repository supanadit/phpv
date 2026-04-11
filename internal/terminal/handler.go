package terminal

import (
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/source"
)

type UseResult struct {
	ExactVersion string
	ShimPath     string
	OutputPath   string
}

type TerminalHandler struct {
	BundlerRepo bundler.BundlerRepository
	Silo        silo.SiloRepository
	Source      source.SourceRepository
}

func NewHandler(
	bundlerRepo bundler.BundlerRepository,
	siloRepo silo.SiloRepository,
	sourceSvc source.SourceRepository,
) *TerminalHandler {
	return &TerminalHandler{
		BundlerRepo: bundlerRepo,
		Silo:        siloRepo,
		Source:      sourceSvc,
	}
}
