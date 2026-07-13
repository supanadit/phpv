package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) rehashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rehash",
		Short: "Regenerate all shims",
		Long:  "Regenerate all shims (php, phpize, php-config, php-cgi, phpdgb) to match the current installation.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := h.shimSvc.RegenerateAll(); err != nil {
				return fmt.Errorf("rehash failed: %w", err)
			}
			fmt.Println("✓ Shims regenerated")
			return nil
		},
	}
}
