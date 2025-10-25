#!/usr/bin/env bash

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