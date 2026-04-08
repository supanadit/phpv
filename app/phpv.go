package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
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
	"github.com/supanadit/phpv/internal/terminal"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type silentLogger struct{}

func (s *silentLogger) LogEvent(event fxevent.Event) {}

const (
	minBoxWidth = 64
	phpvVersion = "0.1.0"
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
		padding := max(width-len(line), 0)
		fmt.Println("|" + line + strings.Repeat(" ", padding) + "|")
	}
	fmt.Println(bottom)
}

func printInstallSummary(version string, forge domain.Forge) {
	binaryPath := filepath.Join(forge.Prefix, "bin", "php")

	labelVersion := "Version:"
	labelBinary := "Binary:"

	contentWidth := minBoxWidth - 2

	headerWidth := len(labelVersion) + 1 + len(version)
	binaryWidth := len(labelBinary) + 1 + len(binaryPath)

	boxWidth := minBoxWidth
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

	header := "                    PHP Installation Summary"
	versionContent := fmt.Sprintf("%s %s", labelVersion, version)
	binaryContent := fmt.Sprintf("%s %s", labelBinary, displayBinaryPath)

	fmt.Println()
	printBox(boxWidth, []string{
		"",
		header,
		versionContent,
		binaryContent,
	})
}

func main() {
	viper.AutomaticEnv()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("You must have a home directory set for phpv to work")
	}
	viper.SetDefault("PHPV_ROOT", filepath.Join(homeDir, ".phpv"))

	opts := []fx.Option{
		fx.WithLogger(func() fxevent.Logger { return &silentLogger{} }),
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
		SiloRepo:  sil,
		Verbose:   verbose,
	}, nil
}

func run(
	shutdowner fx.Shutdowner,
	sil *disk.SiloRepository,
	cfg bundler.BundlerServiceConfig,
	flagResolverRepo domain.FlagResolverRepository,
	src source.SourceRepository,
) {
	if err := sil.EnsurePaths(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		shutdowner.Shutdown(fx.ExitCode(1))
		return
	}

	bundlerRepo := disk.NewBundlerRepository(cfg, flagResolverRepo)
	handler := terminal.NewHandler(bundlerRepo, sil, src)

	rootCmd := &cobra.Command{
		Use:   "phpv",
		Short: "PHP Version Manager",
		Long:  `A PHP Version Manager for building and managing multiple PHP versions from source.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	installCmd := &cobra.Command{
		Use:   "install <version>",
		Short: "Install a PHP version",
		Long:  `Install the latest PHP version matching the given version constraint. Examples: phpv install 8.5, phpv install 8.4, phpv install 8`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			compiler, _ := cmd.Flags().GetString("compiler")
			fresh, _ := cmd.Flags().GetBool("fresh")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			quiet, _ := cmd.Flags().GetBool("quiet")
			force, _ := cmd.Flags().GetBool("force")

			if dryRun {
				fmt.Println("[dry-run] Would install PHP", args[0])
				return nil
			}

			if jsonOutput {
				fmt.Printf(`{"command":"install","version":"%s","compiler":"%s","fresh":%t}\n`, args[0], compiler, fresh)
			}

			if quiet {
				verbose = false
			}

			forge, err := handler.Install(args[0], compiler, verbose, fresh || force)
			if err != nil {
				return err
			}

			if !quiet {
				printInstallSummary(args[0], forge)
			}
			return nil
		},
	}
	installCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	installCmd.Flags().String("compiler", "", "Force a specific compiler (e.g., zig, gcc)")
	installCmd.Flags().Bool("fresh", false, "Clean existing installation before installing")
	installCmd.Flags().Bool("dry-run", false, "Preview install steps without executing")
	installCmd.Flags().Bool("json", false, "JSON output for machine parsing")
	installCmd.Flags().BoolP("quiet", "q", false, "Suppress non-essential output")
	installCmd.Flags().Bool("force", false, "Force rebuild even if already installed")

	useCmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a PHP version in current shell",
		Long:  `Generate shims for the specified PHP version and print PATH instructions.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := handler.Use(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Switched to PHP %s\n", result.ExactVersion)
			fmt.Printf("Add to PATH: export PATH=%s:$PATH\n", result.ShimPath)
			fmt.Println("Or restart your shell to use the shims")
			return nil
		},
	}

	defaultCmd := &cobra.Command{
		Use:   "default <version>",
		Short: "Set default PHP version",
		Long:  `Set the specified PHP version as the default version.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := handler.SetDefault(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("PHP %s is now the default\n", args[0])
			return nil
		},
	}

	versionsCmd := &cobra.Command{
		Use:   "versions",
		Short: "List installed PHP versions",
		Long:  `List all PHP versions that are currently installed.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			versions, err := handler.ListInstalled()
			if err != nil {
				return err
			}
			if len(versions) == 0 {
				fmt.Println("No PHP versions installed")
				return nil
			}
			currentDefault, _ := handler.GetDefault()
			fmt.Println("Installed PHP versions:")
			for _, v := range versions {
				if v == currentDefault {
					fmt.Printf("  * %s (default)\n", v)
				} else {
					fmt.Printf("    %s\n", v)
				}
			}
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available PHP versions",
		Long:  `List all PHP versions available to install from remote sources.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sources, err := handler.ListAvailable()
			if err != nil {
				return err
			}
			if len(sources) == 0 {
				fmt.Println("No PHP versions available")
				return nil
			}
			var phpVersions []string
			for _, src := range sources {
				phpVersions = append(phpVersions, src.Version)
			}
			sort.Slice(phpVersions, func(i, j int) bool {
				vi := utils.ParseVersion(phpVersions[i])
				vj := utils.ParseVersion(phpVersions[j])
				return utils.CompareVersions(vi, vj) > 0
			})
			fmt.Println("Available PHP versions:")
			for _, v := range phpVersions {
				fmt.Printf("  %s\n", v)
			}
			return nil
		},
	}

	whichCmd := &cobra.Command{
		Use:   "which",
		Short: "Show path to current PHP",
		Long:  `Print the full path to the currently active PHP binary.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			phpPath, err := handler.Which()
			if err != nil {
				return err
			}
			if phpPath == "" {
				fmt.Println("No default PHP version set")
				return nil
			}
			fmt.Println(phpPath)
			return nil
		},
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall <version>",
		Short: "Uninstall a PHP version",
		Long:  `Remove the specified PHP version and its dependencies. Build-tools that are no longer used will be cleaned up.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := handler.Uninstall(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Uninstalled PHP %s\n", result.Version)
			if len(result.RemovedTools) > 0 {
				fmt.Println("Removed unused build-tools:")
				for _, tool := range result.RemovedTools {
					fmt.Printf("  - %s\n", tool)
				}
			}
			if result.WasDefault {
				fmt.Println("Cleared default PHP version")
			}
			return nil
		},
	}

	buildToolsCmd := &cobra.Command{
		Use:   "build-tools",
		Short: "Manage build-tools",
		Long:  `Manage build-tools used for compiling PHP and its dependencies.`,
	}

	buildToolsCleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove unused build-tools",
		Long:  `Remove build-tools that are no longer used by any PHP version. Use --dry-run to see what would be removed without actually removing.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			result, err := handler.CleanBuildTools(dryRun)
			if err != nil {
				return err
			}
			if dryRun {
				if len(result.WillRemove) == 0 {
					fmt.Println("No unused build-tools to remove")
				} else {
					fmt.Println("Would remove unused build-tools:")
					for _, tool := range result.WillRemove {
						fmt.Printf("  - %s\n", tool)
					}
				}
			} else {
				if len(result.Removed) == 0 {
					fmt.Println("No unused build-tools to remove")
				} else {
					fmt.Println("Removed unused build-tools:")
					for _, tool := range result.Removed {
						fmt.Printf("  - %s\n", tool)
					}
				}
			}
			return nil
		},
	}
	buildToolsCleanCmd.Flags().Bool("dry-run", false, "Show what would be removed without actually removing")

	upgradeCmd := &cobra.Command{
		Use:   "upgrade [constraint]",
		Short: "Upgrade to the latest PHP version",
		Long:  `Upgrade the installed PHP version matching the constraint to the latest available version. If no constraint is given, upgrades the default version.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			constraint := ""
			if len(args) > 0 {
				constraint = args[0]
			} else {
				defaultVer, err := handler.GetDefault()
				if err != nil {
					return err
				}
				if defaultVer == "" {
					return fmt.Errorf("no default version set, specify a version to upgrade")
				}
				constraint = defaultVer
			}
			result, err := handler.Upgrade(constraint)
			if err != nil {
				return err
			}
			fmt.Printf("Upgraded PHP %s -> %s\n", result.FromVersion, result.ToVersion)
			printInstallSummary(result.ToVersion, result.Forge)
			return nil
		},
	}

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system dependencies",
		Long:  `Check if the system has all the required dependencies for building PHP.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := handler.Doctor()
			if err != nil {
				return err
			}
			if len(result.Issues) > 0 {
				fmt.Println("Issues found:")
				for _, issue := range result.Issues {
					fmt.Printf("  [%s] %s\n", issue.Category, issue.Message)
				}
			}
			fmt.Println("Doctor check complete")
			return nil
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show phpv version",
		Long:  `Show the version of phpv being used.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("phpv %s\n", phpvVersion)
			return nil
		},
	}

	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for the specified shell.
		
To load completions:

Bash:

  $ source <(phpv completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ phpv completion bash > /etc/bash_completion.d/phpv
  # macOS:
  $ phpv completion bash > /usr/local/etc/bash_completion.d/phpv

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ phpv completion zsh > "${fpath[1]}/_phpv"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ phpv completion fish | source

  # To load completions for each session, execute once:
  $ phpv completion fish > ~/.config/fish/completions/phpv.fish

PowerShell:

  PS> phpv completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  PS> phpv completion powershell > phpv.ps1
  # and source this file from your PowerShell profile.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(defaultCmd)
	rootCmd.AddCommand(versionsCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(whichCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(buildToolsCmd)
	buildToolsCmd.AddCommand(buildToolsCleanCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)

	rootCmd.Version = phpvVersion

	if err := rootCmd.Execute(); err != nil {
		shutdowner.Shutdown(fx.ExitCode(1))
		return
	}

	shutdowner.Shutdown()
}
