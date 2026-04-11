package terminal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func registerVersionCommands(root *cobra.Command, handler *TerminalHandler) {
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

	root.AddCommand(useCmd)
	root.AddCommand(shellUseCmd)
	root.AddCommand(autoDetectCmd)
	root.AddCommand(autoDetectResolveCmd)
	root.AddCommand(writeDefaultCmd)
	root.AddCommand(defaultCmd)
	root.AddCommand(versionsCmd)
	root.AddCommand(listCmd)
	root.AddCommand(whichCmd)
	root.AddCommand(uninstallCmd)
}
