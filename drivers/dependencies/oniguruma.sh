#!/usr/bin/env bash

# Install oniguruma from source
install_oniguruma_from_source() {
    local php_version="$1"
    local version
    
    # Use oniguruma 6.9.9 for PHP 7.x and above, 5.9.6 for PHP 5.x and below
    if [[ -n "$php_version" && "$php_version" =~ ^[7-9] ]]; then
        version="6.9.9"
    else
        version="5.9.6"
    fi
    
    local url="https://github.com/kkos/oniguruma/releases/download/v$version/onig-$version.tar.gz"
    build_from_source "onig" "$version" "$url"
}