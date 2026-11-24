package args

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"go.uber.org/fx"
)

// SiloService is the set of silo operations the terminal handler needs.
type SiloService interface {
	Download(name, version string) (bool, error)
	DownloadURL(url, checksumType, checksumValue string) (bool, error)
	Extract(archivePath, destDir string) (bool, error)
	GetSilo() domain.Silo
	GetState(name, version string) (domain.InstallState, error)
	GetDefault() (string, error)
	SetDefault(version string) error
	PackagePrefix(name, version string) string
	SourcePath(pkg, version string) string
	PHPOutputPath(phpVersion string) string
	GetExtensionManifest(phpVersion string) (*domain.ExtensionManifest, error)
	SaveExtensionManifest(phpVersion string, m *domain.ExtensionManifest) error
}

// PHPHandler registers cobra commands and delegates to services.
type PHPHandler struct {
	ctx          context.Context
	siloSvc      SiloService
	assemblerSvc AssemblerService
	registrySvc  RegistryService
	bundleSvc    BundleService
	systemSvc    SystemService
	shimSvc      ShimService
	peclSvc      PECLService
	configSvc    ConfigService
	doctorSvc    DoctorService
	updateSvc    UpdateService
	patchSync    PatchSync
	patchLister  PatchLister
	fpmSvc       FPMService
	apacheSvc    ApacheService
	serverAsbl   ServerAssembler
	serverMgrFn  ServerManagerFactory
	version      string
}

// HandlerDeps groups the terminal handler's dependencies.
type HandlerDeps struct {
	RootCmd     *cobra.Command
	Lifecycle   fx.Lifecycle
	Silo        SiloService
	Assembler   AssemblerService
	Registry    RegistryService
	Bundle      BundleService
	System      SystemService
	Shim        ShimService
	PECL        PECLService
	Config      ConfigService
	Doctor      DoctorService
	Update      UpdateService
	PatchSync   PatchSync
	PatchLister PatchLister
	FPM         FPMService
	Apache      ApacheService
	ServerAsm   ServerAssembler
	ServerMgr   ServerManagerFactory
	Version     string
}

// NewPHPHandler builds the handler and registers its commands on the root
// command. The returned handler lets callers and tests inspect what was wired.
func NewPHPHandler(deps HandlerDeps) *PHPHandler {
	h := &PHPHandler{
		ctx:          context.Background(),
		siloSvc:      deps.Silo,
		assemblerSvc: deps.Assembler,
		registrySvc:  deps.Registry,
		bundleSvc:    deps.Bundle,
		systemSvc:    deps.System,
		shimSvc:      deps.Shim,
		peclSvc:      deps.PECL,
		configSvc:    deps.Config,
		doctorSvc:    deps.Doctor,
		updateSvc:    deps.Update,
		patchSync:    deps.PatchSync,
		patchLister:  deps.PatchLister,
		fpmSvc:       deps.FPM,
		apacheSvc:    deps.Apache,
		serverAsbl:   deps.ServerAsm,
		serverMgrFn:  deps.ServerMgr,
		version:      deps.Version,
	}
	deps.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			h.ctx = ctx
			return nil
		},
	})
	deps.RootCmd.AddCommand(h.downloadCmd())
	deps.RootCmd.AddCommand(h.installCmd())
	deps.RootCmd.AddCommand(h.versionsCmd())
	deps.RootCmd.AddCommand(h.listCmd())
	deps.RootCmd.AddCommand(h.whichCmd())
	deps.RootCmd.AddCommand(h.defaultCmd())
	deps.RootCmd.AddCommand(h.useCmd())
	deps.RootCmd.AddCommand(h.shareCmd())
	deps.RootCmd.AddCommand(h.extensionCmd())
	deps.RootCmd.AddCommand(h.initCmd())
	deps.RootCmd.AddCommand(h.rehashCmd())
	deps.RootCmd.AddCommand(h.pharCmd())
	deps.RootCmd.AddCommand(h.serverCmd())
	deps.RootCmd.AddCommand(h.autoDetectResolveCmd())
	deps.RootCmd.AddCommand(h.peclCmd())
	deps.RootCmd.AddCommand(h.uninstallCmd())
	deps.RootCmd.AddCommand(h.configCmd())
	deps.RootCmd.AddCommand(h.completionCmd())
	deps.RootCmd.AddCommand(h.doctorCmd())
	deps.RootCmd.AddCommand(h.updateCmd())
	deps.RootCmd.AddCommand(h.patchCmd())
	return h
}

func (h *PHPHandler) uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <version>",
		Short: "Remove an installed PHP version",
		Long:  "Remove an installed PHP version including its prefix, source, and state files.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.uninstall,
	}
	cmd.Flags().Bool("yes", false, "Skip confirmation prompt when uninstalling the default version")
	return cmd
}

func (h *PHPHandler) uninstall(cmd *cobra.Command, args []string) error {
	version, err := h.resolveVersion(args[0])
	if err != nil {
		return err
	}

	state, err := h.siloSvc.GetState("php", version)
	if err != nil {
		return fmt.Errorf("check state: %w", err)
	}
	if state != domain.StateInstalled {
		return fmt.Errorf("PHP %s is not installed (run `phpv versions` to see installed versions)", version)
	}

	defaultVer, _ := h.siloSvc.GetDefault()
	if version == defaultVer {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			fmt.Printf("PHP %s is the current default version. Uninstall anyway? [y/N] ", version)
			os.Stdout.Sync()
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				fmt.Println("[non-interactive, skipping]")
				return fmt.Errorf("uninstall cancelled: default version requires confirmation; use --yes to skip")
			}
			reader := bufio.NewReader(os.Stdin)
			answer, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				return fmt.Errorf("uninstall cancelled")
			}
		}
		if err := h.siloSvc.SetDefault(""); err != nil {
			return fmt.Errorf("clear default: %w", err)
		}
		fmt.Printf("→ Cleared default version\n")
	}

	prefix := h.siloSvc.PackagePrefix("php", version)
	fmt.Printf("Removing %s...\n", prefix)
	os.RemoveAll(prefix)

	sourceDir := h.siloSvc.SourcePath("php", version)
	if _, err := os.Stat(sourceDir); err == nil {
		fmt.Printf("Removing source %s...\n", sourceDir)
		os.RemoveAll(sourceDir)
	}

	if err := h.shimSvc.RegenerateAll(); err != nil {
		return fmt.Errorf("regenerate shims: %w", err)
	}

	fmt.Printf("✓ PHP %s uninstalled\n", version)
	return nil
}
