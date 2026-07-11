package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
)

// PHPService represents the usecase contract for the PHP delivery layer.
// This mirrors the pattern in the REST delivery layer where the handler
// defines its own interface rather than importing the service directly,
// keeping the dependency direction pointing inward.
type PHPService interface {
	Download(name string, version string) (err error)
}

// PHPHandler represents the terminal handler for PHP commands.
// It handles download, and in the future install, uninstall, etc.
type PHPHandler struct {
	Service PHPService
}

// defaultName is the package name used when no --name flag is provided.
// Since phpv is a PHP version manager, "php" is the sensible default.
const defaultName = "php"

// NewPHPHandler registers all PHP subcommands onto the given root command.
// This mirrors NewArticleHandler in the REST delivery layer which
// registers routes onto the Echo instance.
func NewPHPHandler(rootCmd *cobra.Command, svc PHPService) {
	handler := &PHPHandler{
		Service: svc,
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

	cmd.Flags().StringP("name", "n", defaultName, "package name to download")

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

	if err := h.Service.Download(name, version); err != nil {
		return err
	}

	fmt.Printf("Successfully downloaded %s %s\n", name, version)
	return nil
}
