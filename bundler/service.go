package bundler

import (
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/extension"
	"github.com/supanadit/phpv/flagresolver"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type BundlerRepository interface {
	Install(version string, compiler string, extensions []string, fresh bool) (domain.Forge, error)
	Rebuild(version string, compiler string, extensions []string) (domain.Forge, error)
	Orchestrate(name, exactVersion string, compiler string, extensions []string, fresh bool) (domain.Forge, error)
	PECLInstall(archivePath string, phpVersion string) (*domain.Extension, error)
	PECLList(phpVersion string) ([]string, error)
	PECLUninstall(name string, phpVersion string) error
}

type BundlerServiceConfig struct {
	Assembler     assembler.AssemblerRepository
	Advisor       advisor.AdvisorRepository
	Forge         forge.ForgeRepository
	Download      download.DownloadRepository
	Unload        unload.UnloadRepository
	ExtensionRepo extension.Repository
	Source        source.SourceRepository
	Silo          *domain.Silo
	SiloRepo      silo.SiloRepository
	Jobs          int
	Verbose       bool
	Logger        utils.Logger
	Extensions    []string
}

type Service struct {
	assemblerSvc    *assembler.AssemblerService
	advisorSvc      *advisor.Service
	forgeSvc        *forge.Service
	downloadSvc     *download.Service
	unloadSvc       *unload.Service
	sourceSvc       *source.Service
	flagResolverSvc *flagresolver.Service
	silo            *domain.Silo
	siloRepo        silo.SiloRepository
	logger          utils.Logger
}

func NewService(cfg BundlerServiceConfig, extRepo extension.Repository) *Service {
	assemblerSvc := assembler.NewAssemblerServiceWithRepo(cfg.Assembler)
	advisorSvc := advisor.NewAdvisorService(cfg.Advisor)
	flagResolverSvc := flagresolver.NewService(extRepo)
	forgeSvc := forge.NewService(cfg.Forge, flagResolverSvc)
	downloadSvc := download.NewService(cfg.Download)
	unloadSvc := unload.NewService(cfg.Unload)
	sourceSvc := source.NewService(cfg.Source)

	return &Service{
		assemblerSvc:    assemblerSvc,
		advisorSvc:      advisorSvc,
		forgeSvc:        forgeSvc,
		downloadSvc:     downloadSvc,
		unloadSvc:       unloadSvc,
		sourceSvc:       sourceSvc,
		flagResolverSvc: flagResolverSvc,
		silo:            cfg.Silo,
		siloRepo:        cfg.SiloRepo,
		logger:          cfg.Logger,
	}
}
