package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerPharCommands(root *cobra.Command, handler *TerminalHandler) {
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
	root.AddCommand(pharCmd)
}
