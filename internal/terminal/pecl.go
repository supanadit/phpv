package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) peclCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pecl",
		Short: "Manage PECL extensions",
		Long: `Manage PECL extensions for an installed PHP version.

Install from pecl.php.net by name:
  phpv pecl install redis 8.4.0

Install from a local archive:
  phpv pecl install /path/to/redis-6.0.2.tgz 8.4.0

List installed PECL extensions:
  phpv pecl list 8.4.0

Uninstall a PECL extension:
  phpv pecl uninstall redis 8.4.0`,
	}

	installCmd := &cobra.Command{
		Use:   "install <name|archive> [version]",
		Short: "Install a PECL extension",
		Long: `Install a PECL extension. Source can be a package name (auto-downloads
from pecl.php.net) or a local .tgz/.tar.gz/.tar.bz2 archive path.

The PHP version defaults to the active version; use --version to target a
specific one. The positional [version] argument is kept for backwards
compatibility.

Examples:
  phpv pecl install redis
  phpv pecl install redis --version 8.4
  phpv pecl install /tmp/redis-6.0.2.tgz 8.4`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			var versionFlag string
			if cmd.Flags().Changed("version") {
				versionFlag, _ = cmd.Flags().GetString("version")
			}
			phpVersion, err := h.resolvePeclVersion(cmd, args, versionFlag)
			if err != nil {
				return err
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Install PECL extension %q for PHP %s? [y/N] ", source, phpVersion)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			jobsFlag, _ := cmd.Flags().GetInt("jobs")
			jobs := resolveJobs(jobsFlag, h.configSvc)

			result, err := h.peclSvc.Install(h.ctx, source, phpVersion, jobs)
			if err != nil {
				return fmt.Errorf("pecl install: %w", err)
			}
			fmt.Printf("✓ Installed %s %s\n", result.Name, result.Version)
			fmt.Printf("  Location: %s\n", result.InstallDir)
			return nil
		},
	}
	installCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	installCmd.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	installCmd.Flags().String("version", "", "PHP version to install into (defaults to the active version)")

	listCmd := &cobra.Command{
		Use:   "list [version]",
		Short: "List installed PECL extensions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			phpVersion, err := h.resolveVersion("")
			if len(args) == 1 {
				phpVersion, err = h.resolveVersion(args[0])
			}
			if err != nil {
				return err
			}

			exts, err := h.peclSvc.List(phpVersion)
			if err != nil {
				return fmt.Errorf("pecl list: %w", err)
			}

			if jsonFlag {
				type peclListEntry struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}
				type peclListResponse struct {
					PHPVersion string          `json:"php_version"`
					Extensions []peclListEntry `json:"extensions"`
				}
				var entries []peclListEntry
				for _, e := range exts {
					entries = append(entries, peclListEntry{Name: e.Name, Version: e.Version})
				}
				return printJSON(jsonResponse{SchemaVersion: 1, Data: peclListResponse{
					PHPVersion: phpVersion,
					Extensions: entries,
				}})
			}

			if len(exts) == 0 {
				fmt.Printf("No PECL extensions installed for PHP %s\n", phpVersion)
				return nil
			}
			fmt.Printf("PECL extensions for PHP %s:\n", phpVersion)
			for _, e := range exts {
				v := e.Version
				if v == "" {
					v = "?"
				}
				fmt.Printf("  - %s (%s)\n", e.Name, v)
			}
			return nil
		},
	}
	listCmd.Flags().Bool("json", false, "Output in JSON format")

	uninstallCmd := &cobra.Command{
		Use:   "uninstall <name> [version]",
		Short: "Uninstall a PECL extension",
		Long: `Uninstall a PECL extension.

The PHP version defaults to the active version; use --version to target a
specific one. The positional [version] argument is kept for backwards
compatibility.

Examples:
  phpv pecl uninstall redis
  phpv pecl uninstall redis --version 8.4`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var versionFlag string
			if cmd.Flags().Changed("version") {
				versionFlag, _ = cmd.Flags().GetString("version")
			}
			phpVersion, err := h.resolvePeclVersion(cmd, args, versionFlag)
			if err != nil {
				return err
			}

			if err := h.peclSvc.Uninstall(name, phpVersion); err != nil {
				return fmt.Errorf("pecl uninstall: %w", err)
			}
			fmt.Printf("✓ Uninstalled %s from PHP %s\n", name, phpVersion)
			return nil
		},
	}
	uninstallCmd.Flags().String("version", "", "PHP version to uninstall from (defaults to the active version)")

	cmd.AddCommand(installCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(uninstallCmd)
	return cmd
}

// resolvePeclVersion resolves the PHP version for a PECL install/uninstall.
// Precedence: explicit --version flag, then the legacy positional [version]
// argument, then the active version. Providing both --version and a positional
// version is an error to avoid ambiguity.
func (h *PHPHandler) resolvePeclVersion(cmd *cobra.Command, args []string, versionFlag string) (string, error) {
	if cmd.Flags().Changed("version") && len(args) == 2 {
		return "", fmt.Errorf("specify the PHP version either with --version or as a positional argument, not both")
	}
	constraint := ""
	if cmd.Flags().Changed("version") {
		constraint = versionFlag
	} else if len(args) == 2 {
		constraint = args[1]
	}
	return h.resolveVersion(constraint)
}
