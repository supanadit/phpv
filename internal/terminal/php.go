package terminal

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/silo"
)

// PHPHandler registers cobra commands and delegates to services.
type PHPHandler struct {
	siloSvc     *silo.Service
	assemblerSvc *assembler.Service
}

// NewPHPHandler registers all PHP subcommands onto the given root command.
func NewPHPHandler(rootCmd *cobra.Command, siloSvc *silo.Service, assemblerSvc *assembler.Service) {
	h := &PHPHandler{
		siloSvc:     siloSvc,
		assemblerSvc: assemblerSvc,
	}
	rootCmd.AddCommand(h.downloadCmd())
	rootCmd.AddCommand(h.installCmd())
}

func (h *PHPHandler) downloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download <version>",
		Short: "Download a PHP version",
		Long:  "Download a specific version of PHP and all its transitive dependencies to the local cache.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.download,
	}
}

func (h *PHPHandler) download(cmd *cobra.Command, args []string) error {
	version := args[0]
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = "php"
	}

	// TODO: silo.Service needs a BatchDownload or similar to handle transitive deps.
	// For now, just download the single package.
	fmt.Printf("Downloading %s@%s...\n", name, version)
	downloaded, err := h.siloSvc.Download(name, version)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if downloaded {
		fmt.Printf("✓ Downloaded %s@%s\n", name, version)
	} else {
		fmt.Printf("→ Skipped %s@%s (already exists)\n", name, version)
	}
	return nil
}

func (h *PHPHandler) installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <version>",
		Short: "Install a PHP version",
		Long:  "Download, build, and install a specific version of PHP with all its dependencies.",
		Args:  cobra.ExactArgs(1),
		RunE:  h.install,
	}
}

func (h *PHPHandler) install(cmd *cobra.Command, args []string) error {
	version := args[0]

	fmt.Printf("Installing PHP %s...\n", version)

	result, err := h.assemblerSvc.Assemble("php", version)
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	var downloaded, skipped, extracted int
	for _, r := range result.DownloadResults {
		if r.Err != nil {
			fmt.Printf("  ✗ Failed %s@%s: %v\n", r.Name, r.Version, r.Err)
			continue
		}
		if r.Downloaded {
			downloaded++
		} else {
			skipped++
		}
		if r.Extracted {
			extracted++
		}
	}
	fmt.Printf("Downloaded: %d, Skipped: %d, Extracted: %d\n", downloaded, skipped, extracted)
	fmt.Printf("✓ PHP %s installed at %s\n", result.PHPVersion, result.Prefix)
	return nil
}
