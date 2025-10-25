#!/usr/bin/env bash

# PHPV - Dependency installation functions

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

# Install zlib from source
install_zlib_from_source() {
    build_from_source "zlib" "1.3.1" "https://zlib.net/zlib-1.3.1.tar.gz" "--shared"
}

# Install OpenSSL from source
install_openssl_from_source() {
    local php_version="${1:-}"
    local version="3.0.13"
    
    # Use OpenSSL 1.0.x for PHP 5.x versions for compatibility
    if [[ -n "$php_version" && "$php_version" == 5.* ]]; then
        version="1.0.1u"
    # Use OpenSSL 1.1.x for PHP 7.x and 8.0.x versions for compatibility
    # PHP 8.0 does not support OpenSSL 3.0 due to removed RSA_SSLV23_PADDING constant
    elif [[ -n "$php_version" && ( "$php_version" == 7.* || "$php_version" == 8.0.* ) ]]; then
        version="1.1.1w"
    fi
    
    local url="https://www.openssl.org/source/openssl-$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/openssl-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/openssl-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        ./config --prefix="$PHPV_DEPS_DIR" --openssldir="$PHPV_DEPS_DIR/ssl" shared
        make -j$(nproc)
        make -j$(nproc) install
    else
        run_with_progress "Configuring OpenSSL" 30 ./config --prefix="$PHPV_DEPS_DIR" --openssldir="$PHPV_DEPS_DIR/ssl" shared || return 1
        run_with_progress "Building OpenSSL" 50 make -j$(nproc) || return 1
        run_with_progress "Installing OpenSSL" 20 make -j$(nproc) install || return 1
    fi
}

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

# Install libpng from source
install_libpng_from_source() {
    build_from_source "libpng" "1.6.40" "https://download.sourceforge.net/libpng/libpng-1.6.40.tar.gz"
}

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

# Install freetype from source
install_freetype_from_source() {
    build_from_source "freetype" "2.13.2" "https://download.savannah.gnu.org/releases/freetype/freetype-2.13.2.tar.gz"
}

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

# Install curl from source
install_curl_from_source() {
    local php_version="$1"
    local version
    local -a urls

    if [[ "$php_version" =~ ^5\.[0-2]\. ]]; then
        version="7.12.0"
    elif [[ "$php_version" == 5.* ]]; then
        version="7.29.0"
    else
        version="8.5.0"
    fi
    if [[ "$version" =~ ^7\. ]]; then
        urls+=("https://curl.se/download/old/curl-$version.tar.gz")
        urls+=("https://curl.se/download/archeology/curl-$version.tar.gz")
        urls+=("https://curl.haxx.se/download/curl-$version.tar.gz")
        urls+=("http://curl.se/download/old/curl-$version.tar.gz")
        urls+=("http://curl.se/download/archeology/curl-$version.tar.gz")
        urls+=("http://curl.haxx.se/download/curl-$version.tar.gz")
    fi
    urls+=("https://curl.se/download/curl-$version.tar.gz")
    local cache_file="$PHPV_CACHE_DIR/curl-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/curl-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        local downloaded=false
        for url in "${urls[@]}"; do
            if safe_download "$url" "$cache_file"; then
                downloaded=true
                break
            fi
            rm -f "$cache_file"
        done
        if [[ "$downloaded" != true ]]; then
            return 1
        fi
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    local configure_cmd="./configure --prefix=$PHPV_DEPS_DIR --with-openssl=$PHPV_DEPS_DIR"
    local restore_select_cache=false
    if [[ "$php_version" == 5.* ]]; then
        configure_cmd="$configure_cmd --without-libssh2 --disable-ldap --disable-ldaps" # Avoid modern libssh2 and LDAP API mismatches with legacy curl
        if [[ -z "${ac_cv_func_select:-}" ]]; then
            export ac_cv_func_select=yes
            restore_select_cache=true
        fi
        if [[ -z "${ac_cv_func_socket:-}" ]]; then
            export ac_cv_func_socket=yes
            restore_select_cache=true
        fi
    fi
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        eval "$configure_cmd"
        if [[ "$restore_select_cache" == true ]]; then
            unset ac_cv_func_select ac_cv_func_socket
        fi
        make -j$(nproc)
        make -j$(nproc) install
    else
        run_with_progress "Configuring curl" 30 eval "$configure_cmd" || return 1
        if [[ "$restore_select_cache" == true ]]; then
            unset ac_cv_func_select ac_cv_func_socket
        fi
        run_with_progress "Building curl" 50 make -j$(nproc) || return 1
        run_with_progress "Installing curl" 20 make -j$(nproc) install || return 1
    fi
}

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

# Install libzip from source
install_libzip_from_source() {
    build_with_cmake "libzip" "1.10.1" "https://libzip.org/download/libzip-1.10.1.tar.gz"
}

# Install unixODBC from source
install_unixodbc_from_source() {
    build_from_source "unixODBC" "2.3.12" "https://www.unixodbc.org/unixODBC-2.3.12.tar.gz"
}

# Install MySQL ODBC driver from source
install_mysql_odbc_from_source() {
    local version="1.4.17"
    local url="https://www.freetds.org/files/stable/freetds-${version}.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/freetds-${version}.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/freetds-${version}"
    
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        ./configure --prefix="$PHPV_DEPS_DIR" \
                    --with-unixodbc="$PHPV_DEPS_DIR" \
                    --with-openssl="$PHPV_DEPS_DIR" \
                    --enable-sybase-compat \
                    --disable-dependency-tracking
        make -j$(nproc)
        make -j$(nproc) install
    else
        run_with_progress "Configuring FreeTDS" 30 ./configure --prefix="$PHPV_DEPS_DIR" \
                    --with-unixodbc="$PHPV_DEPS_DIR" \
                    --with-openssl="$PHPV_DEPS_DIR" \
                    --enable-sybase-compat \
                    --disable-dependency-tracking || return 1
        run_with_progress "Building FreeTDS" 50 make -j$(nproc) || return 1
        run_with_progress "Installing FreeTDS" 20 make -j$(nproc) install || return 1
    fi
}

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

install_mysql_legacy_connector_from_source() {
    local version="${1:-6.1.11}"
    local binary_basename="mysql-connector-c-${version}-linux-glibc2.12-x86_64"  # Default (will be overridden)
    
    # Determine glibc and architecture suffixes based on version
    local glibc_suffix="glibc2.3"  # Default for very old versions (< 6.1.0)
    local arch_suffix="x86-x64bit"  # Default for very old versions (< 6.1.0)
    if [[ "$version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
        local major=${BASH_REMATCH[1]}
        local minor=${BASH_REMATCH[2]}
        local patch=${BASH_REMATCH[3]}
        if (( major > 6 )) || (( major == 6 && minor > 1 )) || (( major == 6 && minor == 1 && patch >= 10 )); then
            glibc_suffix="glibc2.12"
            arch_suffix="x86_64"
        elif (( major == 6 && minor == 1 && patch >= 0 )); then
            glibc_suffix="glibc2.5"
            arch_suffix="x86_64"
        fi
    fi
    
    # Update binary_basename with the correct suffixes
    binary_basename="mysql-connector-c-${version}-linux-${glibc_suffix}-${arch_suffix}"
    
    # Determine primary download URL based on version
    local primary_url
    if [[ "$version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
        local major=${BASH_REMATCH[1]}
        local minor=${BASH_REMATCH[2]}
        local patch=${BASH_REMATCH[3]}
        if (( major < 6 )) || (( major == 6 && minor < 1 )) || (( major == 6 && minor == 1 && patch <= 11 )); then
            primary_url="https://cdn.mysql.com/archives/mysql-connector-c/${binary_basename}.tar.gz"
        else
            primary_url="https://cdn.mysql.com/Downloads/Connector-C/${binary_basename}.tar.gz"
        fi
    else
        primary_url="https://cdn.mysql.com/Downloads/Connector-C/${binary_basename}.tar.gz"
    fi
    
    local -a binary_urls=(
        "$primary_url"
        "https://downloads.mysql.com/archives/get/p/19/file/${binary_basename}.tar.gz"
    )
    local binary_cache_file="$PHPV_CACHE_DIR/${binary_basename}.tar.gz"
    local binary_extract_dir="$PHPV_CACHE_DIR/${binary_basename}-extract"

    # Validate cache directory early to prevent null directory errors
    if [[ -z "$PHPV_CACHE_DIR" || ! -d "$PHPV_CACHE_DIR" ]]; then
        log_error "PHPV_CACHE_DIR is not set or does not exist: $PHPV_CACHE_DIR"
        return 1
    fi

    local old_cwd
    old_cwd=$(pwd)

    if [[ ! -f "$binary_cache_file" ]]; then
        local downloaded=false
        for url in "${binary_urls[@]}"; do
            if safe_download "$url" "$binary_cache_file"; then
                downloaded=true
                break
            fi
            rm -f "$binary_cache_file"
        done
        if [[ "$downloaded" != true ]]; then
            rm -f "$binary_cache_file"
        fi
    fi

    if [[ -f "$binary_cache_file" ]]; then
        rm -rf "$binary_extract_dir"
        mkdir -p "$binary_extract_dir" || {
            log_error "Failed to create binary extract directory: $binary_extract_dir"
            return 1
        }
        if tar -xzf "$binary_cache_file" -C "$binary_extract_dir"; then
            local staging_dir="$binary_extract_dir/$binary_basename"
            [[ -d "$staging_dir" ]] || staging_dir="$binary_extract_dir"

            rm -rf "$PHPV_DEPS_DIR/bin" "$PHPV_DEPS_DIR/include" "$PHPV_DEPS_DIR/lib" "$PHPV_DEPS_DIR/share" "$PHPV_DEPS_DIR/lib64"
            mkdir -p "$PHPV_DEPS_DIR/bin" "$PHPV_DEPS_DIR/include" "$PHPV_DEPS_DIR/lib" "$PHPV_DEPS_DIR/share" "$PHPV_DEPS_DIR/lib64"

            if [[ -d "$staging_dir/bin" ]]; then
                cp -a "$staging_dir/bin/." "$PHPV_DEPS_DIR/bin/"
            fi
            if [[ -d "$staging_dir/include" ]]; then
                cp -a "$staging_dir/include/." "$PHPV_DEPS_DIR/include/"
            fi
            if [[ -d "$staging_dir/lib" ]]; then
                cp -a "$staging_dir/lib/." "$PHPV_DEPS_DIR/lib/"
            fi
            if [[ -d "$staging_dir/lib64" ]]; then
                cp -a "$staging_dir/lib64/." "$PHPV_DEPS_DIR/lib64/"
            fi
            if [[ -d "$staging_dir/share" ]]; then
                cp -a "$staging_dir/share/." "$PHPV_DEPS_DIR/share/"
            fi

            if [[ -f "$PHPV_DEPS_DIR/lib/libmysqlclient.so" && ! -e "$PHPV_DEPS_DIR/lib64/libmysqlclient.so" ]]; then
                ln -sf "$PHPV_DEPS_DIR/lib/libmysqlclient.so" "$PHPV_DEPS_DIR/lib64/libmysqlclient.so"
            fi

            rm -rf "$binary_extract_dir"

            if [[ -x "$PHPV_DEPS_DIR/bin/mysql_config" ]]; then
                normalize_mysql_config "$PHPV_DEPS_DIR/bin/mysql_config"
                return 0
            fi
        fi
        rm -rf "$binary_extract_dir"
    fi

    # Fallback: Download and build MySQL source directly (no RPM tools needed)
    local mysql_version="5.0.77"
    local mysql_url="https://downloads.mysql.com/archives/mysql-${mysql_version}.tar.gz"
    local mysql_cache_file="$PHPV_CACHE_DIR/mysql-${mysql_version}.tar.gz"
    local mysql_extract_dir="$PHPV_CACHE_DIR/mysql-${mysql_version}-extract"

    if [[ ! -f "$mysql_cache_file" ]]; then
        safe_download "$mysql_url" "$mysql_cache_file" || {
            log_error "Failed to download MySQL source from $mysql_url"
            return 1
        }
    fi

    rm -rf "$mysql_extract_dir"
    mkdir -p "$mysql_extract_dir" || {
        log_error "Failed to create MySQL extract directory: $mysql_extract_dir"
        return 1
    }
    cd "$mysql_extract_dir" || {
        log_error "Failed to cd to MySQL extract directory: $mysql_extract_dir"
        return 1
    }

    # Extract MySQL source directly
    tar -xzf "$mysql_cache_file" --strip-components=1 || {
        log_error "Failed to extract MySQL source tarball"
        cd "$old_cwd" 2>/dev/null || true
        rm -rf "$mysql_extract_dir"
        return 1
    }

    # Configure and build MySQL client only (same as before, but in the direct extract dir)
    # Use the already-configured LLVM toolchain from the parent context

    local cc_binary="${CC:-}"
    local cxx_binary="${CXX:-}"

    if [[ -z "$cc_binary" || -z "$cxx_binary" ]]; then
        if [[ -n "$LLVM_HOME" && -x "$LLVM_HOME/bin/clang" && -x "$LLVM_HOME/bin/clang++" ]]; then
            cc_binary="$LLVM_HOME/bin/clang"
            cxx_binary="$LLVM_HOME/bin/clang++"
        elif [[ -n "$effective_llvm" ]]; then
            local llvm_candidate="$PHPV_DEPS_DIR/llvm-$effective_llvm"
            if [[ -x "$llvm_candidate/bin/clang" && -x "$llvm_candidate/bin/clang++" ]]; then
                cc_binary="$llvm_candidate/bin/clang"
                cxx_binary="$llvm_candidate/bin/clang++"
            fi
        fi
    fi

    if [[ -z "$cc_binary" || -z "$cxx_binary" ]]; then
        local latest_llvm_dir=""
        if [[ -d "$PHPV_DEPS_DIR" ]]; then
            latest_llvm_dir=$(find "$PHPV_DEPS_DIR" -maxdepth 1 -type d -name "llvm-*" | sort -V | tail -n1 2>/dev/null || true)
        fi
        if [[ -n "$latest_llvm_dir" && -x "$latest_llvm_dir/bin/clang" && -x "$latest_llvm_dir/bin/clang++" ]]; then
            cc_binary="$latest_llvm_dir/bin/clang"
            cxx_binary="$latest_llvm_dir/bin/clang++"
        fi
    fi

    if [[ -z "$cc_binary" || -z "$cxx_binary" ]]; then
        log_error "Failed to locate Clang toolchain for MySQL legacy connector build"
        cd "$old_cwd" 2>/dev/null || true
        rm -rf "$mysql_extract_dir"
        return 1
    fi

    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        ./configure \
            --prefix="$PHPV_DEPS_DIR" \
            --without-server \
            --without-docs \
            --without-man \
            --without-bench \
            --enable-thread-safe-client \
            --with-zlib-dir="$PHPV_DEPS_DIR" \
            --with-openssl="$PHPV_DEPS_DIR" \
            --with-named-curses-libs="$PHPV_DEPS_DIR/lib/libncurses.so" \
            CC="$cc_binary" \
            CXX="$cxx_binary" \
            CFLAGS="-I$PHPV_DEPS_DIR/include -Wno-implicit-int -Wno-implicit-function-declaration" \
            CXXFLAGS="-I$PHPV_DEPS_DIR/include" \
            LDFLAGS="-L$PHPV_DEPS_DIR/lib -L$PHPV_DEPS_DIR/lib64" || {
            log_error "MySQL configure failed"
            cd "$old_cwd" 2>/dev/null || true
            rm -rf "$mysql_extract_dir"
            return 1
        }
    else
        if ! run_with_progress "Configuring MySQL Connector" 30 ./configure \
            --prefix="$PHPV_DEPS_DIR" \
            --without-server \
            --without-docs \
            --without-man \
            --without-bench \
            --enable-thread-safe-client \
            --with-zlib-dir="$PHPV_DEPS_DIR" \
            --with-openssl="$PHPV_DEPS_DIR" \
            --with-named-curses-libs="$PHPV_DEPS_DIR/lib/libncurses.so" \
            CC="$cc_binary" \
            CXX="$cxx_binary" \
            CFLAGS="-I$PHPV_DEPS_DIR/include -Wno-implicit-int -Wno-implicit-function-declaration" \
            CXXFLAGS="-I$PHPV_DEPS_DIR/include" \
            LDFLAGS="-L$PHPV_DEPS_DIR/lib -L$PHPV_DEPS_DIR/lib64"; then
            log_error "MySQL configure failed"
            cd "$old_cwd" 2>/dev/null || true
            rm -rf "$mysql_extract_dir"
            return 1
        fi
    fi

    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        make -j$(nproc) || {
            log_error "MySQL build failed"
            cd "$old_cwd" 2>/dev/null || true
            rm -rf "$mysql_extract_dir"
            return 1
        }

        make -j$(nproc) install || {
            log_error "MySQL install failed"
            cd "$old_cwd" 2>/dev/null || true
            rm -rf "$mysql_extract_dir"
            return 1
        }
    else
        if ! run_with_progress "Building MySQL Connector" 50 make -j$(nproc); then
            log_error "MySQL build failed"
            cd "$old_cwd" 2>/dev/null || true
            rm -rf "$mysql_extract_dir"
            return 1
        fi

        if ! run_with_progress "Installing MySQL Connector" 20 make -j$(nproc) install; then
            log_error "MySQL install failed"
            cd "$old_cwd" 2>/dev/null || true
            rm -rf "$mysql_extract_dir"
            return 1
        fi
    fi

    cd "$old_cwd" 2>/dev/null || true
    rm -rf "$mysql_extract_dir"

    if [[ -x "$PHPV_DEPS_DIR/bin/mysql_config" ]]; then
        normalize_mysql_config "$PHPV_DEPS_DIR/bin/mysql_config"
        return 0
    fi

    log_error "mysql_config not found after installation"
    return 1
}

install_mysql_legacy_from_source() {
    local version="$1"
    local url="https://downloads.mysql.com/archives/mysql-${version}.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/mysql-${version}.tar.gz"
    local source_dir="$PHPV_CACHE_DIR/mysql-${version}-src"
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

    # Configure with minimal options for client library only
    ../configure \
        --prefix="$PHPV_DEPS_DIR" \
        --without-server \
        --without-docs \
        --without-man \
        --without-bench \
        --enable-thread-safe-client \
        --with-openssl="$PHPV_DEPS_DIR" \
        --with-zlib-dir="$PHPV_DEPS_DIR" \
        --enable-shared \
        --disable-static \
        CFLAGS="-Wno-implicit-int -Wno-implicit-function-declaration" || {
        cd "$old_cwd"
        return 1
    }

    make -j$(nproc) || {
        cd "$old_cwd"
        return 1
    }

    make -j$(nproc) install || {
        cd "$old_cwd"
        return 1
    }

    cd "$old_cwd"

    if [[ -x "$PHPV_DEPS_DIR/bin/mysql_config" ]]; then
        normalize_mysql_config "$PHPV_DEPS_DIR/bin/mysql_config"
    else
        log_error "mysql_config not found after installing MySQL $version"
        return 1
    fi
}

# Install PostgreSQL client libraries from source (dev libs only, no server)
install_postgresql_client_from_source() {
    local php_version="$1"
    local version="15.4"  # Stable version compatible with most PHP versions; adjust if needed

    if [[ "$php_version" =~ ^5\. ]]; then
        version="8.0.26"  # Use older version for PHP 5.x OpenSSL compatibility
    fi

    local url="https://ftp.postgresql.org/pub/source/v${version}/postgresql-${version}.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/postgresql-${version}.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/postgresql-${version}"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    
    # Set paths for custom dependencies, including system includes for missing headers
    export CPPFLAGS="-I$PHPV_DEPS_DIR/include $CPPFLAGS"
    export LDFLAGS="-L$PHPV_DEPS_DIR/lib $LDFLAGS"

    # For PHP 5.x, suppress clang warnings for legacy code (implicit declarations, etc.)
    if [[ "$php_version" =~ ^5\. ]]; then
        export CFLAGS="-Wno-implicit-int -Wno-implicit-function-declaration -Wno-deprecated-declarations -Wno-deprecated-non-prototype -Wno-visibility -Wno-pointer-sign -fcommon $CFLAGS"
    fi
    
    # Configure for client-only build (no server components)
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        # For PostgreSQL versions with major version less than 10, there's no --without-server option, so we skip building server components manually
        local major="${version%%.*}"
        if (( major < 10 )); then
            ./configure --prefix="$PHPV_DEPS_DIR" --with-openssl="$PHPV_DEPS_DIR" --with-zlib="$PHPV_DEPS_DIR"
        else
            ./configure --prefix="$PHPV_DEPS_DIR" --without-server --without-openssl --with-zlib
        fi
        # Build only libpq and headers (skip src/bin to avoid pg_dump dependency)
        make -j$(nproc) -C src/interfaces/libpq
        make -j$(nproc) -C src/include
        make -j$(nproc) -C src/interfaces/libpq install
        make -j$(nproc) -C src/include install
    else
        # For PostgreSQL versions with major version less than 10, there's no --without-server option, so we skip building server components manually
        local major="${version%%.*}"
        if (( major < 10 )); then
            run_with_progress "Configuring PostgreSQL client" 30 ./configure --prefix="$PHPV_DEPS_DIR" --with-openssl="$PHPV_DEPS_DIR" --with-zlib="$PHPV_DEPS_DIR" || return 1
        else
            run_with_progress "Configuring PostgreSQL client" 30 ./configure --prefix="$PHPV_DEPS_DIR" --without-server --without-openssl --with-zlib || return 1
        fi
        run_with_progress "Building PostgreSQL client libpq" 50 make -j$(nproc) -C src/interfaces/libpq || return 1
        run_with_progress "Building PostgreSQL client includes" 75 make -j$(nproc) -C src/include || return 1
        run_with_progress "Installing PostgreSQL client libpq" 100 make -j$(nproc) -C src/interfaces/libpq install || return 1
        run_with_progress "Installing PostgreSQL client includes" 100 make -j$(nproc) -C src/include install || return 1
    fi
}

ensure_mysql_client_for_php() {
    local php_version="$1"

    if [[ "$php_version" == 5.* ]]; then
        # For PHP 5.x, use ODBC instead of MySQL Connector/C
        log_info "Installing unixODBC and MySQL ODBC driver for PHP $php_version compatibility..."
        
        if [[ ! -f "$PHPV_DEPS_DIR/lib/libodbc.so" ]]; then
            install_unixodbc_from_source || return 1
        fi
        
        if [[ ! -f "$PHPV_DEPS_DIR/lib/libmyodbc.so" ]]; then
            install_mysql_odbc_from_source || return 1
        fi
        
        # Note: Native MySQL extensions (--with-mysqli, --with-pdo-mysql) won't be available.
        # Users can connect via odbc extension with DSN like 'odbc:DSN=my_mysql_dsn'
        return 0
    else
        local required_version="3.3.7"
        local current_version=""
        if [[ -x "$PHPV_DEPS_DIR/bin/mysql_config" ]]; then
            current_version="$($PHPV_DEPS_DIR/bin/mysql_config --version 2>/dev/null || true)"
        fi
        if [[ "$current_version" != ${required_version}* ]]; then
            log_info "Installing MariaDB Connector/C $required_version..."
            install_mariadb_connector_from_source || return 1
        fi
    fi
}