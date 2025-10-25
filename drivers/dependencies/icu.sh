#!/usr/bin/env bash

# Install ICU from source
install_icu_from_source() {
    local php_version="$1"
    local version
    local url
    
    # Use older ICU version for PHP 5.x compatibility
    if [[ "$php_version" == 5.* ]]; then
        version="4.8.1"
        url="https://github.com/unicode-org/icu/releases/download/icu4c-4_8_1-src.tgz/icu4c-4_8_1-src.tgz"
    else
        version="73.2"
        url="https://github.com/unicode-org/icu/releases/download/release-$(echo $version | tr . -)/icu4c-$(echo $version | tr . _)-src.tgz"
    fi
    
    local cache_file="$PHPV_CACHE_DIR/icu4c-$(echo $version | tr . _)-src.tgz"
    local build_dir="$PHPV_CACHE_DIR/icu-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    tar -xzf "$cache_file" -C "$build_dir"
    cd "$build_dir/icu/source"
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        ./configure --prefix="$PHPV_DEPS_DIR"
        make -j$(nproc)
        make -j$(nproc) install
    else
        run_with_progress "Configuring ICU" 30 ./configure --prefix="$PHPV_DEPS_DIR" || return 1
        run_with_progress "Building ICU" 50 make -j$(nproc) || return 1
        run_with_progress "Installing ICU" 20 make -j$(nproc) install || return 1
    fi
}
