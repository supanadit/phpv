package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/internal/utils"
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
		Use:   "doctor [version]",
		Short: "Check system dependencies",
		Long:  `Check if the system has all the required dependencies for building PHP. Optionally analyze extension availability for a specific PHP version.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
				// Resolve version constraint to exact version for extension analysis
				if sources, err := handler.Source.GetVersions(); err == nil {
					var phpVersions []string
					for _, src := range sources {
						if src.Name == "php" {
							phpVersions = append(phpVersions, src.Version)
						}
					}
					if resolved, err := utils.ResolveVersionConstraint(phpVersions, version); err == nil {
						version = resolved
					}
				}
			}

			result, err := handler.DoctorV2(version)
			if err != nil {
				return err
			}

			fmt.Println("═══ Build Tools ═══")
			for _, t := range result.BuildTools {
				if t.Available {
					fmt.Printf("  ✓ %-14s %s\n", t.Name, t.Version)
				} else {
					fmt.Printf("  ✗ %-14s %s\n", t.Name, t.Suggestion)
				}
			}

			fmt.Println("\n═══ System Libraries ═══")
			for _, l := range result.LibChecks {
				if l.Available {
					fmt.Printf("  ✓ %-14s %s\n", l.Name, l.Version)
				} else {
					fmt.Printf("  ✗ %-14s %s\n", l.Name, l.Suggestion)
				}
			}

			if version != "" && len(result.Extensions) > 0 {
				fmt.Printf("\n═══ PHP %s Extensions ═══\n", version)
				for _, e := range result.Extensions {
					switch e.Status {
					case "builtin":
						fmt.Printf("  · %-18s built-in\n", e.Extension)
					case "system":
						if e.ExpectedVer != "" {
							fmt.Printf("  ✓ %-18s system (%s, need %s)\n", e.Extension, e.SystemVer, e.ExpectedVer)
						} else {
							fmt.Printf("  ✓ %-18s system (%s)\n", e.Extension, e.SystemVer)
						}
					case "build":
						fmt.Printf("  ◷ %-18s buildable (phpv builds %s)\n", e.Extension, e.Package)
					case "mismatch":
						fmt.Printf("  ⚠ %-18s version mismatch: system %s, need %s\n", e.Extension, e.SystemVer, e.ExpectedVer)
					case "missing":
						fmt.Printf("  ✗ %-18s %s\n", e.Extension, e.Suggestion)
					}
				}
			}

			fmt.Println("\n" + result.Summary)
			return nil
		},
	}

	root.AddCommand(buildToolsCmd)
	buildToolsCmd.AddCommand(buildToolsCleanCmd)
	root.AddCommand(upgradeCmd)
	root.AddCommand(doctorCmd)
}
