#!/usr/bin/env bash

# Execute PHP with current version
exec_php() {
    local php_path
    php_path=$(get_php_path)
    
    if [[ -z "$php_path" ]]; then
        log_error "PHP is not available"
        return 1
    fi

    local current_version
    current_version=$(get_current_version)
    if [[ -n "$current_version" && "$current_version" != "system" ]]; then
        local lib_joined
        lib_joined=$(get_lib_paths_for_version "$current_version")
        if [[ -n "$lib_joined" ]]; then
            if [[ -n "${LD_LIBRARY_PATH:-}" ]]; then
                export LD_LIBRARY_PATH="$lib_joined:$LD_LIBRARY_PATH"
            else
                export LD_LIBRARY_PATH="$lib_joined"
            fi
        fi
    fi
    
    exec "$php_path" "$@"
}