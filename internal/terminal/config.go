package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage phpv configuration",
		Long:  "View and modify phpv configuration stored in ~/.phpv/config.toml.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val, err := h.configSvc.Get(args[0])
			if err != nil {
				return err
			}
			if val == "" {
				fmt.Printf("%s: (unset)\n", args[0])
				return nil
			}
			fmt.Println(val)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := h.configSvc.Set(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("✓ %s set to %s\n", args[0], args[1])
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all config values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := h.configSvc.List()
			if err != nil {
				return err
			}
			fmt.Println("Config:")
			for _, line := range lines {
				fmt.Printf("  %s\n", line)
			}
			return nil
		},
	})

	return cmd
}

// configCompletion provides shell completion for config keys.
func configCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	keys := []string{"cache_dir", "concurrency", "compiler", "mirror", "static_libgcc"}
	return keys, cobra.ShellCompDirectiveNoFileComp
}
