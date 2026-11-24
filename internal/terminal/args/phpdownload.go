package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/pkg/compiler"
)

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
	cmd := &cobra.Command{
		Use:   "install <version>",
		Short: "Install a PHP version",
		Long: `Download, build, and install a specific version of PHP with all its dependencies.

Version syntax:
  8.4.4    exact version
  8.4      latest 8.4.x patch
  8        latest 8.x.x minor`,
		Args: cobra.ExactArgs(1),
		RunE: h.install,
	}
	cmd.Flags().String("from", "", "Install from a bundle file instead of building from source")
	cmd.Flags().Bool("static", false, "Build with static linking for cross-distro portability")
	cmd.Flags().String("ext", "", "Comma-separated list of extensions to build (e.g., openssl,curl,pdo_mysql)")
	cmd.Flags().Bool("auto-deps", false, "Install missing system packages without prompting")
	cmd.Flags().Bool("no-system", false, "Skip system package check, always build from source")
	cmd.Flags().Bool("dry-run", false, "Show what would be done without doing it")
	cmd.Flags().Bool("fresh", false, "Delete existing install prefix and rebuild (keeps cached source)")
	cmd.Flags().Bool("clean", false, "Delete everything including cached source and rebuild from scratch")
	cmd.Flags().Bool("verbose", false, "Show full build output instead of spinner")
	cmd.Flags().Int("jobs", 0, "Number of parallel build jobs (default: CPU count)")
	cmd.Flags().Bool("yes", false, "Skip confirmation prompts")
	cmd.Flags().Bool("force", false, "Skip state checks and reinstall")
	cmd.Flags().Bool("minimal", false, "Build with --enable-cli only (no default extensions)")
	cmd.Flags().String("compiler", "", fmt.Sprintf("C compiler to use (%s); defaults to the phpv compiler config key, then the platform default", strings.Join(compiler.Names(), ", ")))
	return cmd
}
