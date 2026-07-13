package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundle"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

// PHPHandler registers cobra commands and delegates to services.
type PHPHandler struct {
	siloSvc      *silo.Service
	assemblerSvc *assembler.Service
	registrySvc  *registry.Service
	bundleSvc    *bundle.Service
}

// NewPHPHandler registers all PHP subcommands onto the given root command.
func NewPHPHandler(rootCmd *cobra.Command, siloSvc *silo.Service, assemblerSvc *assembler.Service, registrySvc *registry.Service, bundleSvc *bundle.Service) {
	h := &PHPHandler{
		siloSvc:      siloSvc,
		assemblerSvc: assemblerSvc,
		registrySvc:  registrySvc,
		bundleSvc:    bundleSvc,
	}
	rootCmd.AddCommand(h.downloadCmd())
	rootCmd.AddCommand(h.installCmd())
	rootCmd.AddCommand(h.versionsCmd())
	rootCmd.AddCommand(h.listCmd())
	rootCmd.AddCommand(h.whichCmd())
	rootCmd.AddCommand(h.defaultCmd())
	rootCmd.AddCommand(h.useCmd())
	rootCmd.AddCommand(h.shareCmd())
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
		Long:  "Download, build, and install a specific version of PHP with all its dependencies.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.install,
	}
	cmd.Flags().String("from", "", "Install from a bundle file instead of building from source")
	cmd.Flags().Bool("static", false, "Build with static linking for cross-distro portability")
	return cmd
}

func (h *PHPHandler) install(cmd *cobra.Command, args []string) error {
	version := args[0]

	fromBundle, _ := cmd.Flags().GetString("from")
	if fromBundle != "" {
		fmt.Printf("Installing PHP %s from bundle %s...\n", version, fromBundle)
		if err := h.bundleSvc.Import(fromBundle, version); err != nil {
			return fmt.Errorf("install from bundle failed: %w", err)
		}
		fmt.Printf("✓ PHP %s installed from bundle\n", version)
		return nil
	}

	static, _ := cmd.Flags().GetBool("static")

	fmt.Printf("Installing PHP %s...\n\n", version)

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
					if current != "" {
						fmt.Fprintf(os.Stdout, "\r\033[2K%s\n", current)
					}
					return
				}
				current = fmt.Sprintf("%s %s", stageGlyph(msg.stage), msg.message)
				fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
			case <-ticker.C:
				if current != "" {
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
				}
				frame++
			}
		}
	}()

	result, err := h.assemblerSvc.Assemble("php", version, static, func(stage, message string) {
		progressCh <- progressMsg{stage: stage, message: message}
	})
	close(progressCh)
	<-doneCh

	if err != nil {
		fmt.Println()
		return fmt.Errorf("install failed: %w", err)
	}
	fmt.Println()
	fmt.Printf("✓ PHP %s installed at %s\n", result.Version, result.Prefix)
	return nil
}

// versionsCmd lists installed PHP versions.
func (h *PHPHandler) versionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "List installed PHP versions",
		Args:  cobra.NoArgs,
		RunE:  h.versions,
	}
}

func (h *PHPHandler) versions(cmd *cobra.Command, args []string) error {
	silo := h.siloSvc.GetSilo()
	versionsDir := filepath.Join(silo.Root, "versions")

	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No PHP versions installed.")
			return nil
		}
		return fmt.Errorf("read versions dir: %w", err)
	}

	defaultVer, _ := h.siloSvc.GetDefault()

	var installed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		phpBin := filepath.Join(versionsDir, e.Name(), "output", "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			installed = append(installed, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(installed)))

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
	return &cobra.Command{
		Use:   "list",
		Short: "List available PHP versions",
		Args:  cobra.NoArgs,
		RunE:  h.listAvailable,
	}
}

func (h *PHPHandler) listAvailable(cmd *cobra.Command, args []string) error {
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
	return &cobra.Command{
		Use:   "default <version>",
		Short: "Set global default PHP version",
		Args:  cobra.ExactArgs(1),
		RunE:  h.setDefault,
	}
}

func (h *PHPHandler) setDefault(cmd *cobra.Command, args []string) error {
	version := args[0]

	// Verify the version is installed.
	silo := h.siloSvc.GetSilo()
	phpBin := filepath.Join(silo.Root, "versions", version, "output", "bin", "php")
	if _, err := os.Stat(phpBin); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", version, version)
	}

	if err := h.siloSvc.SetDefault(version); err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	fmt.Printf("✓ Default PHP version set to %s\n", version)
	return nil
}

// useCmd switches the active PHP version for the current session.
func (h *PHPHandler) useCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a PHP version",
		Long:  "Switch to a specific PHP version. Use 'system' to use the system PHP.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.use,
	}
}

func (h *PHPHandler) use(cmd *cobra.Command, args []string) error {
	version := args[0]

	if version == "system" {
		// Check that system PHP exists.
		systemPHP, err := exec.LookPath("php")
		if err != nil {
			return fmt.Errorf("system PHP not found in PATH")
		}
		// Make sure it's not a phpv shim.
		silo := h.siloSvc.GetSilo()
		phpvBin := filepath.Join(silo.Root, "bin", "php")
		if systemPHP == phpvBin {
			return fmt.Errorf("system PHP is managed by phpv; use a specific version instead")
		}
		fmt.Printf("→ Using system PHP at %s\n", systemPHP)
		fmt.Println("Run `phpv init` in your shell to enable version switching.")
		return nil
	}

	// Resolve the version constraint to an exact installed version.
	exactVersion, err := h.resolveInstalledVersion(version)
	if err != nil {
		return err
	}

	// Set as default.
	if err := h.siloSvc.SetDefault(exactVersion); err != nil {
		return fmt.Errorf("set default: %w", err)
	}

	silo := h.siloSvc.GetSilo()
	phpBin := filepath.Join(silo.Root, "versions", exactVersion, "output", "bin", "php")
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
		phpBin := filepath.Join(silo.Root, "versions", envVer, "output", "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 2. Check .phpvrc in current or parent directories.
	if ver := findPhpvrc(); ver != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "versions", ver, "output", "bin", "php")
		if _, err := os.Stat(phpBin); err == nil {
			return phpBin, nil
		}
	}

	// 3. Check default.
	defaultVer, err := h.siloSvc.GetDefault()
	if err == nil && defaultVer != "" {
		silo := h.siloSvc.GetSilo()
		phpBin := filepath.Join(silo.Root, "versions", defaultVer, "output", "bin", "php")
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

// resolveInstalledVersion resolves a version constraint to an exact installed version.
func (h *PHPHandler) resolveInstalledVersion(constraint string) (string, error) {
	silo := h.siloSvc.GetSilo()
	versionsDir := filepath.Join(silo.Root, "versions")

	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return "", fmt.Errorf("no PHP versions installed")
	}

	var installed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		phpBin := filepath.Join(versionsDir, e.Name(), "output", "bin", "php")
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

	// Major.minor match (e.g., "8.4" → latest 8.4.x).
	if strings.Count(constraint, ".") == 1 {
		prefix := constraint + "."
		var candidates []string
		for _, v := range installed {
			if strings.HasPrefix(v, prefix) {
				candidates = append(candidates, v)
			}
		}
		if len(candidates) > 0 {
			sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
			return candidates[0], nil
		}
	}

	return "", fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", constraint, constraint)
}

// findPhpvrc walks up from the current directory looking for a .phpvrc file.
func findPhpvrc() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		rcPath := filepath.Join(dir, ".phpvrc")
		if data, err := os.ReadFile(rcPath); err == nil {
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
	return cmd
}

func (h *PHPHandler) share(cmd *cobra.Command, args []string) error {
	version := args[0]
	output, _ := cmd.Flags().GetString("output")

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
