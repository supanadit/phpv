package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerInstallCommands(root *cobra.Command, handler *TerminalHandler) {
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

	root.AddCommand(installCmd)
	root.AddCommand(rebuildCmd)
}
