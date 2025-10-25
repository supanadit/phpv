#!/usr/bin/env bash

# PHPV - Build utilities
build_from_source() {
    local name="$1"
    local version="$2"
    local url="$3"
    local configure_args="${4:-}"

    local cache_file="$PHPV_CACHE_DIR/$name-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/$name-$version"

    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi

    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1

    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        ./configure --prefix="$PHPV_DEPS_DIR" $configure_args
        make -j$(nproc)
        make -j$(nproc) install
    else
        run_with_progress "Configuring $name" 30 ./configure --prefix="$PHPV_DEPS_DIR" $configure_args || return 1
        run_with_progress "Building $name" 50 make -j$(nproc) || return 1
        run_with_progress "Installing $name" 20 make -j$(nproc) install || return 1
    fi
}