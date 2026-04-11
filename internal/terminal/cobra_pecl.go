package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func registerPeclCommands(root *cobra.Command, handler *TerminalHandler) {
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
	root.AddCommand(peclCmd)
}
