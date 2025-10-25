#!/usr/bin/env bash

# Install libxml2 from source
install_libxml2_from_source() {
    local php_version="${1:-}"
    local version="2.11.5"

    # Use older libxml2 for PHP 5.x compatibility
    if [[ -n "$php_version" && "$php_version" == 5.* ]]; then
        version="2.9.14"
    fi

    local series="${version%.*}"
    local url="https://download.gnome.org/sources/libxml2/$series/libxml2-$version.tar.xz"
    local cache_file="$PHPV_CACHE_DIR/libxml2-$version.tar.xz"
    local build_dir="$PHPV_CACHE_DIR/libxml2-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xf "$cache_file" --strip-components=1
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        ./configure --prefix="$PHPV_DEPS_DIR" --without-python
        make -j$(nproc)
        make -j$(nproc) install
    else
        run_with_progress "Configuring libxml2" 30 ./configure --prefix="$PHPV_DEPS_DIR" --without-python || return 1
        run_with_progress "Building libxml2" 50 make -j$(nproc) || return 1
        run_with_progress "Installing libxml2" 20 make -j$(nproc) install || return 1
    fi
}