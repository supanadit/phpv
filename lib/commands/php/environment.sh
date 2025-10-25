#!/usr/bin/env bash

# Get PHP binary path
get_php_path() {
    local current_version
    current_version=$(get_current_version)
    
    if [[ "$current_version" == "system" ]]; then
        command -v php 2>/dev/null || echo ""
    else
        local php_path="$PHPV_VERSIONS_DIR/$current_version/bin/php"
        if [[ -x "$php_path" ]]; then
            echo "$php_path"
        else
            echo ""
        fi
    fi
}