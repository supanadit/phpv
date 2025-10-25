#!/usr/bin/env bash

# PHPV - Dependency installation functions

# Install libjpeg from source
install_libjpeg_from_source() {
    local version="9e"
    local url="https://www.ijg.org/files/jpegsrc.v$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/jpegsrc.v$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/jpeg-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        ./configure --prefix="$PHPV_DEPS_DIR"
        make -j$(nproc)
        make -j$(nproc) install
    else
        run_with_progress "Configuring libjpeg" 30 ./configure --prefix="$PHPV_DEPS_DIR" || return 1
        run_with_progress "Building libjpeg" 50 make -j$(nproc) || return 1
        run_with_progress "Installing libjpeg" 20 make -j$(nproc) install || return 1
    fi
}