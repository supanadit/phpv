package terminal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
)

func registerExtraCommands(root *cobra.Command, handler *TerminalHandler) {
	initCmd := &cobra.Command{
		Use:   "init [bash|zsh|fish]",
		Short: "Output shell initialization code",
		Long: `Output shell initialization code for the specified shell. Add this to your shell RC file or eval it:

    eval "$(phpv init)"

After initialization, you can use 'phpv use <version>' to switch PHP versions in the current shell.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := "bash"
			if len(args) > 0 {
				shell = args[0]
			}
			initCode, err := handler.GetInitCode(shell)
			if err != nil {
				return err
			}
			fmt.Print(initCode)
			return nil
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show phpv version",
		Long:  `Show the version of phpv being used.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("phpv %s\n", domain.AppVersion)
			return nil
		},
	}

	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for the specified shell.
		
To load completions:

Bash:

  $ source <(phpv completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ phpv completion bash > /etc/bash_completion.d/phpv
  # macOS:
  $ phpv completion bash > /usr/local/etc/bash_completion.d/phpv

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ phpv completion zsh > "${fpath[1]}/_phpv"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ phpv completion fish | source

  # To load completions for each session, execute once:
  $ phpv completion fish > ~/.config/fish/completions/phpv.fish

PowerShell:

  PS> phpv completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  PS> phpv completion powershell > phpv.ps1
  # and source this file from your PowerShell profile.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	root.AddCommand(initCmd)
	root.AddCommand(versionCmd)
	root.AddCommand(completionCmd)
}
