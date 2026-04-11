package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerToolsCommands(root *cobra.Command, handler *TerminalHandler) {
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

	root.AddCommand(buildToolsCmd)
	buildToolsCmd.AddCommand(buildToolsCleanCmd)
	root.AddCommand(upgradeCmd)
	root.AddCommand(doctorCmd)
}
