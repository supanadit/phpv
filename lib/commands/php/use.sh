#!/usr/bin/env bash

# Switch to a specific PHP version
use_php_version() {
    local input_version="$1"
    
    if [[ -z "$input_version" ]]; then
        log_error "Please specify a version"
        return 1
    fi
    
    # Resolve the actual version to use (e.g., "7.4" -> "7.4.33")
    local version
    version=$(resolve_installed_version "$input_version")
    
    if [[ -z "$version" ]]; then
        log_error "PHP $input_version is not installed"
        log_info "Available versions:"
        get_installed_versions | sed 's/^/  /'
        return 1
    fi
    
    # If we resolved to a different version, inform the user
    if [[ "$version" != "$input_version" ]]; then
        log_info "Using $version (matched from '$input_version')"
    fi
    
    set_current_version "$version"
    log_success "Now using PHP $version"
    
    # Show current PHP version
    show_current_version
}