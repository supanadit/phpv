package terminal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundle"
	"github.com/supanadit/phpv/config"
	"github.com/supanadit/phpv/doctor"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/appctx"
	"github.com/supanadit/phpv/internal/repository"
	"github.com/supanadit/phpv/pecl"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/shim"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/system"
	"github.com/supanadit/phpv/update"
)

// PHPHandler registers cobra commands and delegates to services.
type PHPHandler struct {
	ctx          context.Context
	siloSvc      *silo.Service
	assemblerSvc *assembler.Service
	registrySvc  *registry.Service
	bundleSvc    *bundle.Service
	systemSvc    *system.Service
	shimSvc      *shim.Service
	peclSvc      *pecl.Service
	configSvc    *config.Service
	doctorSvc    *doctor.Service
	updateSvc    *update.Service
	version      string
}

func NewPHPHandler(rootCmd *cobra.Command, ac appctx.AppContext, siloSvc *silo.Service, assemblerSvc *assembler.Service, registrySvc *registry.Service, bundleSvc *bundle.Service, systemSvc *system.Service, shimSvc *shim.Service, peclSvc *pecl.Service, configSvc *config.Service, doctorSvc *doctor.Service, updateSvc *update.Service, version string) {
	h := &PHPHandler{
		ctx:          ac.Ctx,
		siloSvc:      siloSvc,
		assemblerSvc: assemblerSvc,
		registrySvc:  registrySvc,
		bundleSvc:    bundleSvc,
		systemSvc:    systemSvc,
		shimSvc:      shimSvc,
		peclSvc:      peclSvc,
		configSvc:    configSvc,
		doctorSvc:    doctorSvc,
		updateSvc:    updateSvc,
		version:      version,
	}
	rootCmd.AddCommand(h.downloadCmd())
	rootCmd.AddCommand(h.installCmd())
	rootCmd.AddCommand(h.versionsCmd())
	rootCmd.AddCommand(h.listCmd())
	rootCmd.AddCommand(h.whichCmd())
	rootCmd.AddCommand(h.defaultCmd())
	rootCmd.AddCommand(h.useCmd())
	rootCmd.AddCommand(h.shareCmd())
	rootCmd.AddCommand(h.extensionCmd())
	rootCmd.AddCommand(h.initCmd())
	rootCmd.AddCommand(h.rehashCmd())
	rootCmd.AddCommand(h.pharCmd())
	rootCmd.AddCommand(h.autoDetectResolveCmd())
	rootCmd.AddCommand(h.peclCmd())
	rootCmd.AddCommand(h.uninstallCmd())
	rootCmd.AddCommand(h.configCmd())
	rootCmd.AddCommand(h.completionCmd())
	rootCmd.AddCommand(h.doctorCmd())
	rootCmd.AddCommand(h.updateCmd())
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

func (h *PHPHandler) downloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download <version>",
		Short: "Download a PHP version",
		Long:  "Download a specific version of PHP and all its transitive dependencies to the local cache.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.download,
	}
}

func (h *PHPHandler) download(cmd *cobra.Command, args []string) error {
	version := args[0]
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = "php"
	}

	fmt.Printf("Downloading %s@%s...\n", name, version)
	downloaded, err := h.siloSvc.Download(name, version)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if downloaded {
		fmt.Printf("✓ Downloaded %s@%s\n", name, version)
	} else {
		fmt.Printf("→ Skipped %s@%s (already exists)\n", name, version)
	}
	return nil
}

func (h *PHPHandler) installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <version>",
		Short: "Install a PHP version",
		Long: `Download, build, and install a specific version of PHP with all its dependencies.

Version syntax:
  8.4.4    exact version
  8.4      latest 8.4.x patch
  8        latest 8.x.x minor`,
		Args: cobra.ExactArgs(1),
		RunE: h.install,
	}
	cmd.Flags().String("from", "", "Install from a bundle file instead of building from source")
	cmd.Flags().Bool("static", false, "Build with static linking for cross-distro portability")
	cmd.Flags().String("ext", "", "Comma-separated list of extensions to build (e.g., openssl,curl,pdo_mysql)")
	cmd.Flags().Bool("auto-deps", false, "Install missing system packages without prompting")
	cmd.Flags().Bool("no-system", false, "Skip system package check, always build from source")
	cmd.Flags().Bool("dry-run", false, "Show what would be done without doing it")
	cmd.Flags().Bool("fresh", false, "Delete existing install prefix and rebuild (keeps cached source)")
	cmd.Flags().Bool("clean", false, "Delete everything including cached source and rebuild from scratch")
	cmd.Flags().Bool("verbose", false, "Show full build output instead of spinner")
	cmd.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	cmd.Flags().Bool("yes", false, "Skip confirmation prompts")
	cmd.Flags().Bool("force", false, "Skip state checks and reinstall")
	cmd.Flags().Bool("minimal", false, "Build with --enable-cli only (no default extensions)")
	return cmd
}

func (h *PHPHandler) install(cmd *cobra.Command, args []string) error {
	version := args[0]

	// Detect if the argument is a local bundle file path.
	if isBundlePath(version) {
		fmt.Printf("Installing PHP from bundle %s...\n", version)
		if err := h.bundleSvc.ImportFromPath(version); err != nil {
			return fmt.Errorf("install from bundle failed: %w", err)
		}
		if err := h.shimSvc.RegenerateAll(); err != nil {
			return fmt.Errorf("regenerate shims: %w", err)
		}
		fmt.Printf("✓ PHP installed from bundle\n")
		return nil
	}

	fromBundle, _ := cmd.Flags().GetString("from")
	if fromBundle != "" {
		fmt.Printf("Installing PHP %s from bundle %s...\n", version, fromBundle)
		if err := h.bundleSvc.Import(fromBundle, version); err != nil {
			return fmt.Errorf("install from bundle failed: %w", err)
		}
		if err := h.shimSvc.RegenerateAll(); err != nil {
			return fmt.Errorf("regenerate shims: %w", err)
		}
		fmt.Printf("✓ PHP %s installed from bundle\n", version)
		return nil
	}

	state, _ := h.siloSvc.GetState("php", version)
	if state == domain.StateFailed || state == domain.StateInterrupted || state == domain.StateInProgress {
		force, _ := cmd.Flags().GetBool("force")
		clean, _ := cmd.Flags().GetBool("clean")
		if !force && !clean {
			stateMsg := string(state)
			if state == domain.StateInterrupted {
				stateMsg = "interrupted (you pressed Ctrl+C)"
			} else if state == domain.StateInProgress {
				stateMsg = "in_progress (likely crashed)"
			}
			fmt.Printf("Previous install of PHP %s was '%s'.\n", version, stateMsg)
			fmt.Printf("Run `phpv install %s --force` to retry (deps preserved).\n", version)
			fmt.Printf("Or use `--clean` to delete the PHP prefix/source first.\n")
			return nil
		}
	}

	static, _ := cmd.Flags().GetBool("static")
	extStr, _ := cmd.Flags().GetString("ext")
	minimal, _ := cmd.Flags().GetBool("minimal")
	autoDeps, _ := cmd.Flags().GetBool("auto-deps")
	noSystem, _ := cmd.Flags().GetBool("no-system")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	clean, _ := cmd.Flags().GetBool("clean")
	fresh, _ := cmd.Flags().GetBool("fresh")
	verbose, _ := cmd.Flags().GetBool("verbose")
	jobsFlag, _ := cmd.Flags().GetInt("jobs")

	jobs := resolveJobs(jobsFlag, h.configSvc)

	if extStr != "" && minimal {
		return fmt.Errorf("cannot use --ext with --minimal")
	}

	resolvedVersion, resolveErr := h.assemblerSvc.ResolveVersion("php", version)
	if resolveErr != nil {
		return fmt.Errorf("resolve version: %w", resolveErr)
	}

	var extensions []string
	var err error
	if extStr != "" {
		for _, e := range strings.Split(extStr, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				extensions = append(extensions, e)
			}
		}
	} else if !minimal {
		defaults, skipped := h.assemblerSvc.Graph().DefaultExtensions(resolvedVersion)
		if len(defaults) == 0 && len(skipped) > 0 {
			fmt.Printf("Note: No extensions from the default set are compatible with PHP %s.\n", resolvedVersion)
			fmt.Println("Use --ext <list> to build specific extensions, or --minimal for a bare PHP.")
		} else {
			fmt.Printf("Default extension set (%d extensions", len(defaults))
			if len(skipped) > 0 {
				fmt.Printf(", %d skipped", len(skipped))
			}
			fmt.Println("):")
			fmt.Printf("  %s\n", strings.Join(defaults, ", "))
			if len(skipped) > 0 {
				fmt.Println()
				for _, s := range skipped {
					fmt.Printf("  skipped: %s\n", s)
				}
			}
		}
		extensions = defaults
	}
	extensions, added := h.assemblerSvc.Graph().ExpandImplied(extensions)
	if len(added) > 0 {
		fmt.Printf("Including implied extensions: %s\n", strings.Join(added, ", "))
	}

	var systemPkgs map[string]system.Package
	if !noSystem {
		systemPkgs, err = h.checkSystemDeps(extensions, autoDeps, dryRun)
		if err != nil {
			return err
		}
	}

	if dryRun {
		fmt.Println("Dry run complete. Run without --dry-run to install.")
		return nil
	}

	if clean {
		prefix := h.siloSvc.PackagePrefix("php", version)
		fmt.Printf("Cleaning existing install at %s...\n", prefix)
		os.RemoveAll(prefix)
		sourceDir := h.siloSvc.SourcePath("php", version)
		os.RemoveAll(sourceDir)
		force = true
	} else if fresh {
		prefix := h.siloSvc.PackagePrefix("php", version)
		fmt.Printf("Refreshing install at %s (keeping cached source)...\n", prefix)
		os.RemoveAll(prefix)
		force = true
	}

	bannerVersion := version
	if exact, err := h.assemblerSvc.ResolveVersion("php", version); err == nil {
		bannerVersion = exact
	}
	fmt.Printf("Installing PHP %s...\n\n", bannerVersion)

	if verbose {
		result, err := h.assemblerSvc.Assemble(h.ctx, "php", version, static, extensions, true, nil, systemPkgs, jobs, force)
		if err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
		fmt.Println()
		if result.AlreadyInstalled {
			fmt.Printf("✓ PHP %s is already installed\n", result.Version)
			return nil
		}
		sharedOnly := h.assemblerSvc.Graph().SharedOnlyExtensions(resolvedVersion, extensions)
		if len(sharedOnly) > 0 {
			sourceDir := h.siloSvc.SourcePath("php", resolvedVersion)
			srcPath := assembler.FindSourceDir(sourceDir, "php", resolvedVersion)
			if srcPath == "" {
				fmt.Printf("Warning: PHP source not found at %s, skipping shared extension builds\n", sourceDir)
			} else {
				for _, ext := range sharedOnly {
					fmt.Printf("Building %s as shared extension...\n", ext)
					if err := h.assemblerSvc.InstallExtension(h.ctx, resolvedVersion, ext, srcPath, result.Prefix, jobs); err != nil {
						fmt.Printf("Warning: %s shared build failed: %v\n", ext, err)
						fmt.Printf("  Run `phpv extension %s add %s` to retry.\n", resolvedVersion, ext)
					} else {
						fmt.Printf("✓ %s installed (shared)\n", ext)
					}
				}
			}
		}
		if err := h.shimSvc.RegenerateAll(); err != nil {
			return fmt.Errorf("regenerate shims: %w", err)
		}
		return nil
	}

	progressCh := make(chan progressMsg, 64)
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)
		var current string
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for {
			select {
			case msg, ok := <-progressCh:
				if !ok {
					return
				}
				glyph := stageGlyph(msg.stage)
				if msg.stage == "done" {
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s\n", glyph, msg.message)
				} else {
					current = fmt.Sprintf("%s %s", glyph, msg.message)
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
				}
			case <-ticker.C:
				if current != "" {
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
				}
				frame++
			}
		}
	}()

	result, err := h.assemblerSvc.Assemble(h.ctx, "php", version, static, extensions, false, func(stage, message string) {
		progressCh <- progressMsg{stage: stage, message: message}
	}, systemPkgs, jobs, force)
	close(progressCh)
	<-doneCh

	if err != nil {
		fmt.Println()
		return fmt.Errorf("install failed: %w", err)
	}
	fmt.Println()
	if result.AlreadyInstalled {
		fmt.Printf("✓ PHP %s is already installed\n", result.Version)
		return nil
	}

	sharedOnly := h.assemblerSvc.Graph().SharedOnlyExtensions(resolvedVersion, extensions)
	if len(sharedOnly) > 0 {
		sourceDir := h.siloSvc.SourcePath("php", resolvedVersion)
		srcPath := assembler.FindSourceDir(sourceDir, "php", resolvedVersion)
		if srcPath == "" {
			fmt.Printf("Warning: PHP source not found at %s, skipping shared extension builds\n", sourceDir)
		} else {
			for _, ext := range sharedOnly {
				fmt.Printf("Building %s as shared extension...\n", ext)
				if err := h.assemblerSvc.InstallExtension(h.ctx, resolvedVersion, ext, srcPath, result.Prefix, jobs); err != nil {
					fmt.Printf("Warning: %s shared build failed: %v\n", ext, err)
					fmt.Printf("  Run `phpv extension %s add %s` to retry.\n", resolvedVersion, ext)
				} else {
					fmt.Printf("✓ %s installed (shared)\n", ext)
				}
			}
		}
	}

	if err := h.shimSvc.RegenerateAll(); err != nil {
		return fmt.Errorf("regenerate shims: %w", err)
	}
	return nil
}

// versionsCmd lists installed PHP versions.
func (h *PHPHandler) versionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions",
		Short: "List installed PHP versions",
		Args:  cobra.NoArgs,
		RunE:  h.versions,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

type versionEntry struct {
	Version string `json:"version"`
	Default bool   `json:"default"`
}

func (h *PHPHandler) versions(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	silo := h.siloSvc.GetSilo()
	phpDir := filepath.Join(silo.Root, "packages", "php")

	entries, err := os.ReadDir(phpDir)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonFlag {
				return printJSON(jsonResponse{SchemaVersion: 1, Data: []versionEntry{}})
			}
			fmt.Println("No PHP versions installed.")
			return nil
		}
		return fmt.Errorf("read php versions dir: %w", err)
	}

	defaultVer, _ := h.siloSvc.GetDefault()

	var installed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		phpBin := filepath.Join(phpDir, e.Name(), "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			installed = append(installed, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(installed)))

	if jsonFlag {
		var vers []versionEntry
		for _, v := range installed {
			vers = append(vers, versionEntry{Version: v, Default: v == defaultVer})
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: vers})
	}

	if len(installed) == 0 {
		fmt.Println("No PHP versions installed.")
		return nil
	}

	fmt.Println("Installed PHP versions:")
	for _, v := range installed {
		marker := " "
		if v == defaultVer {
			marker = "*"
		}
		fmt.Printf("  %s %s\n", marker, v)
	}
	if defaultVer != "" {
		fmt.Printf("\n(* = default)\n")
	}
	return nil
}

// listCmd lists available PHP versions from the registry.
func (h *PHPHandler) listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available PHP versions",
		Args:  cobra.NoArgs,
		RunE:  h.listAvailable,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

type listEntry struct {
	Version string `json:"version"`
}

func (h *PHPHandler) listAvailable(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	entries, err := h.registrySvc.List("php")
	if err != nil {
		return fmt.Errorf("list php versions: %w", err)
	}

	seen := make(map[string]bool)
	var versions []string
	for _, e := range entries {
		if !seen[e.Version] {
			seen[e.Version] = true
			versions = append(versions, e.Version)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	if jsonFlag {
		var vers []listEntry
		for _, v := range versions {
			vers = append(vers, listEntry{Version: v})
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: vers})
	}

	fmt.Println("Available PHP versions:")
	for _, v := range versions {
		fmt.Printf("  %s\n", v)
	}
	return nil
}

// whichCmd shows the path to the current PHP binary.
func (h *PHPHandler) whichCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "which",
		Short: "Show path to current PHP binary",
		Args:  cobra.NoArgs,
		RunE:  h.which,
	}
}

func (h *PHPHandler) which(cmd *cobra.Command, args []string) error {
	path, err := h.resolveActivePHP()
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// defaultCmd sets the global default PHP version.
func (h *PHPHandler) defaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default <version>",
		Short: "Set default PHP version",
		Long: `Set the default PHP version.

--global (default) writes to ~/.phpv/default.
--local writes to .php-version in the current directory.`,
		Args: cobra.ExactArgs(1),
		RunE: h.setDefault,
	}
	cmd.Flags().Bool("global", true, "Set as global default (writes to ~/.phpv/default)")
	cmd.Flags().Bool("local", false, "Set as local project version (writes .php-version in CWD)")
	cmd.MarkFlagsMutuallyExclusive("global", "local")
	return cmd
}

func (h *PHPHandler) setDefault(cmd *cobra.Command, args []string) error {
	version, err := h.resolveVersion(args[0])
	if err != nil {
		return err
	}

	local, _ := cmd.Flags().GetBool("local")
	if local {
		if err := writeLocalVersionFile(version); err != nil {
			return fmt.Errorf("write local version: %w", err)
		}
		fmt.Printf("✓ Local PHP version set to %s (.php-version)\n", version)
		return nil
	}

	if err := h.siloSvc.SetDefault(version); err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	fmt.Printf("✓ Default PHP version set to %s\n", version)
	return nil
}

// useCmd switches the active PHP version for the current session.
func (h *PHPHandler) useCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a PHP version",
		Long: `Switch to a specific PHP version. Use 'system' to use the system PHP.

--global (default) writes to ~/.phpv/default and regenerates shims.
--local writes to .php-version in the current directory only.`,
		Args: cobra.ExactArgs(1),
		RunE: h.use,
	}
	cmd.Flags().Bool("global", true, "Set as global default (writes to ~/.phpv/default, regenerates shims)")
	cmd.Flags().Bool("local", false, "Set as local project version (writes .php-version in CWD)")
	cmd.MarkFlagsMutuallyExclusive("global", "local")
	return cmd
}

func (h *PHPHandler) use(cmd *cobra.Command, args []string) error {
	version := args[0]

	if version == "system" {
		systemPHP, err := exec.LookPath("php")
		if err != nil {
			return fmt.Errorf("system PHP not found in PATH")
		}
		silo := h.siloSvc.GetSilo()
		phpvBin := filepath.Join(silo.Root, "bin", "php")
		if systemPHP == phpvBin {
			return fmt.Errorf("system PHP is managed by phpv; use a specific version instead")
		}
		if err := h.shimSvc.SetSystemMode(true); err != nil {
			return fmt.Errorf("set system mode: %w", err)
		}
		if err := h.shimSvc.RegenerateAll(); err != nil {
			return fmt.Errorf("regenerate shims: %w", err)
		}
		fmt.Printf("→ Using system PHP at %s\n", systemPHP)
		fmt.Println("Run `phpv init` in your shell to enable version switching.")
		return nil
	}

	exactVersion, err := h.resolveVersion(version)
	if err != nil {
		return err
	}

	local, _ := cmd.Flags().GetBool("local")
	if local {
		if err := writeLocalVersionFile(exactVersion); err != nil {
			return fmt.Errorf("write local version: %w", err)
		}
		fmt.Printf("✓ Switched to PHP %s (local, .php-version)\n", exactVersion)
		return nil
	}

	if err := h.shimSvc.SetSystemMode(false); err != nil {
		return fmt.Errorf("clear system mode: %w", err)
	}
	if err := h.siloSvc.SetDefault(exactVersion); err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	if err := h.shimSvc.RegenerateAll(); err != nil {
		return fmt.Errorf("regenerate shims: %w", err)
	}

	silo := h.siloSvc.GetSilo()
	phpBin := filepath.Join(silo.Root, "packages", "php", exactVersion, "bin", "php")
	fmt.Printf("✓ Switched to PHP %s (%s)\n", exactVersion, phpBin)
	fmt.Println("Run `phpv init` in your shell to enable version switching.")
	return nil
}

// resolveActivePHP resolves the active PHP binary path.
// Priority: PHPV_CURRENT env > .phpvrc > default > system PHP.
func (h *PHPHandler) resolveActivePHP() (string, error) {
	// 1. Check PHPV_CURRENT env var.
	if envVer := os.Getenv("PHPV_CURRENT"); envVer != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "packages", "php", envVer, "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 2. Check .php-version or .phpvrc in current or parent directories.
	if ver := findProjectVersionFile(); ver != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "packages", "php", ver, "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 3. Check default.
	defaultVer, err := h.siloSvc.GetDefault()
	if err == nil && defaultVer != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "packages", "php", defaultVer, "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 4. Fall back to system PHP.
	systemPHP, err := exec.LookPath("php")
	if err == nil {
		return systemPHP, nil
	}

	return "", fmt.Errorf("no PHP version found (install one with `phpv install <version>`)")
}

// resolveActiveVersion returns the active PHP version string.
// Priority: PHPV_CURRENT env > .php-version > .phpvrc > default.
func (h *PHPHandler) resolveActiveVersion() (string, error) {
	if envVer := os.Getenv("PHPV_CURRENT"); envVer != "" {
		return envVer, nil
	}
	if ver := findProjectVersionFile(); ver != "" {
		return ver, nil
	}
	defaultVer, err := h.siloSvc.GetDefault()
	if err == nil && defaultVer != "" {
		return defaultVer, nil
	}
	return "", fmt.Errorf("no active PHP version (set one with `phpv use <version>` or `export PHPV_CURRENT=<version>`)")
}

// resolveVersion resolves a version constraint to an exact installed version.
// Empty string falls back to the active version.
func (h *PHPHandler) resolveVersion(constraint string) (string, error) {
	if constraint == "" {
		return h.resolveActiveVersion()
	}
	return h.resolveInstalledVersion(constraint)
}

// resolveInstalledVersion resolves a version constraint to an exact installed version.
func (h *PHPHandler) resolveInstalledVersion(constraint string) (string, error) {
	silo := h.siloSvc.GetSilo()
	phpDir := filepath.Join(silo.Root, "packages", "php")

	entries, err := os.ReadDir(phpDir)
	if err != nil {
		return "", fmt.Errorf("no PHP versions installed")
	}

	var installed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		phpBin := filepath.Join(phpDir, e.Name(), "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			installed = append(installed, e.Name())
		}
	}

	// Exact match first.
	for _, v := range installed {
		if v == constraint {
			return v, nil
		}
	}

	// Major.minor or major-only match (e.g., "8.4" → latest 8.4.x, "8" → latest 8.x.x).
	if strings.Count(constraint, ".") == 1 || strings.Count(constraint, ".") == 0 {
		prefix := constraint + "."
		if best := repository.LatestMatching(installed, prefix); best != "" {
			return best, nil
		}
	}

	return "", fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", constraint, constraint)
}

// findProjectVersionFile walks up from the current directory looking for
// a .php-version or .phpvrc file. .php-version takes priority.
func findProjectVersionFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, ".php-version")); err == nil {
			return strings.TrimSpace(string(data))
		}
		if data, err := os.ReadFile(filepath.Join(dir, ".phpvrc")); err == nil {
			return strings.TrimSpace(string(data))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// writeLocalVersionFile writes the given version to .php-version in the
// current working directory.
func writeLocalVersionFile(version string) error {
	return os.WriteFile(".php-version", []byte(version+"\n"), 0644)
}

// resolveJobs resolves the effective parallelism value.
// Priority: CLI flag > config concurrency > runtime.NumCPU().
func resolveJobs(flagVal int, cfgSvc *config.Service) int {
	if flagVal > 0 {
		return flagVal
	}
	if cfgSvc != nil {
		val, err := cfgSvc.Get("concurrency")
		if err == nil && val != "" {
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				return n
			}
		}
	}
	return runtime.NumCPU()
}

// checkSystemDeps checks for system packages needed by PHP deps and extensions.
// If autoDeps is true, installs missing packages without prompting.
// If dryRun is true, only prints what would be done.
// Returns a map of available system packages (name -> Package) for use in hybrid builds.
func (h *PHPHandler) checkSystemDeps(extensions []string, autoDeps, dryRun bool) (map[string]system.Package, error) {
	phpDeps := []string{"openssl", "libxml2", "zlib", "oniguruma", "curl", "sqlite3", "readline", "icu", "pcre2", "argon2", "sodium"}
	for _, ext := range extensions {
		switch ext {
		case "openssl":
		case "curl":
		case "gd":
			phpDeps = append(phpDeps, "libpng", "libjpeg", "freetype")
		case "intl":
			phpDeps = append(phpDeps, "icu")
		case "libxml":
			phpDeps = append(phpDeps, "libxml2")
		}
	}

	result, err := h.systemSvc.Check(phpDeps)
	if err != nil {
		return nil, fmt.Errorf("system check: %w", err)
	}

	systemAvail := make(map[string]system.Package)
	for _, p := range result.Available {
		systemAvail[p.Name] = p
	}

	if result.Distro.PM == "unknown" {
		return systemAvail, nil
	}

	if len(result.Missing) == 0 {
		return systemAvail, nil
	}

	fmt.Println("\nMissing system packages:")
	for _, p := range result.Missing {
		fmt.Printf("  ✗ %s (%s)\n", p.Name, p.SystemName)
	}

	installCmd := h.systemSvc.InstallCommand(result.Missing)
	fmt.Printf("\nInstall via %s? ", installCmd)
	os.Stdout.Sync()

	if autoDeps {
		fmt.Println("[Y] (--auto-deps)")
	} else if dryRun {
		fmt.Println("[skipped, --dry-run]")
		return systemAvail, nil
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			fmt.Println("[non-interactive, skipping]")
			return systemAvail, nil
		}
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[read error: %v, skipping]\n", err)
			return systemAvail, nil
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" && answer != "" {
			fmt.Println("Building from source instead.")
			return systemAvail, nil
		}
	}

	fmt.Println("Installing system packages...")
	if err := h.systemSvc.Install(result.Missing); err != nil {
		return nil, fmt.Errorf("install system packages: %w", err)
	}
	fmt.Println("✓ System packages installed")

	for _, p := range result.Missing {
		p.Installed = true
		systemAvail[p.Name] = p
	}

	return systemAvail, nil
}

// shareCmd exports an installed PHP version as a portable bundle.
func (h *PHPHandler) shareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "share <version>",
		Short: "Export installed PHP as a portable bundle",
		Long:  "Export an installed PHP version as a portable .tar.gz bundle that can be shared with others.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.share,
	}
	cmd.Flags().StringP("output", "o", "", "Output path for the bundle file")
	cmd.Flags().Bool("portable", false, "Build a portable bundle from source (requires build env)")
	cmd.Flags().String("libc", "musl", "Libc variant for portable build: musl (default, runs anywhere) or glibc (glibc hosts only)")
	return cmd
}

func (h *PHPHandler) share(cmd *cobra.Command, args []string) error {
	version := args[0]
	output, _ := cmd.Flags().GetString("output")
	portable, _ := cmd.Flags().GetBool("portable")

	if portable {
		libc, _ := cmd.Flags().GetString("libc")
		if libc != "musl" && libc != "glibc" {
			return fmt.Errorf("--libc must be 'musl' or 'glibc', got %q", libc)
		}
		fmt.Printf("Building portable PHP %s (%s)...\n", version, libc)
		opts := bundle.PublishOpts{
			Version:    version,
			OutputPath: output,
			Jobs:       0,
			Force:      false,
			Libc:       libc,
		}
		if err := h.bundleSvc.Publish(h.ctx, opts); err != nil {
			return fmt.Errorf("portable build failed: %w", err)
		}
		return nil
	}

	fmt.Printf("Exporting PHP %s...\n", version)
	if err := h.bundleSvc.Export(version, output); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}
	if output == "" {
		output = fmt.Sprintf("php-%s-%s-%s.tar.gz", version, "linux", "amd64")
	}
	fmt.Printf("✓ PHP %s exported to %s\n", version, output)
	return nil
}

// autoDetectResolveCmd is a hidden subcommand used by shim scripts to resolve version constraints.
func (h *PHPHandler) autoDetectResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "auto-detect-resolve [version]",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			constraint := ""
			if len(args) == 1 {
				constraint = args[0]
			}
			exact, err := h.assemblerSvc.ResolveVersion("php", constraint)
			if err != nil {
				return err
			}
			fmt.Println(exact)
			return nil
		},
	}
}

// isBundlePath returns true if the argument looks like a local bundle file path.
func isBundlePath(arg string) bool {
	exts := []string{".tar.gz", ".tgz", ".tar.zst", ".zip"}
	for _, ext := range exts {
		if strings.HasSuffix(arg, ext) {
			if _, err := os.Stat(arg); err == nil {
				return true
			}
		}
	}
	return false
}

// progressMsg is sent by the assembler through a progress callback.
type progressMsg struct {
	stage   string
	message string
}

// spinnerFrames are the animation frames for the spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// stageGlyph returns a short glyph for the given stage.
func stageGlyph(stage string) string {
	switch stage {
	case "resolve":
		return "→"
	case "deps":
		return "→"
	case "download":
		return "↓"
	case "build":
		return "⚙"
	case "configure":
		return "⚙"
	case "make":
		return "⚙"
	case "install":
		return "↑"
	case "skip":
		return "↷"
	case "patch":
		return "✎"
	case "error":
		return "✗"
	case "done":
		return "✓"
	default:
		return "·"
	}
}

// printJSON marshals v to stdout as indented JSON. All JSON responses include
// a schema_version field for forward compatibility.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// jsonResponse wraps a payload with a schema version.
type jsonResponse struct {
	SchemaVersion int `json:"schema_version"`
	Data          any `json:"data"`
}
