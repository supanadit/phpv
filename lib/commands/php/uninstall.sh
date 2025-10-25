#!/usr/bin/env bash

# Uninstall a PHP version
uninstall_php_version() {
    local input_version="$1"
    
    if [[ -z "$input_version" ]]; then
        log_error "Please specify a version to uninstall"
        return 1
    fi
    
    if [[ "$input_version" == "system" ]]; then
        log_error "Cannot uninstall system PHP"
        return 1
    fi
    
    # Resolve the actual version to uninstall (e.g., "7.4" -> "7.4.33")
    local version
    version=$(resolve_installed_version "$input_version")
    
    if [[ -z "$version" ]]; then
        log_error "PHP $input_version is not installed"
        return 1
    fi
    
    # If we resolved to a different version, inform the user
    if [[ "$version" != "$input_version" ]]; then
        log_info "Uninstalling $version (matched from '$input_version')"
    fi
    
    local current_version
    current_version=$(get_current_version)
    
    if [[ "$version" == "$current_version" ]]; then
        log_warning "Currently using PHP $version, switching to system"
        use_php_version "system"
    fi
    
    log_info "Uninstalling PHP $version..."
    local llvm_version_file="$PHPV_VERSIONS_DIR/$version/.llvm_version"
    local llvm_version=""
    if [[ -f "$llvm_version_file" ]]; then
        llvm_version=$(cat "$llvm_version_file" 2>/dev/null || true)
    fi

    rm -rf "$PHPV_VERSIONS_DIR/$version"

    if [[ -n "$llvm_version" ]]; then
        local version_deps_dir
        if version_deps_dir=$(get_deps_dir_for_version "$version" "$llvm_version"); then
            if [[ -d "$version_deps_dir" ]]; then
                log_info "Removing isolated dependencies for PHP $version..."
                rm -rf "$version_deps_dir"
            fi
        fi
    fi
    log_success "PHP $version uninstalled"
}