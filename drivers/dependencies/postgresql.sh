#!/usr/bin/env bash

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