package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"go.uber.org/fx"
)

func ExecuteCobra(handler *TerminalHandler, shutdowner fx.Shutdowner) error {
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
			extStr, _ := cmd.Flags().GetString("ext")

			extensions := parseExtensions(extStr)

			if dryRun {
				fmt.Println("[dry-run] Would install PHP", args[0])
				if len(extensions) > 0 {
					fmt.Printf("[dry-run] Extensions: %s\n", extStr)
				}
				return nil
			}

			if jsonOutput {
				fmt.Printf(`{"command":"install","version":"%s","compiler":"%s","extensions":%v,"fresh":%t}`+"\n", args[0], compiler, extensions, fresh)
				return nil
			}

			if quiet {
				verbose = false
			}

			forge, err := handler.Install(args[0], compiler, extensions, verbose, fresh || force)
			if err != nil {
				return err
			}

			if !quiet {
				PrintInstallSummary(args[0], forge)
			}
			return nil
		},
	}
	installCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	installCmd.Flags().String("compiler", "", "Force a specific compiler (e.g., zig, gcc)")
	installCmd.Flags().String("ext", "", "Comma-separated list of bundled extensions to enable (e.g., opcache,mbstring,curl)")
	installCmd.Flags().Bool("fresh", false, "Clean existing installation before installing")
	installCmd.Flags().Bool("dry-run", false, "Preview install steps without executing")
	installCmd.Flags().Bool("json", false, "JSON output for machine parsing")
	installCmd.Flags().BoolP("quiet", "q", false, "Suppress non-essential output")
	installCmd.Flags().Bool("force", false, "Force rebuild even if already installed")

	rebuildCmd := &cobra.Command{
		Use:   "rebuild <version>",
		Short: "Rebuild PHP with different extensions without reinstalling dependencies",
		Long: `Rebuild an existing PHP installation with new extension flags. This is faster than 'install --fresh' because it preserves the downloaded archive and extracted source, only recompiling PHP with the new configuration.

Example:
  phpv rebuild 8 --ext phar,iconv,filter,fileinfo
  phpv rebuild 8 --ext phar,iconv,filter,fileinfo,dom,session`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			compiler, _ := cmd.Flags().GetString("compiler")
			extStr, _ := cmd.Flags().GetString("ext")
			quiet, _ := cmd.Flags().GetBool("quiet")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			extensions := parseExtensions(extStr)

			if quiet {
				verbose = false
			}

			if jsonOutput {
				fmt.Printf(`{"command":"rebuild","version":"%s","compiler":"%s","extensions":%v}`+"\n", args[0], compiler, extensions)
				return nil
			}

			forge, err := handler.Rebuild(args[0], compiler, extensions, verbose)
			if err != nil {
				return err
			}

			if !quiet {
				PrintInstallSummary(args[0], forge)
			}
			return nil
		},
	}
	rebuildCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rebuildCmd.Flags().String("compiler", "", "Force a specific compiler (e.g., zig, gcc)")
	rebuildCmd.Flags().String("ext", "", "Comma-separated list of bundled extensions to enable (e.g., opcache,mbstring,curl)")
	rebuildCmd.Flags().Bool("json", false, "JSON output for machine parsing")
	rebuildCmd.Flags().BoolP("quiet", "q", false, "Suppress non-essential output")

	useCmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a PHP version for the current session",
		Long:  `Switch to the specified PHP version. This sets PHPV_CURRENT for the current session only. Use 'phpv default' to set a global default. Use 'phpv use system' to use the system-installed PHP.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var result *UseResult
			var err error

			if args[0] == "system" {
				result, err = handler.UseSystem()
			} else {
				result, err = handler.Use(args[0])
			}
			if err != nil {
				return err
			}
			fmt.Printf("PHP %s is now active in this session\n", result.ExactVersion)
			fmt.Printf("To use this version in new terminals, run:\n")
			fmt.Printf("  export PHPV_CURRENT=%s\n", result.ExactVersion)
			fmt.Printf("Or add 'export PATH=%s:$PATH' and use .phpvrc or composer.json for auto-switching\n", result.ShimPath)
			fmt.Printf("To set a global default, use: phpv default %s\n", result.ExactVersion)
			return nil
		},
	}

	shellUseCmd := &cobra.Command{
		Use:    "shell-use <version>",
		Hidden: true,
		Short:  "Internal command for shell integration",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			err := handler.ShellUse(version)
			if err != nil {
				return err
			}
			fmt.Printf("export PHPV_CURRENT=%s\n", version)
			return nil
		},
	}

	autoDetectCmd := &cobra.Command{
		Use:    "auto-detect",
		Hidden: true,
		Short:  "Detect PHP version from composer.json",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := handler.AutoDetect()
			if err != nil {
				os.Exit(1)
			}
			fmt.Println(version)
			return nil
		},
	}

	autoDetectResolveCmd := &cobra.Command{
		Use:    "auto-detect-resolve [constraint]",
		Hidden: true,
		Short:  "Detect and resolve PHP version from composer.json",
		Args:   cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			constraint := ""
			if len(args) > 0 {
				constraint = args[0]
			}
			version, err := handler.AutoDetectResolve(constraint)
			if err != nil {
				os.Exit(1)
			}
			fmt.Println(version)
			return nil
		},
	}

	writeDefaultCmd := &cobra.Command{
		Use:    "write-default <version>",
		Hidden: true,
		Short:  "Internal command to write default version",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return handler.SetDefault(args[0])
		},
	}

	initCmd := &cobra.Command{
		Use:   "init [bash|zsh|fish]",
		Short: "Output shell initialization code",
		Long: `Output shell initialization code for the specified shell. Add this to your shell RC file or eval it:

    eval "$(phpv init)"

After initialization, you can use 'phpv use <version>' to switch PHP versions in the current shell.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := "bash"
			if len(args) > 0 {
				shell = args[0]
			}
			initCode, err := handler.GetInitCode(shell)
			if err != nil {
				return err
			}
			fmt.Print(initCode)
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
			result, err := handler.ListVersionsFormatted()
			if err != nil {
				return err
			}

			printer := &VersionsPrinter{
				Versions:   result.Versions,
				DefaultVer: result.DefaultVer,
			}
			printer.Print()
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available PHP versions",
		Long:  `List all PHP versions available to install from remote sources.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := handler.ListAvailableFormatted()
			if err != nil {
				return err
			}
			if len(result.Versions) == 0 {
				fmt.Println("No PHP versions available")
				return nil
			}
			fmt.Println("Available PHP versions:")
			for _, v := range result.Versions {
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
			PrintInstallSummary(result.ToVersion, result.Forge)
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
			fmt.Printf("phpv %s\n", domain.AppVersion)
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
	rootCmd.AddCommand(rebuildCmd)
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
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(shellUseCmd)
	rootCmd.AddCommand(writeDefaultCmd)
	rootCmd.AddCommand(autoDetectCmd)
	rootCmd.AddCommand(autoDetectResolveCmd)

	peclCmd := &cobra.Command{
		Use:   "pecl",
		Short: "Manage PECL extensions",
		Long:  `Manage PECL extensions for the currently active PHP version. Use 'phpv use <version>' to switch PHP versions first.`,
	}

	peclInstallCmd := &cobra.Command{
		Use:   "install <archive.tgz>",
		Short: "Install a PECL extension from archive",
		Long: `Install a PECL extension from a downloaded .tgz archive.
First download the extension archive from https://pecl.php.net, then run:
  phpv pecl install /path/to/extension-1.2.3.tgz`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			yes, _ := cmd.Flags().GetBool("yes")

			defaultVer, err := handler.GetDefault()
			if err != nil {
				return fmt.Errorf("failed to get default PHP version: %w", err)
			}
			if defaultVer == "" {
				return fmt.Errorf("no default PHP version set. Run 'phpv use <version>' first")
			}

			if !yes {
				fmt.Printf("Installing %s for PHP %s? [y/N] ", args[0], defaultVer)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			result, err := handler.PECLInstall(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Installed %s %s\n", result.Name, result.Version)
			fmt.Printf("  Extension directory: %s\n", result.InstallDir)
			return nil
		},
	}
	peclInstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	peclListCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed PECL extensions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			extensions, err := handler.PECLList()
			if err != nil {
				return err
			}
			if len(extensions) == 0 {
				fmt.Println("No PECL extensions installed")
				return nil
			}
			fmt.Println("Installed PECL extensions:")
			for _, ext := range extensions {
				fmt.Printf("  - %s\n", ext)
			}
			return nil
		},
	}

	peclUninstallCmd := &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Uninstall a PECL extension",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := handler.PECLUninstall(args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ Uninstalled %s\n", args[0])
			return nil
		},
	}

	peclCmd.AddCommand(peclInstallCmd)
	peclCmd.AddCommand(peclListCmd)
	peclCmd.AddCommand(peclUninstallCmd)
	rootCmd.AddCommand(peclCmd)

	pharCmd := &cobra.Command{
		Use:   "phar",
		Short: "Manage PHAR files",
		Long:  `Manage PHAR files like Composer for the currently active PHP version.`,
	}

	pharInstallCmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a PHAR file",
		Long:  `Download and install a PHAR file (e.g., composer).`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version, _ := cmd.Flags().GetString("version")
			result, err := handler.PharInstall(args[0], version)
			if err != nil {
				return err
			}
			verb := "Installed"
			if result.Updated {
				verb = "Updated"
			}
			fmt.Printf("%s %s %s\n", verb, result.Name, result.Version)
			fmt.Printf("  Location: %s\n", result.Path)
			return nil
		},
	}
	pharInstallCmd.Flags().StringP("version", "v", "", "Specific version to install")

	pharUpdateCmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a PHAR file",
		Long:  `Update an existing PHAR file to the latest or specified version.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version, _ := cmd.Flags().GetString("version")
			result, err := handler.PharUpdate(args[0], version)
			if err != nil {
				return err
			}
			fmt.Printf("Updated %s %s\n", result.Name, result.Version)
			fmt.Printf("  Location: %s\n", result.Path)
			return nil
		},
	}
	pharUpdateCmd.Flags().StringP("version", "v", "", "Specific version to update to")

	pharRemoveCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a PHAR file",
		Long:  `Remove an installed PHAR file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := handler.PharRemove(args[0]); err != nil {
				return err
			}
			fmt.Printf("Removed %s\n", args[0])
			return nil
		},
	}

	pharWhichCmd := &cobra.Command{
		Use:   "which <name>",
		Short: "Show path to installed PHAR",
		Long:  `Show the full path to an installed PHAR file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := handler.PharWhich(args[0])
			if err != nil {
				return err
			}
			if path == "" {
				return fmt.Errorf("%s not found", args[0])
			}
			fmt.Println(path)
			return nil
		},
	}

	pharListCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed PHAR files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			phars, err := handler.PharList()
			if err != nil {
				return err
			}
			if len(phars) == 0 {
				fmt.Println("No PHAR files installed")
				return nil
			}
			fmt.Println("Installed PHAR files:")
			for _, phar := range phars {
				fmt.Printf("  - %s\n", phar)
			}
			return nil
		},
	}

	pharCmd.AddCommand(pharInstallCmd)
	pharCmd.AddCommand(pharUpdateCmd)
	pharCmd.AddCommand(pharRemoveCmd)
	pharCmd.AddCommand(pharWhichCmd)
	pharCmd.AddCommand(pharListCmd)
	rootCmd.AddCommand(pharCmd)

	rootCmd.Version = domain.AppVersion

	if err := rootCmd.Execute(); err != nil {
		shutdowner.Shutdown(fx.ExitCode(1))
		return err
	}

	return nil
}

func parseExtensions(extStr string) []string {
	if extStr == "" {
		return nil
	}
	extensions := []string{}
	for _, ext := range strings.Split(extStr, ",") {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			extensions = append(extensions, ext)
		}
	}
	return extensions
}
