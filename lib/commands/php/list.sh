#!/usr/bin/env bash

# List all versions
list_versions() {
    local current_version
    current_version=$(get_current_version)
    
    echo "Installed versions:"
    while IFS= read -r version; do
        if [[ "$version" == "$current_version" ]]; then
            echo -e "  ${GREEN}* $version${NC}"
        else
            echo "    $version"
        fi
    done < <(get_installed_versions)
}

# List available versions for download
list_available() {
    local filter="${1:-}"
    echo "Available versions for download:"
    if [[ -z "$filter" ]]; then
        get_available_versions | sed 's/^/  /'
    else
        # Add dot to filter if it doesn't end with one
        local filter_pattern="$filter"
        if [[ "$filter_pattern" != *"." ]]; then
            filter_pattern="$filter_pattern."
        fi
        get_available_versions | grep "^$filter_pattern" | sed 's/^/  /'
    fi
}