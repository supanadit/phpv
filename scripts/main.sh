#!/usr/bin/env bash

# PHPV - Main dispatcher and help

# Show help
show_help() {
    cat << 'EOF'
PHPV - PHP Version Manager

USAGE:
    phpv [--verbose] <command> [arguments]

OPTIONS:
    --verbose, -v               Show detailed build output (default: show progress bars)

COMMANDS:
    install <version>           Install a specific PHP version (supports partial versions: e.g., 8, 8.3)
    uninstall <version>         Uninstall a specific PHP version (supports partial versions: e.g., 8, 8.3)
    use <version>               Switch to a specific PHP version (supports partial versions: e.g., 8, 8.3)
    current                     Show the current PHP version
    list                        List installed PHP versions
    list-available [filter]     List available PHP versions for download (optional filter: e.g., 8, 8.3)
    exec <command>              Execute command with current PHP version
    which                       Show path to current PHP binary
    env                         Print environment overrides for current version
    help                        Show this help message

EXAMPLES:
    phpv install 8.3.12         # Install PHP 8.3.12 with progress bars
    phpv --verbose install 8.3  # Install latest 8.3.x with verbose output
    phpv -v install 8           # Install latest 8.x.x with verbose output
    phpv use 8.3.12             # Switch to PHP 8.3.12
    phpv use 8.3                # Switch to latest installed 8.3.x
    phpv use system             # Switch to system PHP
    phpv current                # Show current version
    phpv list                   # List installed versions
    phpv list-available         # List all available versions
    phpv list-available 8       # List only 8.x versions
    phpv list-available 8.3     # List only 8.3.x versions
    phpv exec -v                # Run 'php -v' with current version
    phpv which                  # Show current PHP binary path

ENVIRONMENT VARIABLES:
    PHPV_ROOT    Root directory for phpv (default: ~/.phpv)
EOF
}

# Main command dispatcher
main() {
    init_phpv
    
    # Parse verbose flag
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                PHPV_VERBOSE=1
                shift
                ;;
            *)
                break
                ;;
        esac
    done
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        "install")
            install_php_version "$1"
            ;;
        "uninstall")
            uninstall_php_version "$1"
            ;;
        "use")
            use_php_version "$1"
            ;;
        "current")
            show_current_version
            ;;
        "list")
            list_versions
            ;;
        "list-available")
            list_available "$@"
            ;;
        "exec")
            exec_php "$@"
            ;;
        "which")
            get_php_path
            ;;
        "env")
            print_environment_overrides
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            echo
            show_help
            exit 1
            ;;
    esac
}