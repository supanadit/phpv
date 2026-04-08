package terminal

func bashInitCode(phpvRoot string) string {
	return `export PHPV_ROOT="` + phpvRoot + `"
export PATH="$PHPV_ROOT/bin:$PATH"

phpv() {
    local cmd="$1"
    shift
    case "$cmd" in
        use|install|default|versions|list|which|uninstall|doctor|upgrade|auto-detect)
            command phpv "$cmd" "$@"
            ;;
        shell-use)
            local ver="$1"
            if [ -z "$ver" ]; then
                echo "Error: version required" >&2
                return 1
            fi
            export PHPV_CURRENT="$ver"
            phpv write-default "$ver"
            ;;
        *)
            command phpv "$cmd" "$@"
            ;;
    esac
}

_phpv_auto_switch() {
    if [ -f composer.json ] && command -v phpv >/dev/null 2>&1; then
        local phpver
        phpver=$(phpv auto-detect-resolve 2>/dev/null)
        if [ -n "$phpver" ]; then
            local current="${PHPV_CURRENT:-$(cat "$PHPV_ROOT/default" 2>/dev/null)}"
            if [ "$current" != "$phpver" ]; then
                export PHPV_CURRENT="$phpver"
                phpv write-default "$phpver" 2>/dev/null
            fi
        fi
    fi
}

PROMPT_COMMAND="_phpv_auto_switch;$PROMPT_COMMAND"
`
}

func zshInitCode(phpvRoot string) string {
	return `export PHPV_ROOT="` + phpvRoot + `"
export PATH="$PHPV_ROOT/bin:$PATH"

phpv() {
    local cmd="$1"
    shift
    case "$cmd" in
        use|install|default|versions|list|which|uninstall|doctor|upgrade|auto-detect)
            command phpv "$cmd" "$@"
            ;;
        shell-use)
            local ver="$1"
            if [ -z "$ver" ]; then
                echo "Error: version required" >&2
                return 1
            fi
            export PHPV_CURRENT="$ver"
            phpv write-default "$ver"
            ;;
        *)
            command phpv "$cmd" "$@"
            ;;
    esac
}

_phpv_auto_switch() {
    if [ -f composer.json ] && command -v phpv >/dev/null 2>&1; then
        local phpver
        phpver=$(phpv auto-detect-resolve 2>/dev/null)
        if [ -n "$phpver" ]; then
            local current="${PHPV_CURRENT:-$(cat "$PHPV_ROOT/default" 2>/dev/null)}"
            if [ "$current" != "$phpver" ]; then
                export PHPV_CURRENT="$phpver"
                phpv write-default "$phpver" 2>/dev/null
            fi
        fi
    fi
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd _phpv_auto_switch
`
}

func fishInitCode(phpvRoot string) string {
	return `set -gx PHPV_ROOT "` + phpvRoot + `"
set -gx PATH "$PHPV_ROOT/bin" $PATH

function phpv
    set -l cmd "$argv[1]"
    set -e argv[1]
    switch "$cmd"
        case use|install|default|versions|list|which|uninstall|doctor|upgrade|auto-detect
            command phpv "$cmd" $argv
        case shell-use
            set -gx PHPV_CURRENT "$argv[1]"
            command phpv write-default "$argv[1]"
        case '*'
            command phpv "$cmd" $argv
    end
end

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
                command phpv write-default "$phpver" 2>/dev/null
            end
        end
    end
end

functions -c fish_prompt _phpv_auto_switch
`
}

func GetInitCodeForShell(shell string, phpvRoot string) string {
	switch shell {
	case "fish":
		return fishInitCode(phpvRoot)
	case "zsh":
		return zshInitCode(phpvRoot)
	case "bash":
		return bashInitCode(phpvRoot)
	default:
		return bashInitCode(phpvRoot)
	}
}
