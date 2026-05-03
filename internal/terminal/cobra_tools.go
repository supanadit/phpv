package terminal

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/internal/compiler"
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
					} else {
						// Invalid version constraint - show error but continue with system check
						fmt.Printf("Warning: Version %q not found, checking system dependencies only\n", version)
						version = ""
					}
				}
			}

			result, err := handler.DoctorV2(version)
			if err != nil {
				return err
			}

			// ── Readiness ──
			fmt.Println("═══ System Readiness ═══")
			verdictIcon := "✓"
			switch result.Verdict {
			case "blocked":
				verdictIcon = "✗"
			case "minor":
				verdictIcon = "⚡"
			}
			fmt.Printf("  %s %s\n", verdictIcon, result.VerdictMsg)

			// Show compiler status for each major PHP version
			// Buildable is determined by: make available AND (gcc available OR zig available OR zig auto-downloaded)
			hasMake := false
			for _, tool := range result.BuildTools {
				if tool.Name == "make" && tool.Available {
					hasMake = true
					break
				}
			}

			for _, v := range result.CompilerByMajor {
				statusIcon := "✓"
				statusMsg := fmt.Sprintf("PHP %d.x buildable    (uses %s)", v.MajorVersion, v.Compiler)
				if !v.Available {
					statusIcon = "✗"
					statusMsg = fmt.Sprintf("PHP %d.x not buildable  (no compiler)", v.MajorVersion)
				} else if v.AutoDownload {
					statusMsg = fmt.Sprintf("PHP %d.x buildable    (uses %s, auto-downloaded)", v.MajorVersion, v.Compiler)
				}
				fmt.Printf("  %s %s\n", statusIcon, statusMsg)
			}

			if result.QuickFix != "" {
				fmt.Printf("\n═══ Quick Fix ═══\n")
				fmt.Printf("  %s\n", result.QuickFix)
			}

			// Recommendation section
			if version == "" && result.Verdict != "blocked" && hasMake {
				fmt.Printf("\n═══ Recommendation ═══\n")

				// Find the best PHP version to recommend (prefer highest major that's buildable)
				var recommendedMajor int
				var recommendedCompiler string
				for i := len(result.CompilerByMajor) - 1; i >= 0; i-- {
					if result.CompilerByMajor[i].Available {
						recommendedMajor = result.CompilerByMajor[i].MajorVersion
						recommendedCompiler = result.CompilerByMajor[i].Compiler
						break
					}
				}

				if recommendedMajor > 0 {
					fmt.Printf("  phpv install %d\n", recommendedMajor)
					if recommendedCompiler == string(compiler.CompilerTypeZig) {
						autoNote := ""
						if result.CompilerByMajor[len(result.CompilerByMajor)-1].AutoDownload {
							autoNote = " (auto-downloaded)"
						}
						fmt.Printf("    Uses zig compiler%s\n", autoNote)
					} else {
						fmt.Println("    Uses gcc (fastest build)")
					}
					fmt.Println("    Add --ext openssl,curl,mbstring,intl for common extensions.")
				}
			}

			// ── Group all items by category ──
			allItems := append(result.BuildTools, result.LibChecks...)
			var available, phpvHandles, sysReq []DoctorCheckItem
			for _, item := range allItems {
				switch item.Category {
				case "available":
					available = append(available, item)
				case "autodownload", "buildable":
					phpvHandles = append(phpvHandles, item)
				default:
					sysReq = append(sysReq, item)
				}
			}

			// ── Available on System ──
			if len(available) > 0 {
				fmt.Printf("\n═══ Available on System (%d) ═══\n", len(available))
				for _, item := range available {
					fmt.Printf("  ✓ %-14s %s\n", item.Name, item.Version)
				}
			}

			// ── phpv Will Handle ──
			if len(phpvHandles) > 0 {
				fmt.Printf("\n═══ phpv Will Handle (%d) ═══\n", len(phpvHandles))
				for _, item := range phpvHandles {
					fmt.Printf("  ◷ %-14s %s\n", item.Name, item.Suggestion)
				}
			}

			// ── System Packages Required ──
			if len(sysReq) > 0 {
				fmt.Printf("\n═══ System Packages Required (%d) ═══\n", len(sysReq))
				for _, item := range sysReq {
					fmt.Printf("  ✗ %-14s %s\n", item.Name, item.Suggestion)
				}
			}

			// ── PHP Install (version-specific) ──
			if version != "" && result.PHPInstall != nil {
				fmt.Printf("\n═══ PHP %s Installation ═══\n", version)
				if result.PHPInstall.Installed {
					fmt.Printf("  ✓ Installed at: %s\n", result.PHPInstall.BinaryPath)
					if result.PHPInstall.ConfigFlags != "" {
						fmt.Printf("  Configure: %s\n", result.PHPInstall.ConfigFlags)
					}
					if n := len(result.PHPInstall.EnabledExts); n > 0 {
						fmt.Printf("  Enabled extensions (%d): %s\n", n, strings.Join(result.PHPInstall.EnabledExts, ", "))
					}
				} else {
					fmt.Printf("  ✗ Not installed\n")
				}
			}

			// ── Extension Analysis (version-specific) ──
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
