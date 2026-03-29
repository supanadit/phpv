package bundler

import (
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type BundlerRepository interface {
	Install(version string) (domain.Forge, error)
	Orchestrate(name, exactVersion string) (domain.Forge, error)
}

type BundlerServiceConfig struct {
	Assembler assembler.AssemblerRepository
	Advisor   advisor.AdvisorRepository
	Forge     forge.ForgeRepository
	Download  download.DownloadRepository
	Unload    unload.UnloadRepository
	Source    source.SourceRepository
	Silo      *domain.Silo
	Jobs      int
	Verbose   bool
}
