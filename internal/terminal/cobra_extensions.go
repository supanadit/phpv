package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerExtensionCommands(root *cobra.Command, handler *TerminalHandler) {
	extensionsCmd := &cobra.Command{
		Use:   "extensions",
		Short: "List available PHP extensions",
		Long: `List all PHP extensions that can be enabled during installation.

Examples:
  phpv extensions              # List all extensions
  phpv extensions --php 8.4    # List extensions compatible with PHP 8.4`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			phpVersion, _ := cmd.Flags().GetString("php")

			result, err := handler.ListExtensions(phpVersion)
			if err != nil {
				return fmt.Errorf("failed to list extensions: %w", err)
			}

			printer := &ExtensionsPrinter{
				Extensions: result.Extensions,
				PHPVersion: result.PHPVersion,
			}
			printer.Print()
			return nil
		},
	}

	extensionsCmd.Flags().String("php", "", "Filter extensions compatible with a specific PHP version (e.g., 8.4)")

	root.AddCommand(extensionsCmd)

	validateExtCmd := &cobra.Command{
		Use:   "validate-extensions <ext1,ext2,...>",
		Short: "Validate extensions for a PHP version",
		Long: `Check if extensions are valid and compatible with a PHP version.

Examples:
  phpv validate-extensions curl,mbstring --php 8.4`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			phpVersion, _ := cmd.Flags().GetString("php")
			if phpVersion == "" {
				return fmt.Errorf("--php flag is required")
			}

			extensions := parseExtensions(args[0])
			result, err := handler.ValidateExtensions(extensions, phpVersion)
			if err != nil {
				return fmt.Errorf("failed to validate extensions: %w", err)
			}

			if !result.HasErrors() {
				fmt.Printf("All extensions are valid for PHP %s\n", phpVersion)
				return nil
			}

			return fmt.Errorf("%s", result.ErrorMessage())
		},
	}

	validateExtCmd.Flags().String("php", "", "PHP version to validate against (required)")

	root.AddCommand(validateExtCmd)
}
