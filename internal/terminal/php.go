package terminal

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/silo"
)

// PHPHandler registers cobra commands and delegates to services.
type PHPHandler struct {
	siloSvc      *silo.Service
	assemblerSvc *assembler.Service
}

// NewPHPHandler registers all PHP subcommands onto the given root command.
func NewPHPHandler(rootCmd *cobra.Command, siloSvc *silo.Service, assemblerSvc *assembler.Service) {
	h := &PHPHandler{
		siloSvc:      siloSvc,
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

	fmt.Printf("Installing PHP %s...\n\n", version)

	progressCh := make(chan progressMsg, 64)
	doneCh := make(chan struct{})

	// Spinner goroutine reads progress messages and animates a spinner.
	go func() {
		defer close(doneCh)
		var current string
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for {
			select {
			case msg, ok := <-progressCh:
				if !ok {
					if current != "" {
						fmt.Fprintf(os.Stdout, "\r\033[2K%s\n", current)
					}
					return
				}
				current = fmt.Sprintf("%s %s", stageGlyph(msg.stage), msg.message)
				fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
			case <-ticker.C:
				if current != "" {
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
				}
				frame++
			}
		}
	}()

	// Assemble runs synchronously; progress is sent via the callback into progressCh.
	result, err := h.assemblerSvc.Assemble("php", version, func(stage, message string) {
		progressCh <- progressMsg{stage: stage, message: message}
	})
	close(progressCh)
	<-doneCh

	if err != nil {
		fmt.Println()
		return fmt.Errorf("install failed: %w", err)
	}
	fmt.Println()
	fmt.Printf("✓ PHP %s installed at %s\n", result.PHPVersion, result.Prefix)
	return nil
}

// progressMsg is sent by the assembler through a progress callback.
type progressMsg struct {
	stage   string
	message string
}

// spinnerFrames are the animation frames for the spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// stageGlyph returns a short glyph for the given stage.
func stageGlyph(stage string) string {
	switch stage {
	case "resolve":
		return "→"
	case "deps":
		return "→"
	case "download":
		return "↓"
	case "build":
		return "⚙"
	case "configure":
		return "⚙"
	case "make":
		return "⚙"
	case "install":
		return "↑"
	case "skip":
		return "↷"
	case "patch":
		return "✎"
	case "error":
		return "✗"
	case "done":
		return "✓"
	default:
		return "·"
	}
}
