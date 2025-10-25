#!/usr/bin/env bash

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