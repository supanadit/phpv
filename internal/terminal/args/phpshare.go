package args

import (
	"fmt"

	"github.com/spf13/cobra"
)

// shareCmd exports an installed PHP version as a portable bundle.
func (h *PHPHandler) shareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "share <version>",
		Short: "Export installed PHP as a portable bundle",
		Long:  "Export an installed PHP version as a portable .tar.gz bundle that can be shared with others.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.share,
	}
	cmd.Flags().StringP("output", "o", "", "Output path for the bundle file")
	return cmd
}

func (h *PHPHandler) share(cmd *cobra.Command, args []string) error {
	version := args[0]
	output, _ := cmd.Flags().GetString("output")

	fmt.Printf("Exporting PHP %s...\n", version)
	if err := h.bundleSvc.Export(version, output); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}
	if output == "" {
		output = fmt.Sprintf("php-%s-%s-%s.tar.gz", version, "linux", "amd64")
	}
	fmt.Printf("✓ PHP %s exported to %s\n", version, output)
	return nil
}
