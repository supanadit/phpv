package terminal

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [shell]",
		Short: "Generate shell integration script",
		Long: `Generate shell integration for PHP version switching.
Supports: bash, zsh, fish, pwsh, ksh.
If no shell is specified, auto-detects from $SHELL.`,
		Args: cobra.MaximumNArgs(1),
		RunE: h.initShell,
	}
}

func (h *PHPHandler) initShell(cmd *cobra.Command, args []string) error {
	shell := ""
	if len(args) == 1 {
		shell = args[0]
	} else {
		shell = detectShell()
	}

	if err := h.shimSvc.RegenerateAll(); err != nil {
		return fmt.Errorf("regenerate shims: %w", err)
	}

	bin := h.siloSvc.GetSilo().Root + "/bin"

	switch shell {
	case "bash", "zsh", "ksh":
		fmt.Printf("export PATH=\"%s:$PATH\"\n", bin)
	case "fish":
		fmt.Printf("fish_add_path %s\n", bin)
	case "pwsh":
		fmt.Printf("$env:PATH = \"%s;$env:PATH\"\n", bin)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, pwsh, ksh)", shell)
	}
	return nil
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}
	parts := strings.Split(shell, "/")
	name := parts[len(parts)-1]
	switch name {
	case "bash", "zsh", "fish", "pwsh", "ksh":
		return name
	default:
		return "bash"
	}
}
