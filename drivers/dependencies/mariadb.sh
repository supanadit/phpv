#!/usr/bin/env bash

install_mariadb_connector_from_source() {
    local version="3.3.7"
    local url="https://archive.mariadb.org/connector-c-$version/mariadb-connector-c-$version-src.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/mariadb-connector-c-$version.tar.gz"
    local source_dir="$PHPV_CACHE_DIR/mariadb-connector-c-$version-src"
    local build_dir="$source_dir/build"
    local old_cwd
    old_cwd=$(pwd)

    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi

    rm -rf "$source_dir"
    mkdir -p "$source_dir"
    tar -xzf "$cache_file" -C "$source_dir" --strip-components=1

    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"

    rm -rf "$PHPV_DEPS_DIR/lib/mariadb"
    rm -f "$PHPV_DEPS_DIR/lib/libmysqlclient.so" \
        "$PHPV_DEPS_DIR/lib/libmysqlclient.so.18" \
        "$PHPV_DEPS_DIR/lib/libmysqlclient.a"

    if ! install_cmake_from_source; then
        cd "$old_cwd"
        return 1
    fi

    local cmake_cmd="$PHPV_DEPS_DIR/bin/cmake"
    if [[ ! -x "$cmake_cmd" ]]; then
        cmake_cmd="cmake"
    fi

    local old_path="$PATH"
    local old_ld_library_path="${LD_LIBRARY_PATH:-}"
    local old_pkg_config="${PKG_CONFIG_PATH:-}"
    local old_ldflags="${LDFLAGS:-}"
    local old_cppflags="${CPPFLAGS:-}"

    export PATH="$PHPV_DEPS_DIR/bin:/usr/local/bin:/usr/bin:/bin"
    export LD_LIBRARY_PATH="$PHPV_DEPS_DIR/lib:$PHPV_DEPS_DIR/lib64${old_ld_library_path:+:$old_ld_library_path}"
    export PKG_CONFIG_PATH="$PHPV_DEPS_DIR/lib/pkgconfig:$PHPV_DEPS_DIR/lib64/pkgconfig"
    export LDFLAGS="-L$PHPV_DEPS_DIR/lib -L$PHPV_DEPS_DIR/lib64"
    export CPPFLAGS="-I$PHPV_DEPS_DIR/include"

    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        if ! "$cmake_cmd" .. \
            -DCMAKE_INSTALL_PREFIX="$PHPV_DEPS_DIR" \
            -DCMAKE_PREFIX_PATH="$PHPV_DEPS_DIR" \
            -DWITH_EXTERNAL_ZLIB=ON \
            -DZLIB_INCLUDE_DIR="$PHPV_DEPS_DIR/include" \
            -DZLIB_LIBRARY="$PHPV_DEPS_DIR/lib/libz.so" ; then
            cd "$old_cwd"
            return 1
        fi

        make -j$(nproc)
        make -j$(nproc) install
    else
        if ! run_with_progress "Configuring MariaDB Connector" 30 "$cmake_cmd" .. \
            -DCMAKE_INSTALL_PREFIX="$PHPV_DEPS_DIR" \
            -DCMAKE_PREFIX_PATH="$PHPV_DEPS_DIR" \
            -DWITH_EXTERNAL_ZLIB=ON \
            -DZLIB_INCLUDE_DIR="$PHPV_DEPS_DIR/include" \
            -DZLIB_LIBRARY="$PHPV_DEPS_DIR/lib/libz.so" ; then
            # Restore environment
            export PATH="$old_path"
            export LD_LIBRARY_PATH="$old_ld_library_path"
            export PKG_CONFIG_PATH="$old_pkg_config"
            export LDFLAGS="$old_ldflags"
            export CPPFLAGS="$old_cppflags"
            return 1
        fi
        
        # Restore environment
        export PATH="$old_path"
        export LD_LIBRARY_PATH="$old_ld_library_path"
        export PKG_CONFIG_PATH="$old_pkg_config"
        export LDFLAGS="$old_ldflags"
        export CPPFLAGS="$old_cppflags"
        
        run_with_progress "Building MariaDB Connector" 50 make -j$(nproc) || return 1
        run_with_progress "Installing MariaDB Connector" 20 make -j$(nproc) install || return 1
    fi

    export PATH="$old_path"
    export LD_LIBRARY_PATH="$old_ld_library_path"
    export PKG_CONFIG_PATH="$old_pkg_config"
    export LDFLAGS="$old_ldflags"
    export CPPFLAGS="$old_cppflags"

    if [[ -x "$PHPV_DEPS_DIR/bin/mariadb_config" && ! -e "$PHPV_DEPS_DIR/bin/mysql_config" ]]; then
        cat > "$PHPV_DEPS_DIR/bin/mysql_config" << 'EOF'
#!/usr/bin/env bash
exec "$(dirname "$0")/mariadb_config" "$@"
EOF
        chmod +x "$PHPV_DEPS_DIR/bin/mysql_config"
    fi

    local mariadb_lib_dir="$PHPV_DEPS_DIR/lib/mariadb"
    if [[ -d "$mariadb_lib_dir" ]]; then
        if [[ -f "$mariadb_lib_dir/libmariadb.so" ]]; then
            ln -sf "$mariadb_lib_dir/libmariadb.so" "$PHPV_DEPS_DIR/lib/libmysqlclient.so"
        fi
        if [[ -f "$mariadb_lib_dir/libmariadb.so.3" ]]; then
            ln -sf "$mariadb_lib_dir/libmariadb.so.3" "$PHPV_DEPS_DIR/lib/libmysqlclient.so.18"
        fi
        if [[ -f "$mariadb_lib_dir/libmariadb.a" ]]; then
            ln -sf "$mariadb_lib_dir/libmariadb.a" "$PHPV_DEPS_DIR/lib/libmysqlclient.a"
        fi
    fi

    cd "$old_cwd"
}