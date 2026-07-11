package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
)

// PHPHandler represents the terminal handler for PHP commands.
// It orchestrates the download workflow by delegating to the assembler
// service, which resolves transitive dependencies and downloads all
// packages in parallel.
type PHPHandler struct {
	assemblerService *assembler.Service
}

// NewPHPHandler registers all PHP subcommands onto the given root command.
// This mirrors NewArticleHandler in the REST delivery layer which
// registers routes onto the Echo instance.
func NewPHPHandler(rootCmd *cobra.Command, svc *assembler.Service) {
	handler := &PHPHandler{
		assemblerService: svc,
	}
	rootCmd.AddCommand(handler.downloadCmd())
}

// downloadCmd creates the cobra command for `phpv download <version>`.
func (h *PHPHandler) downloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <version>",
		Short: "Download a PHP version",
		Long:  "Download a specific version of PHP and all its transitive dependencies to the local cache.",
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

	fmt.Printf("Downloading %s %s and its dependencies...\n", name, version)

	results, err := h.assemblerService.Download(name, version)
	if err != nil {
		return err
	}

	var downloaded, skipped int
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  ✗ Failed %s@%s: %v\n", r.Name, r.Version, r.Err)
			continue
		}
		if r.Downloaded {
			fmt.Printf("  ✓ Downloaded %s@%s\n", r.Name, r.Version)
			downloaded++
		} else {
			fmt.Printf("  → Skipped %s@%s (already exists)\n", r.Name, r.Version)
			skipped++
		}
	}

	fmt.Printf("Done: %d downloaded, %d skipped\n", downloaded, skipped)
	return nil
}