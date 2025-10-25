#!/usr/bin/env bash

# Show current PHP version
show_current_version() {
    local current_version
    current_version=$(get_current_version)
    
    if [[ "$current_version" == "system" ]]; then
        # When checking for system PHP, exclude phpv-managed versions from PATH
        local clean_path
        clean_path=$(echo "$PATH" | tr ':' '\n' | grep -v "$PHPV_VERSIONS_DIR" | tr '\n' ':')
        clean_path=${clean_path%:}
        
        if PATH="$clean_path" command -v php &> /dev/null; then
            local system_version
            system_version=$(PATH="$clean_path" php -v | head -n1 | cut -d' ' -f2)
            echo "Current: system (PHP $system_version)"
        else
            echo "Current: system (PHP not found in PATH)"
        fi
    else
        local php_path="$PHPV_VERSIONS_DIR/$current_version/bin/php"
        if [[ -x "$php_path" ]]; then
            local previous_ld="${LD_LIBRARY_PATH:-}"
            local lib_joined
            lib_joined=$(get_lib_paths_for_version "$current_version")

            if [[ -n "$lib_joined" ]]; then
                if [[ -n "$previous_ld" ]]; then
                    export LD_LIBRARY_PATH="$lib_joined:$previous_ld"
                else
                    export LD_LIBRARY_PATH="$lib_joined"
                fi
            fi

            local version_info
            if version_info=$("$php_path" -v | head -n1 2>/dev/null); then
                echo "Current: $current_version ($version_info)"
            else
                echo "Current: $current_version (failed to query version)"
            fi

            if [[ -n "$previous_ld" ]]; then
                export LD_LIBRARY_PATH="$previous_ld"
            else
                unset LD_LIBRARY_PATH
            fi
        else
            echo "Current: $current_version (invalid installation)"
        fi
    fi
}