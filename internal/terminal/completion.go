package terminal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for phpv.

To use, source the output in your shell profile:

  bash:    phpv completion bash > /etc/bash_completion.d/phpv
  zsh:     phpv completion zsh > ~/.zsh/completions/_phpv
  fish:    phpv completion fish > ~/.config/fish/completions/phpv.fish
  pwsh:    phpv completion powershell >> $PROFILE`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletion(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", args[0])
			}
		},
	}
}
