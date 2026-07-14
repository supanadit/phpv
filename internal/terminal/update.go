package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Self-update phpv to the latest version",
		Long: `Download and install the latest phpv release from GitHub.

Uses the GitHub Releases API to find the latest version, downloads
the binary for the current OS/arch, verifies the checksum, and
replaces the current binary in-place.`,
		Args: cobra.NoArgs,
		RunE: h.update,
	}
	cmd.Flags().Bool("check", false, "Only check if an update is available, don't install")
	return cmd
}

func (h *PHPHandler) update(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")

	if checkOnly {
		latest, hasUpdate, err := h.updateSvc.CheckForUpdate()
		if err != nil {
			return fmt.Errorf("check for update: %w", err)
		}
		if hasUpdate {
			fmt.Printf("Update available: %s (current: %s)\n", latest, h.version)
		} else {
			fmt.Printf("Already up to date (%s)\n", h.version)
		}
		return nil
	}

	return h.updateSvc.SelfUpdate()
}
