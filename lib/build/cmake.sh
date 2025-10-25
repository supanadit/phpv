#!/usr/bin/env bash

# PHPV - Build utilities

# Build with CMake
build_with_cmake() {
    local name="$1"
    local version="$2"
    local url="$3"
    local cmake_args="${4:-}"

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

    mkdir -p build
    cd build

    # Completely isolate cmake from custom environment
    local old_path="$PATH"
    local old_ld_library_path="${LD_LIBRARY_PATH:-}"
    local old_pkg_config_path="${PKG_CONFIG_PATH:-}"
    local old_ldflags="${LDFLAGS:-}"
    local old_cppflags="${CPPFLAGS:-}"

    # Keep environment narrow but include freshly built toolchain
    export PATH="$PHPV_DEPS_DIR/bin:/usr/local/bin:/usr/bin:/bin"
    export LD_LIBRARY_PATH="$PHPV_DEPS_DIR/lib:$PHPV_DEPS_DIR/lib64${old_ld_library_path:+:$old_ld_library_path}"
    export PKG_CONFIG_PATH="$PHPV_DEPS_DIR/lib/pkgconfig:$PHPV_DEPS_DIR/lib64/pkgconfig"
    export LDFLAGS="-L$PHPV_DEPS_DIR/lib -L$PHPV_DEPS_DIR/lib64"
    export CPPFLAGS="-I$PHPV_DEPS_DIR/include"

    local cmake_cmd="$PHPV_DEPS_DIR/bin/cmake"
    if [[ ! -x "$cmake_cmd" ]]; then
        cmake_cmd="cmake"
    fi

    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        "$cmake_cmd" \
            -DCMAKE_INSTALL_PREFIX="$PHPV_DEPS_DIR" \
            -DCMAKE_PREFIX_PATH="$PHPV_DEPS_DIR" \
            $cmake_args ..
        local cmake_result=$?
        
        # Restore environment
        export PATH="$old_path"
        export LD_LIBRARY_PATH="$old_ld_library_path"
        export PKG_CONFIG_PATH="$old_pkg_config_path"
        export LDFLAGS="$old_ldflags"
        export CPPFLAGS="$old_cppflags"
        
        if [[ $cmake_result -ne 0 ]]; then
            return 1
        fi
        
        make -j$(nproc)
        make -j$(nproc) install
    else
        if ! run_with_progress "Configuring $name" 30 "$cmake_cmd" \
            -DCMAKE_INSTALL_PREFIX="$PHPV_DEPS_DIR" \
            -DCMAKE_PREFIX_PATH="$PHPV_DEPS_DIR" \
            $cmake_args ..; then
            # Restore environment
            export PATH="$old_path"
            export LD_LIBRARY_PATH="$old_ld_library_path"
            export PKG_CONFIG_PATH="$old_pkg_config_path"
            export LDFLAGS="$old_ldflags"
            export CPPFLAGS="$old_cppflags"
            return 1
        fi
        
        # Restore environment
        export PATH="$old_path"
        export LD_LIBRARY_PATH="$old_ld_library_path"
        export PKG_CONFIG_PATH="$old_pkg_config_path"
        export LDFLAGS="$old_ldflags"
        export CPPFLAGS="$old_cppflags"
        
        run_with_progress "Building $name" 50 make -j$(nproc) || return 1
        run_with_progress "Installing $name" 20 make -j$(nproc) install || return 1
    fi
}