package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundle"
	"github.com/supanadit/phpv/config"
	"github.com/supanadit/phpv/doctor"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/editor"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/graph"
	"github.com/supanadit/phpv/internal/repository"
	"github.com/supanadit/phpv/internal/repository/disk"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/internal/terminal/args"
	"github.com/supanadit/phpv/launchers"
	"github.com/supanadit/phpv/mirror"
	"github.com/supanadit/phpv/patcher"
	"github.com/supanadit/phpv/pecl"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/system"
	"github.com/supanadit/phpv/update"
	"github.com/supanadit/phpv/webserver"
)

var version = "dev"

// NewPatchesRepository builds the file-based patch repository from the
// two PHPV_ROOT patch trees: synced (managed by `phpv patch update`) and
// user-local (never touched by sync), user-local taking precedence.
func NewPatchesRepository() *disk.PatchRepository {
	root := repository.ResolveRoot()
	return disk.NewPatchRepository(
		filepath.Join(root, "patches"),
		filepath.Join(root, "patches-local"),
	)
}

// NewPatchLister exposes the on-disk patch tree to the terminal channel.
func NewPatchLister() args.PatchLister {
	root := repository.ResolveRoot()
	return disk.NewPatchRepository(
		filepath.Join(root, "patches"),
		filepath.Join(root, "patches-local"),
	)
}

// NewPatchSyncService targets the managed patch tree at PHPV_ROOT/patches/
// with the publication remote (this repository's main branch).
func NewPatchSyncService() *mirror.Service {
	root := repository.ResolveRoot()
	return mirror.NewService(disk.NewPatchRemote(), filepath.Join(root, "patches"), 0)
}

func RegisterRootCmd(rootCmd *cobra.Command, lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return rootCmd.Execute()
		},
	})
}

// terminalDeps is the parameter object for wiring the terminal channel. It
// keeps the composition root's closure under the two-parameter limit.
type terminalDeps struct {
	fx.In

	RootCmd     *cobra.Command
	Lifecycle   fx.Lifecycle
	Silo        *silo.Service
	Assembler   *assembler.Service
	Registry    *registry.Service
	Bundle      *bundle.Service
	System      *system.Service
	Shim        *launchers.Service
	PECL        *pecl.Service
	Config      *config.Service
	Doctor      *doctor.Service
	Update      *update.Service
	PatchSync   *mirror.Service
	PatchLister args.PatchLister
}

// serverManagerAdapter maps the concrete webserver manager onto the
// consumer-owned interface declared by the terminal channel.
type serverManagerAdapter struct{ m *webserver.Manager }

func (a serverManagerAdapter) Start(ctx context.Context, opts domain.ServerRunOptions) (domain.ServerInstance, error) {
	return a.m.Start(ctx, opts)
}
func (a serverManagerAdapter) Stop(inst domain.ServerInstance) error { return a.m.Stop(inst) }
func (a serverManagerAdapter) StopByID(id string) (domain.ServerInstance, error) {
	return a.m.StopByID(id)
}
func (a serverManagerAdapter) List() ([]domain.ServerInstance, error) { return a.m.List() }
func (a serverManagerAdapter) Running() ([]domain.ServerInstance, error) {
	return a.m.Running()
}
func (a serverManagerAdapter) Prune() (int, error)     { return a.m.Prune() }
func (a serverManagerAdapter) Store() args.ServerStore { return a.m.Store() }

// NewServerManagerFactory adapts delivery-supplied callbacks to the webserver module.
func NewServerManagerFactory() args.ServerManagerFactory {
	return func(deps args.ServerManagerDeps) args.ServerManager {
		return serverManagerAdapter{m: webserver.NewManager(webserver.Deps{
			Root:                   deps.Root,
			ResolveVersion:         deps.ResolveVersion,
			EnsureFPM:              deps.EnsureFPM,
			EnsureApacheModule:     deps.EnsureApacheModule,
			CLIPrefix:              deps.CLIPrefix,
			InstalledServerVersion: deps.InstalledServerVersion,
			ServerPrefix:           deps.ServerPrefix,
		})}
	}
}

// server service providers: the terminal channel consumes these through
// consumer-owned interfaces rather than importing concrete modules.
func NewFPMServiceForServer(a *assembler.Service) *assembler.FPMService {
	return assembler.NewFPMService(a, repository.ResolveRoot())
}

func NewApacheServiceForServer(a *assembler.Service) *assembler.ApacheService {
	return assembler.NewApacheService(a, repository.ResolveRoot())
}

// assemblerDepsIn collects the concrete services the assembler composes.
type assemblerDepsIn struct {
	fx.In

	Graph    *graph.Service
	Silo     *silo.Service
	Forge    *forge.Service
	Patcher  *patcher.Service
	Registry *registry.Service
	Mirror   *mirror.Service
}

func NewAssemblerDeps(in assemblerDepsIn) assembler.ServiceDeps {
	return assembler.ServiceDeps{
		Graph: in.Graph, Silo: in.Silo, Forge: in.Forge,
		Patcher: in.Patcher, Registry: in.Registry, Mirror: in.Mirror,
	}
}

func wireTerminal(deps terminalDeps) {
	args.NewPHPHandler(args.HandlerDeps{
		RootCmd: deps.RootCmd, Lifecycle: deps.Lifecycle, Silo: deps.Silo,
		Assembler: deps.Assembler, Registry: deps.Registry, Bundle: deps.Bundle,
		System: deps.System, Shim: deps.Shim, PECL: deps.PECL, Config: deps.Config,
		Doctor: deps.Doctor, Update: deps.Update, PatchSync: deps.PatchSync,
		PatchLister: deps.PatchLister,
		Version:     version,
	})
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	options := []fx.Option{
		fx.NopLogger,
		fx.Supply(version),
		fx.Provide(
			args.NewRootCmd,
			fx.Annotate(memory.NewRegistryRepository, fx.As(new(registry.RegistryRepository))),
			fx.Annotate(disk.NewSiloRepository, fx.As(new(silo.SiloRepository))),
			fx.Annotate(disk.NewForgeRepository, fx.As(new(forge.ForgeRepository))),
			fx.Annotate(NewPatchesRepository, fx.As(new(patcher.PatcherRepository))),
			NewPatchSyncService,
			NewPatchLister,
			NewServerManagerFactory,
			fx.Annotate(memory.NewGraphRepository, fx.As(new(graph.GraphRepository))),
			registry.NewService,
			fx.Annotate(registry.NewService, fx.As(new(silo.RegistryLookup))),
			silo.NewService,
			fx.Annotate(silo.NewService, fx.As(new(bundle.BundleSilo))),
			bundle.NewService,
			system.NewService,
			fx.Annotate(system.NewService, fx.As(new(doctor.SystemService))),
			fx.Annotate(silo.NewService, fx.As(new(launchers.ShimSilo))),
			launchers.NewService,
			fx.Annotate(silo.NewService, fx.As(new(pecl.PECLSilo))),
			fx.Annotate(editor.NewService, fx.As(new(pecl.IniEditor))),
			pecl.NewService,
			fx.Annotate(disk.NewConfigRepository, fx.As(new(config.ConfigRepository))),
			config.NewService,
			fx.Annotate(disk.NewDoctorRepository, fx.As(new(doctor.Repository))),
			doctor.NewService,
			fx.Annotate(disk.NewUpdateRepository, fx.As(new(update.UpdateRepository))),
			update.NewService,
			NewAssemblerDeps,
			assembler.NewService,
			fx.Annotate(assembler.NewService, fx.As(new(args.ServerAssembler))),
			fx.Annotate(NewFPMServiceForServer, fx.As(new(args.FPMService))),
			fx.Annotate(NewApacheServiceForServer, fx.As(new(args.ApacheService))),
			forge.NewService,
			patcher.NewService,
			graph.NewService,
			fx.Annotate(graph.NewService, fx.As(new(doctor.GraphLookup))),
		),
		fx.Invoke(wireTerminal),
		fx.Invoke(RegisterRootCmd),
	}

	app := fx.New(options...)

	if err := app.Start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := app.Stop(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
