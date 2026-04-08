package terminal

func bashInitCode(phpvRoot string) string {
	return `export PHPV_ROOT="` + phpvRoot + `"
export PATH="$PHPV_ROOT/bin:$PATH"

phpv() {
    local cmd="$1"
    shift
    case "$cmd" in
        use|install|default|versions|list|which|uninstall|doctor|upgrade)
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
`
}

func zshInitCode(phpvRoot string) string {
	return `export PHPV_ROOT="` + phpvRoot + `"
export PATH="$PHPV_ROOT/bin:$PATH"

phpv() {
    local cmd="$1"
    shift
    case "$cmd" in
        use|install|default|versions|list|which|uninstall|doctor|upgrade)
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
`
}

func fishInitCode(phpvRoot string) string {
	return `set -gx PHPV_ROOT "` + phpvRoot + `"
set -gx PATH "$PHPV_ROOT/bin" $PATH

function phpv
    set -l cmd "$argv[1]"
    set -e argv[1]
    switch "$cmd"
        case use|install|default|versions|list|which|uninstall|doctor|upgrade
            command phpv "$cmd" $argv
        case shell-use
            set -gx PHPV_CURRENT "$argv[1]"
            command phpv write-default "$argv[1]"
        case '*'
            command phpv "$cmd" $argv
    end
end
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
