#!/usr/bin/env bash

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