package args

import (
	"context"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pkg/compiler"
)

// FPMService is the php-fpm capability the server commands consume.
type FPMService interface {
	NeedsRebuild(phpVersion string, tc compiler.Toolchain) bool
	EnsureFPM(ctx context.Context, spec domain.FPMSpec) (string, error)
}

// ApacheService is the mod_php capability the server commands consume.
type ApacheService interface {
	NeedsRebuild(phpVersion, httpdPrefix string, tc compiler.Toolchain) bool
	EnsureApacheModule(ctx context.Context, spec domain.ApacheModuleSpec) (string, error)
}

// ServerAssembler is the webserver-package assembly capability the server
// commands consume.
type ServerAssembler interface {
	AssembleServer(ctx context.Context, spec domain.ServerAssembleSpec) (string, error)
}

// ServerStore exposes the running-instance index of a server manager.
type ServerStore interface {
	Running(inst domain.ServerInstance) bool
}

// ServerManager is the webserver lifecycle capability the server commands
// consume.
type ServerManager interface {
	Start(ctx context.Context, opts domain.ServerRunOptions) (domain.ServerInstance, error)
	Stop(inst domain.ServerInstance) error
	StopByID(id string) (domain.ServerInstance, error)
	List() ([]domain.ServerInstance, error)
	Running() ([]domain.ServerInstance, error)
	Prune() (int, error)
	Store() ServerStore
}

func domainToolchain(tc compiler.Toolchain) domain.Toolchain {
	return domain.Toolchain{Name: tc.Name, CC: tc.CC, CXX: tc.CXX, Hint: tc.Hint}
}

// ServerManagerDeps is the delivery-side description of the webserver manager
// inputs. The composition root adapts it to the concrete manager.
type ServerManagerDeps struct {
	Root                   string
	ResolveVersion         func(constraint string) (string, error)
	EnsureFPM              func(ctx context.Context, phpVersion string, verbose bool, jobs int) (string, error)
	EnsureApacheModule     func(ctx context.Context, phpVersion, httpdPrefix string, opts domain.ServerRunOptions) (string, error)
	CLIPrefix              func(phpVersion string) string
	InstalledServerVersion func(server, constraint string) (string, error)
	ServerPrefix           func(server, version string) string
}

// ServerManagerFactory constructs a server manager from delivery-supplied callbacks.
type ServerManagerFactory func(deps ServerManagerDeps) ServerManager
