#!/usr/bin/env bash

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