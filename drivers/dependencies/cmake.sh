#!/usr/bin/env bash

install_cmake_from_source() {
    local version="3.27.9"
    local cmake_bin="$PHPV_DEPS_DIR/bin/cmake"
    local cache_file="$PHPV_CACHE_DIR/cmake-$version.tar.gz"
    local source_dir="$PHPV_CACHE_DIR/cmake-$version-src"
    local jobs
    jobs=$(nproc)
    local old_cwd
    old_cwd=$(pwd)

    local install_needed=1
    if [[ -x "$cmake_bin" ]]; then
        local existing_version
        existing_version=$("$cmake_bin" --version 2>/dev/null | head -n1 | awk '{print $3}')
        if [[ "$existing_version" == "$version" ]]; then
            install_needed=0
        else
            log_info "Updating bundled CMake from ${existing_version:-unknown} to $version..."
        fi
    fi

    if [[ $install_needed -eq 0 ]]; then
        return 0
    fi

    log_info "Installing CMake $version from source..."

    if [[ ! -f "$cache_file" ]]; then
        local url="https://github.com/Kitware/CMake/releases/download/v$version/cmake-$version.tar.gz"
        safe_download "$url" "$cache_file" || return 1
    fi

    rm -rf "$source_dir"
    mkdir -p "$source_dir"
    if ! tar -xzf "$cache_file" -C "$source_dir" --strip-components=1; then
        cd "$old_cwd"
        return 1
    fi

    cd "$source_dir" || {
        cd "$old_cwd"
        return 1
    }

    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        if ! ./bootstrap --prefix="$PHPV_DEPS_DIR" --parallel="$jobs" -- -DCMake_ENABLE_DEBUGGER=OFF -DBUILD_TESTING=OFF; then
            cd "$old_cwd"
            return 1
        fi

        if ! make -j"$jobs"; then
            cd "$old_cwd"
            return 1
        fi

        if ! make -j$(nproc) install; then
            cd "$old_cwd"
            return 1
        fi
    else
        if ! run_with_progress "Bootstrapping CMake" 30 ./bootstrap --prefix="$PHPV_DEPS_DIR" --parallel="$jobs" -- -DCMake_ENABLE_DEBUGGER=OFF -DBUILD_TESTING=OFF; then
            cd "$old_cwd"
            return 1
        fi

        if ! run_with_progress "Building CMake" 50 make -j"$jobs"; then
            cd "$old_cwd"
            return 1
        fi

        if ! run_with_progress "Installing CMake" 20 make -j$(nproc) install; then
            cd "$old_cwd"
            return 1
        fi
    fi

    cd "$old_cwd"
    rm -rf "$source_dir"
}