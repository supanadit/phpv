package shell

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type ShellType string

const (
	ShellBash ShellType = "bash"
	ShellZsh  ShellType = "zsh"
	ShellFish ShellType = "fish"
	ShellPwsh ShellType = "pwsh"
	ShellKsh  ShellType = "ksh"
)

func (s *Service) Init(shellType string) (string, error) {
	shell := ShellType(shellType)
	if shell == "" {
		shell = detectShell()
	}

	switch shell {
	case ShellBash:
		return s.initBash()
	case ShellZsh:
		return s.initZsh()
	case ShellFish:
		return s.initFish()
	case ShellPwsh:
		return s.initPwsh()
	default:
		return s.initBash()
	}
}

func detectShell() ShellType {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return ShellBash
	}
	shell = filepath.Base(shell)
	switch shell {
	case "bash":
		return ShellBash
	case "zsh":
		return ShellZsh
	case "fish":
		return ShellFish
	case "pwsh", "powershell":
		return ShellPwsh
	case "ksh", "ksh93", "mksh":
		return ShellKsh
	default:
		return ShellBash
	}
}

func (s *Service) initBash() (string, error) {
	root := viper.GetString("PHPV_ROOT")
	binDir := filepath.Join(root, "bin")

	return fmt.Sprintf(`# phpv shell initialization
export PHPV_ROOT="%s"
export PATH="%s:${PATH}"

phpv() {
  local command=${1:-}
  [ "$#" -gt 0 ] && shift
  case "$command" in
    use|shell)
      eval "$(command phpv "sh-$command" "$@")"
      ;;
    *)
      command phpv "$command" "$@"
      ;;
  esac
}

# Load phpv shell functions for bash
_completion_loader phpv 2>/dev/null || true
`, root, binDir), nil
}

func (s *Service) initZsh() (string, error) {
	root := viper.GetString("PHPV_ROOT")
	binDir := filepath.Join(root, "bin")

	return fmt.Sprintf(`# phpv shell initialization
export PHPV_ROOT="%s"
export PATH="%s:${PATH}"

phpv() {
  local command=${1:-}
  [ "$#" -gt 0 ] && shift
  case "$command" in
    use|shell)
      eval "$(command phpv "sh-$command" "$@")"
      ;;
    *)
      command phpv "$command" "$@"
      ;;
  esac
}

# Load phpv completion
autoload -Uz compinit
compinit 2>/dev/null || true
`, root, binDir), nil
}

func (s *Service) initFish() (string, error) {
	root := viper.GetString("PHPV_ROOT")
	binDir := filepath.Join(root, "bin")

	return fmt.Sprintf(`# phpv shell initialization
set -gx PHPV_ROOT "%s"
fish_add_path -g "%s"

function phpv
  set command $argv[1]
  set -e argv[1]

  switch "$command"
    case use shell
      eval (phpv "sh-$command" $argv)
    case '*'
      command phpv $command $argv
  end
end

# Load phpv completions
complete -c phpv -f
`, root, binDir), nil
}

func (s *Service) initPwsh() (string, error) {
	root := viper.GetString("PHPV_ROOT")
	binDir := filepath.Join(root, "bin")

	return fmt.Sprintf(`# phpv shell initialization
$env:PHPV_ROOT = "%s"
$env:PATH = "%s;$env:PATH"

function phpv {
  $command = $args[0]
  $rest = $args[1..($args.Count - 1)]

  switch ($command) {
    "use" { 
      Invoke-Expression (phpv "sh-use" $rest)
    }
    "shell" { 
      Invoke-Expression (phpv "sh-shell" $rest)
    }
    default {
      & phpv $command $rest
    }
  }
}

# Register completion
Register-ArgumentCompleter -CommandName phpv -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  & phpv --complete $wordToComplete 2>/dev/null | ForEach-Object {
    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
  }
}
`, root, binDir), nil
}

func (s *Service) InitPath() string {
	root := viper.GetString("PHPV_ROOT")
	return filepath.Join(root, "bin")
}
