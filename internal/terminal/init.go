package terminal

import (
	"strings"
	"text/template"
)

// ShellInitCodeTemplate contains the base template for shell initialization
const ShellInitCodeTemplate = `
export PHPV_ROOT="{{ .Root }}"
export PATH="$PHPV_ROOT/bin:$PATH"

{{ .ShellFunc }}

{{ .AutoDetectFunc }}

{{ .HookCode }}
`

// ShellTemplateData contains the data for the shell initialization template
type ShellTemplateData struct {
	Root           string
	ShellFunc      string
	AutoDetectFunc string
	HookCode       string
}

// GetInitCodeForShell returns the shell initialization code for the specified shell
func GetInitCodeForShell(shell string, phpvRoot string) string {
	tmpl := template.Must(template.New("shell").Parse(ShellInitCodeTemplate))

	var data ShellTemplateData
	data.Root = phpvRoot

	switch shell {
	case "fish":
		data.ShellFunc = fishShellFunc()
		data.AutoDetectFunc = fishAutoDetectFunc()
		data.HookCode = fishHookCode()
	case "zsh":
		data.ShellFunc = bashZshShellFunc()
		data.AutoDetectFunc = bashZshAutoDetectFunc()
		data.HookCode = zshHookCode()
	case "bash":
		fallthrough
	default:
		data.ShellFunc = bashZshShellFunc()
		data.AutoDetectFunc = bashZshAutoDetectFunc()
		data.HookCode = bashHookCode()
	}

	var buf strings.Builder
	tmpl.Execute(&buf, data)
	return strings.TrimSpace(buf.String())
}

func bashZshShellFunc() string {
	return `
phpv() {
    local cmd="$1"
    [ $# -gt 0 ] && shift
    case "$cmd" in
        use)
            if [ -z "$1" ]; then
                echo "Error: version required" >&2
                return 1
            fi
            command phpv "$cmd" "$@" >/dev/null 2>&1
            resolved=$(phpv auto-detect-resolve "$1" 2>/dev/null)
            if [ -n "$resolved" ]; then
                export PHPV_CURRENT="$resolved"
            fi
            ;;
        install|default|versions|list|which|uninstall|doctor|upgrade|auto-detect)
            command phpv "$cmd" "$@"
            ;;
        shell-use)
            local ver="$1"
            if [ -z "$ver" ]; then
                echo "Error: version required" >&2
                return 1
            fi
            export PHPV_CURRENT="$ver"
            ;;
        *)
            command phpv "$cmd" "$@"
            ;;
    esac
}`
}

func fishShellFunc() string {
	return `
function phpv
    set -l cmd "$argv[1]"
    set -e argv[1]
    switch "$cmd"
        case use
            if test (count $argv) -eq 0
                echo "Error: version required" >&2
                return 1
            end
            command phpv "$cmd" $argv >/dev/null 2>&1
            set -l resolved (phpv auto-detect-resolve "$argv[1]" 2>/dev/null)
            if test -n "$resolved"
                set -gx PHPV_CURRENT "$resolved"
            end
        case install|default|versions|list|which|uninstall|doctor|upgrade|auto-detect
            command phpv "$cmd" $argv
        case shell-use
            set -gx PHPV_CURRENT "$argv[1]"
        case '*'
            command phpv "$cmd" $argv
    end
end`
}

func bashZshAutoDetectFunc() string {
	return `
_phpv_auto_switch() {
    if [ -f composer.json ] && command -v phpv >/dev/null 2>&1; then
        local phpver
        phpver=$(phpv auto-detect-resolve 2>/dev/null)
        if [ -n "$phpver" ]; then
            local current="${PHPV_CURRENT:-$(cat "$PHPV_ROOT/default" 2>/dev/null)}"
            if [ "$current" != "$phpver" ]; then
                export PHPV_CURRENT="$phpver"
            fi
        fi
    fi
}`
}

func fishAutoDetectFunc() string {
	return `
function _phpv_auto_switch
    if test -f composer.json; and type -q phpv
        set phpver (phpv auto-detect-resolve 2>/dev/null)
        if test -n "$phpver"
            set current "$PHPV_CURRENT"
            if test -z "$current"
                set current (cat "$PHPV_ROOT/default" 2>/dev/null)
            end
            if test "$current" != "$phpver"
                set -gx PHPV_CURRENT "$phpver"
            end
        end
    end
end`
}

func bashHookCode() string {
	return `
PROMPT_COMMAND="_phpv_auto_switch;$PROMPT_COMMAND"`
}

func zshHookCode() string {
	return `
autoload -Uz add-zsh-hook
add-zsh-hook precmd _phpv_auto_switch`
}

func fishHookCode() string {
	return `
functions -c fish_prompt _phpv_auto_switch`
}
