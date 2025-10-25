#!/usr/bin/env bash

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