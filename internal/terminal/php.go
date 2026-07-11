package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/silo"
)

// PHPHandler represents the terminal handler for PHP commands.
// It handles download, and in the future install, uninstall, etc.
type PHPHandler struct {
	siloService *silo.Service
}

// NewPHPHandler registers all PHP subcommands onto the given root command.
// This mirrors NewArticleHandler in the REST delivery layer which
// registers routes onto the Echo instance.
func NewPHPHandler(rootCmd *cobra.Command, svc *silo.Service) {
	handler := &PHPHandler{
		siloService: svc,
	}
	rootCmd.AddCommand(handler.downloadCmd())
}

// downloadCmd creates the cobra command for `phpv download <version>`.
func (h *PHPHandler) downloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <version>",
		Short: "Download a PHP version",
		Long:  "Download a specific version of PHP to the local cache.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.download,
	}

	cmd.Flags().StringP("name", "n", "php", "package name to download")

	return cmd
}

// download is the Run handler for the download command.
func (h *PHPHandler) download(cmd *cobra.Command, args []string) error {
	version := args[0]

	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	fmt.Printf("Downloading %s %s...\n", name, version)

	if err := h.siloService.Download(name, version); err != nil {
		return err
	}

	fmt.Printf("Successfully downloaded %s %s\n", name, version)
	return nil
}
