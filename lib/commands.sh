#!/usr/bin/env bash

# PHPV - Main command functions

detect_readdir_r_variant() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/readdir_r.XXXXXX") || return 1

    cat > "$tmp_dir/test_posix.c" <<'EOF'
#define _REENTRANT
#include <sys/types.h>
#include <dirent.h>
int main(void) {
    DIR *dir = 0;
    struct dirent entry;
    struct dirent *result = 0;
    return readdir_r(dir, &entry, &result);
}
EOF

    local variant="unknown"
    if "$cc" -o "$tmp_dir/test_posix" "$tmp_dir/test_posix.c" >/dev/null 2>&1; then
        variant="posix"
    else
        cat > "$tmp_dir/test_old.c" <<'EOF'
#define _REENTRANT
#include <sys/types.h>
#include <dirent.h>
int main(void) {
    DIR *dir = 0;
    struct dirent entry;
    return readdir_r(dir, &entry);
}
EOF
        if "$cc" -o "$tmp_dir/test_old" "$tmp_dir/test_old.c" >/dev/null 2>&1; then
            variant="old"
        fi
    fi

    rm -rf "$tmp_dir"
    [[ "$variant" == "unknown" ]] && return 1
    printf '%s\n' "$variant"
}

detect_fork_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/fork_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_fork.c" <<'EOF'
#include <unistd.h>
int main(void) {
    return fork() == -1;
}
EOF

    if "$cc" -o "$tmp_dir/test_fork" "$tmp_dir/test_fork.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_waitpid_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/waitpid_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_waitpid.c" <<'EOF'
#include <sys/types.h>
#include <sys/wait.h>
int main(void) {
    int (*fn)(pid_t, int*, int) = waitpid;
    return fn == 0;
}
EOF

    if "$cc" -o "$tmp_dir/test_waitpid" "$tmp_dir/test_waitpid.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_sigaction_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/sigaction_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_sigaction.c" <<'EOF'
#include <signal.h>
static void noop(int sig) {(void)sig;}
int main(void) {
    struct sigaction sa;
    sa.sa_handler = noop;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    return sigaction(SIGINT, &sa, 0) == -1;
}
EOF

    if "$cc" -o "$tmp_dir/test_sigaction" "$tmp_dir/test_sigaction.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_wait_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/wait_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_wait.c" <<'EOF'
#include <sys/types.h>
#include <sys/wait.h>
int main(void) {
    int status = 0;
    return wait(&status) == -1;
}
EOF

    if "$cc" -o "$tmp_dir/test_wait" "$tmp_dir/test_wait.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_header_support() {
    local header="$1"
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/header_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_header.c" <<EOF
#include <$header>
int main(void) { return 0; }
EOF

    if "$cc" -c "$tmp_dir/test_header.c" -o "$tmp_dir/test_header.o" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

# Install PHP version
install_php_version() {
    local input_version="$1"
    
    if [[ -z "$input_version" ]]; then
        log_error "Please specify a version to install"
        return 1
    fi
    
    # Resolve the actual version to install
    local version
    version=$(resolve_latest_version "$input_version")
    
    if [[ -z "$version" ]]; then
        log_error "No available version found matching '$input_version'"
        return 1
    fi
    
    # If we resolved to a different version, inform the user
    if [[ "$version" != "$input_version" ]]; then
        log_info "Installing latest version $version (matched from '$input_version')"
    fi
    
    local install_dir="$PHPV_VERSIONS_DIR/$version"
    local cache_file="$PHPV_CACHE_DIR/php-$version.tar.gz"

    if is_version_installed "$version"; then
        log_warning "PHP $version is already installed"
        return 0
    fi

    local resolved_llvm
    resolved_llvm=$(resolve_llvm_version_for_php "$version")
    [[ -z "$resolved_llvm" ]] && resolved_llvm="$PHPV_LLVM_VERSION"

    local version_deps_dir
    version_deps_dir=$(get_deps_dir_for_version "$version" "$resolved_llvm") || {
        log_error "Failed to resolve dependency directory for $version with LLVM $resolved_llvm"
        return 1
    }

    local PHPV_DEPS_DIR="$version_deps_dir"
    local php_old_cflags="$CFLAGS"
    local php_old_cxxflags="$CXXFLAGS"
    local php_old_cppflags="$CPPFLAGS"
    local php_old_ldflags="$LDFLAGS"
    local php_restore_env=false
    local php_extra_ldflags=""
    mkdir -p "$PHPV_DEPS_DIR"
    mkdir -p "$PHPV_DEPS_DIR/lib" "$PHPV_DEPS_DIR/lib64" "$PHPV_DEPS_DIR/include"
    log_info "Using isolated dependency prefix at $PHPV_DEPS_DIR"

    # Save LLVM version for this PHP installation
    mkdir -p "$install_dir"
    echo "$resolved_llvm" > "$install_dir/.llvm_version"

    log_info "Installing PHP $version..."

    if [[ -n "$resolved_llvm" && "$resolved_llvm" != "$PHPV_LLVM_VERSION" ]]; then
        log_info "Using LLVM $resolved_llvm for PHP $version"
    fi

    # Add a flag to skip LLVM (e.g., PHPV_SKIP_LLVM=1)
    PHPV_SKIP_LLVM="${PHPV_SKIP_LLVM:-1}"

    # In install_php_version, before calling ensure_llvm_toolchain:
    if [[ "$PHPV_SKIP_LLVM" != "1" ]]; then
        ensure_llvm_toolchain "$resolved_llvm" || return 1
    else
        # Use system compiler
        unset CC CXX AR NM RANLIB LLVM_HOME
    fi


    local active_llvm="${PHPV_ACTIVE_LLVM_VERSION:-$resolved_llvm}"
    if [[ "$active_llvm" != "$resolved_llvm" ]]; then
        log_warning "LLVM $resolved_llvm was requested but using $active_llvm due to availability"
    fi

    if ! command -v make &> /dev/null; then
        log_error "GNU make is required but not installed"
        log_info "Install make from source: https://ftp.gnu.org/gnu/make/"
        return 1
    fi
    
    # Set environment for custom dependencies (support both lib and lib64)
    export PKG_CONFIG_PATH="$PHPV_DEPS_DIR/lib/pkgconfig:$PHPV_DEPS_DIR/lib64/pkgconfig:$PKG_CONFIG_PATH"
    export LDFLAGS="-L$PHPV_DEPS_DIR/lib -L$PHPV_DEPS_DIR/lib64 $LDFLAGS"
    export CPPFLAGS="-I$PHPV_DEPS_DIR/include $CPPFLAGS"
    if [[ "$version" != 5.* ]]; then
        export CPPFLAGS="-D_GNU_SOURCE -D_POSIX_C_SOURCE=200809L $CPPFLAGS"
        export CFLAGS="-D_GNU_SOURCE -D_POSIX_C_SOURCE=200809L $CFLAGS"
    fi
    export LD_LIBRARY_PATH="$PHPV_DEPS_DIR/lib:$PHPV_DEPS_DIR/lib64:$LD_LIBRARY_PATH"
    
    # For PHP 5.x, DSA_get_default_method is in libcrypto, not libssl
    # Add libcrypto to LDFLAGS to make the function available during configure checks
    if [[ "$version" == 5.* ]]; then
        php_restore_env=true
        php_extra_ldflags="-lssl -lcrypto"
        export CFLAGS="-Wno-error -Wno-error=return-type -Wno-implicit-int -Wno-implicit-function-declaration -Wno-deprecated-declarations -Wno-deprecated-non-prototype -Wno-visibility -Wno-pointer-sign -fcommon $CFLAGS"
        export CXXFLAGS="-Wno-register $CXXFLAGS"
        export CPPFLAGS="-D_GNU_SOURCE -DHAVE_STDARG_PROTOTYPES=1 -D_BSD_SOURCE -D_DEFAULT_SOURCE -D_POSIX_C_SOURCE=200112L -D_XOPEN_SOURCE=600 $CPPFLAGS"
    fi

    if [[ "$php_restore_env" == true ]]; then
        local __php_restore_cmd
        printf -v __php_restore_cmd 'export CFLAGS=%q; export CXXFLAGS=%q; export CPPFLAGS=%q; export LDFLAGS=%q; trap - RETURN;' \
            "$php_old_cflags" "$php_old_cxxflags" "$php_old_cppflags" "$php_old_ldflags"
        trap "$__php_restore_cmd" RETURN
    fi
    
    # Install required dependencies from source if not present
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libz.so" ]]; then
        log_info "Installing zlib from source..."
        install_zlib_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib64/libssl.so" ]] && [[ ! -f "$PHPV_DEPS_DIR/lib/libssl.so" ]]; then
        log_info "Installing OpenSSL from source..."
        install_openssl_from_source "$version" || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libxml2.so" ]]; then
        log_info "Installing libxml2 from source..."
        install_libxml2_from_source "$version" || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libonig.so" ]]; then
        log_info "Installing oniguruma from source..."
        install_oniguruma_from_source "$version" || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libpng.so" ]]; then
        log_info "Installing libpng from source..."
        install_libpng_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libjpeg.so" ]]; then
        log_info "Installing libjpeg from source..."
        install_libjpeg_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libfreetype.so" ]]; then
        log_info "Installing freetype from source..."
        install_freetype_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libicuuc.so" ]] && [[ "$version" != 5.* ]]; then
        log_info "Installing ICU from source..."
        install_icu_from_source "$version" || return 1
    fi
    local curl_required
    if [[ "$version" =~ ^5\.[0-2]\. ]]; then
        curl_required="7.12.0"
    elif [[ "$version" == 5.* ]]; then
        curl_required="7.29.0"
    else
        curl_required="8.5.0"
    fi
    local curl_current=""
    if [[ -x "$PHPV_DEPS_DIR/bin/curl-config" ]]; then
        curl_current="$($PHPV_DEPS_DIR/bin/curl-config --version 2>/dev/null | awk '{print $2}' || true)"
    fi
    if [[ "$curl_current" != "$curl_required" ]]; then
        log_info "Installing curl $curl_required from source..."
        install_curl_from_source "$version" || return 1
    fi

    install_cmake_from_source || return 1

    if [[ ! -f "$PHPV_DEPS_DIR/lib/libzip.so" ]]; then
        log_info "Installing libzip from source..."
        install_libzip_from_source || return 1
    fi
    ensure_mysql_client_for_php "$version" || return 1

    # Install PostgreSQL client libs if not present
    if [[ "$version" != 5.0.* ]] && [[ ! -f "$PHPV_DEPS_DIR/lib/libpq.so" ]]; then
        log_info "Installing PostgreSQL client from source..."
        install_postgresql_client_from_source "$version" || return 1
    fi

    prepend_path "$PHPV_DEPS_DIR/bin"
    
    # Download PHP source if not cached
    if [[ ! -f "$cache_file" ]]; then
        log_info "Downloading PHP $version source..."
        local download_url
        if [[ "$version" =~ ^4\. ]]; then
            download_url="https://museum.php.net/php4/php-$version.tar.gz"
        elif [[ "$version" =~ ^5\.[0-2]\. ]]; then
            download_url="https://museum.php.net/php5/php-$version.tar.gz"
        else
            download_url="https://www.php.net/distributions/php-$version.tar.gz"
        fi
        safe_download "$download_url" "$cache_file" || return 1
    fi
    
    # Extract and build
    local build_dir="$PHPV_CACHE_DIR/php-$version-build"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    
    log_info "Extracting PHP $version..."
    tar -xzf "$cache_file" -C "$build_dir" --strip-components=1
    
    cd "$build_dir"

    local -a configure_env=()
    local posix_macros="-D_GNU_SOURCE -D_POSIX_C_SOURCE=200809L"
    local readdir_variant=""
    if readdir_variant=$(detect_readdir_r_variant 2>/dev/null); then
        case "$readdir_variant" in
            posix)
                configure_env+=("ac_cv_func_readdir_r=yes" "ac_cv_what_readdir_r=POSIX")
                ;;
            old)
                configure_env+=("ac_cv_func_readdir_r=yes" "ac_cv_what_readdir_r=old-style")
                ;;
        esac
    fi

    log_info "Configuring PHP $version..."
    if [[ "$readdir_variant" == "posix" ]]; then
        log_info "Ensuring configure treats readdir_r as POSIX (3 arguments)"
    fi
    
    # Build configure flags based on PHP version
    local configure_flags=(
        --prefix="$install_dir"
        --enable-cli
        --enable-cgi
        --enable-fpm
        --with-config-file-path="$install_dir/etc"
        --with-config-file-scan-dir="$install_dir/etc/conf.d"
        --enable-mbstring
        --with-libxml-dir="$PHPV_DEPS_DIR"
        --with-onig="$PHPV_DEPS_DIR"
        --with-libzip="$PHPV_DEPS_DIR"
        --enable-bcmath
        --enable-calendar
        --enable-exif
        --enable-ftp
        --with-curl="$PHPV_DEPS_DIR"
        --enable-gd
        --with-png-dir="$PHPV_DEPS_DIR"
        --with-jpeg-dir="$PHPV_DEPS_DIR"
        --with-freetype-dir="$PHPV_DEPS_DIR"
        --enable-soap
        --enable-sockets
        --enable-pcntl
        --enable-shmop
        --enable-sysvmsg
        --enable-sysvsem
        --enable-sysvshm
    )

    if version_supports_opcache "$version"; then
        configure_flags+=(--enable-opcache)
    fi

    local pcntl_supported=true

    if detect_fork_support; then
        configure_env+=(ac_cv_func_fork=yes ac_cv_func_fork_works=yes)
    else
        log_warning "Failed to compile fork() probe; pcntl will be disabled"
        pcntl_supported=false
    fi

    if detect_waitpid_support; then
        configure_env+=(ac_cv_func_waitpid=yes ac_cv_func_waitpid_works=yes)
    else
        log_warning "Failed to compile waitpid() probe; pcntl will be disabled"
        pcntl_supported=false
    fi

    if detect_wait_support; then
        configure_env+=(ac_cv_func_wait=yes)
    else
        log_warning "Failed to compile wait() probe; pcntl will be disabled"
        pcntl_supported=false
    fi

    if detect_sigaction_support; then
        configure_env+=(ac_cv_func_sigaction=yes)
    else
        log_warning "Failed to compile sigaction() probe; pcntl will be disabled"
        pcntl_supported=false
    fi

    if detect_header_support "sys/wait.h"; then
        configure_env+=(ac_cv_header_sys_wait_h=yes)
    else
        log_warning "sys/wait.h header not available; pcntl will be disabled"
        pcntl_supported=false
    fi

    if [[ "$pcntl_supported" != true ]]; then
        local tmp_flags=()
        local flag
        for flag in "${configure_flags[@]}"; do
            [[ "$flag" == "--enable-pcntl" ]] && continue
            tmp_flags+=("$flag")
        done
        configure_flags=("${tmp_flags[@]}")
    fi
    
    # Add MySQL/ODBC support based on PHP version
    local php_restore_cache=false
    if [[ "$version" == 5.* ]]; then
        # For PHP 5.x, use ODBC instead of MySQL Connector/C
        configure_flags+=(--with-unixODBC="$PHPV_DEPS_DIR")
        if [[ "$version" =~ ^5\.[1-9] ]]; then
            # PDO_ODBC available from PHP 5.1+
            configure_flags+=(--with-pdo-odbc="unixODBC,$PHPV_DEPS_DIR")
        fi
        if [[ -z "${ac_cv_func_shutdown:-}" ]]; then
            export ac_cv_func_shutdown=yes
            php_restore_cache=true
        fi
    else
        # For PHP 7+, use MySQL client library
        configure_flags+=(--with-mysqli)
        configure_flags+=(--with-pdo-mysql)
    fi

    # Add PostgreSQL support (client libs only, for PHP 5.1+)
    local major minor
    IFS='.' read -r major minor _ <<< "$version"
    if (( major > 5 || (major == 5 && minor >= 1) )); then
        configure_flags+=(--with-pgsql="$PHPV_DEPS_DIR")
        configure_flags+=(--with-pdo-pgsql="$PHPV_DEPS_DIR")
    fi
    
    # Add version-specific flags
    if [[ "$version" =~ ^(8\.|9\.) ]]; then
        configure_flags+=(--with-openssl="$PHPV_DEPS_DIR")
        configure_flags+=(--with-zlib="$PHPV_DEPS_DIR")
    fi
    
    if [[ "$version" == 5.* ]]; then
        # For PHP 5.x: disable SOAP extension due to libxml2 compatibility issues
        configure_flags+=(--disable-soap)
        configure_flags+=(--disable-dom)
        configure_flags+=(--disable-simplexml)
        configure_flags+=(--disable-intl)
    fi
    
    if [[ -n "$php_extra_ldflags" ]]; then
        export LDFLAGS="$php_extra_ldflags $LDFLAGS"
    fi

    # Basic configuration - can be customized
    local configure_success=false
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        if [[ ${#configure_env[@]} -gt 0 ]]; then
            if env "${configure_env[@]}" ./configure "${configure_flags[@]}"; then
                configure_success=true
            fi
        else
            if ./configure "${configure_flags[@]}"; then
                configure_success=true
            fi
        fi
    else
        if [[ ${#configure_env[@]} -gt 0 ]]; then
            if run_with_progress "Configuring PHP $version" 40 env "${configure_env[@]}" ./configure "${configure_flags[@]}"; then
                configure_success=true
            fi
        else
            if run_with_progress "Configuring PHP $version" 40 ./configure "${configure_flags[@]}"; then
                configure_success=true
            fi
        fi
    fi

    if [[ "$configure_success" != true ]]; then
        log_error "Configuration failed. You may need to install development packages:"
        log_info "Ubuntu/Debian: sudo apt-get install libxml2-dev libssl-dev libcurl4-openssl-dev libonig-dev libzip-dev"
        log_info "CentOS/RHEL: sudo yum install libxml2-devel openssl-devel curl-devel oniguruma-devel libzip-devel"
        return 1
    fi

    if [[ "$php_restore_cache" == true ]]; then
        unset ac_cv_func_shutdown
    fi
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        log_info "Building PHP $version (this may take a while)..."
        if ! make -j"$(nproc)"; then
            log_error "Build failed"
            return 1
        fi
        
        log_info "Installing PHP $version..."
        make -j$(nproc) install
    else
        if ! run_with_progress "Building PHP $version" 80 make -j"$(nproc)"; then
            log_error "Build failed. See $PHPV_CACHE_DIR/build.log for details"
            return 1
        fi
        
        if ! run_with_progress "Installing PHP $version" 20 make -j$(nproc) install; then
            log_error "Installation failed. See $PHPV_CACHE_DIR/build.log for details"
            return 1
        fi
    fi
    
    # Create basic php.ini
    mkdir -p "$install_dir/etc/conf.d"
    
    # Find the actual extension directory (future-proof approach)
    local ext_dir
    if [[ -d "$install_dir/lib/php/extensions" ]]; then
        ext_dir=$(find "$install_dir/lib/php/extensions" -maxdepth 1 -type d -name "no-debug-non-zts-*" | head -n1)
        if [[ -z "$ext_dir" ]]; then
            # Fallback to default if no directory found
            ext_dir="$install_dir/lib/php/extensions"
        fi
    else
        # Fallback if extensions directory doesn't exist
        ext_dir="$install_dir/lib/php/extensions"
    fi
    
    cat > "$install_dir/etc/php.ini" << EOF
; Basic PHP configuration
memory_limit = 256M
max_execution_time = 30
upload_max_filesize = 64M
post_max_size = 64M
date.timezone = UTC

; Extensions
extension_dir = "$ext_dir"
EOF

    if version_supports_opcache "$version"; then
        cat >> "$install_dir/etc/php.ini" << 'EOF'

; OPcache
zend_extension=opcache
opcache.enable=1
opcache.memory_consumption=128
opcache.interned_strings_buffer=8
opcache.max_accelerated_files=4000
opcache.revalidate_freq=2
opcache.fast_shutdown=1
EOF
    fi
    
    # Clean up build directory
    rm -rf "$build_dir"
    
    log_success "PHP $version installed successfully"
}

# Switch to a specific PHP version
use_php_version() {
    local input_version="$1"
    
    if [[ -z "$input_version" ]]; then
        log_error "Please specify a version"
        return 1
    fi
    
    # Resolve the actual version to use (e.g., "7.4" -> "7.4.33")
    local version
    version=$(resolve_installed_version "$input_version")
    
    if [[ -z "$version" ]]; then
        log_error "PHP $input_version is not installed"
        log_info "Available versions:"
        get_installed_versions | sed 's/^/  /'
        return 1
    fi
    
    # If we resolved to a different version, inform the user
    if [[ "$version" != "$input_version" ]]; then
        log_info "Using $version (matched from '$input_version')"
    fi
    
    set_current_version "$version"
    log_success "Now using PHP $version"
    
    # Show current PHP version
    show_current_version
}

# Show current PHP version
show_current_version() {
    local current_version
    current_version=$(get_current_version)
    
    if [[ "$current_version" == "system" ]]; then
        # When checking for system PHP, exclude phpv-managed versions from PATH
        local clean_path
        clean_path=$(echo "$PATH" | tr ':' '\n' | grep -v "$PHPV_VERSIONS_DIR" | tr '\n' ':')
        clean_path=${clean_path%:}
        
        if PATH="$clean_path" command -v php &> /dev/null; then
            local system_version
            system_version=$(PATH="$clean_path" php -v | head -n1 | cut -d' ' -f2)
            echo "Current: system (PHP $system_version)"
        else
            echo "Current: system (PHP not found in PATH)"
        fi
    else
        local php_path="$PHPV_VERSIONS_DIR/$current_version/bin/php"
        if [[ -x "$php_path" ]]; then
            local previous_ld="${LD_LIBRARY_PATH:-}"
            local lib_joined
            lib_joined=$(get_lib_paths_for_version "$current_version")

            if [[ -n "$lib_joined" ]]; then
                if [[ -n "$previous_ld" ]]; then
                    export LD_LIBRARY_PATH="$lib_joined:$previous_ld"
                else
                    export LD_LIBRARY_PATH="$lib_joined"
                fi
            fi

            local version_info
            if version_info=$("$php_path" -v | head -n1 2>/dev/null); then
                echo "Current: $current_version ($version_info)"
            else
                echo "Current: $current_version (failed to query version)"
            fi

            if [[ -n "$previous_ld" ]]; then
                export LD_LIBRARY_PATH="$previous_ld"
            else
                unset LD_LIBRARY_PATH
            fi
        else
            echo "Current: $current_version (invalid installation)"
        fi
    fi
}

# List all versions
list_versions() {
    local current_version
    current_version=$(get_current_version)
    
    echo "Installed versions:"
    while IFS= read -r version; do
        if [[ "$version" == "$current_version" ]]; then
            echo -e "  ${GREEN}* $version${NC}"
        else
            echo "    $version"
        fi
    done < <(get_installed_versions)
}

# List available versions for download
list_available() {
    local filter="${1:-}"
    echo "Available versions for download:"
    if [[ -z "$filter" ]]; then
        get_available_versions | sed 's/^/  /'
    else
        # Add dot to filter if it doesn't end with one
        local filter_pattern="$filter"
        if [[ "$filter_pattern" != *"." ]]; then
            filter_pattern="$filter_pattern."
        fi
        get_available_versions | grep "^$filter_pattern" | sed 's/^/  /'
    fi
}

# Uninstall a PHP version
uninstall_php_version() {
    local input_version="$1"
    
    if [[ -z "$input_version" ]]; then
        log_error "Please specify a version to uninstall"
        return 1
    fi
    
    if [[ "$input_version" == "system" ]]; then
        log_error "Cannot uninstall system PHP"
        return 1
    fi
    
    # Resolve the actual version to uninstall (e.g., "7.4" -> "7.4.33")
    local version
    version=$(resolve_installed_version "$input_version")
    
    if [[ -z "$version" ]]; then
        log_error "PHP $input_version is not installed"
        return 1
    fi
    
    # If we resolved to a different version, inform the user
    if [[ "$version" != "$input_version" ]]; then
        log_info "Uninstalling $version (matched from '$input_version')"
    fi
    
    local current_version
    current_version=$(get_current_version)
    
    if [[ "$version" == "$current_version" ]]; then
        log_warning "Currently using PHP $version, switching to system"
        use_php_version "system"
    fi
    
    log_info "Uninstalling PHP $version..."
    local llvm_version_file="$PHPV_VERSIONS_DIR/$version/.llvm_version"
    local llvm_version=""
    if [[ -f "$llvm_version_file" ]]; then
        llvm_version=$(cat "$llvm_version_file" 2>/dev/null || true)
    fi

    rm -rf "$PHPV_VERSIONS_DIR/$version"

    if [[ -n "$llvm_version" ]]; then
        local version_deps_dir
        if version_deps_dir=$(get_deps_dir_for_version "$version" "$llvm_version"); then
            if [[ -d "$version_deps_dir" ]]; then
                log_info "Removing isolated dependencies for PHP $version..."
                rm -rf "$version_deps_dir"
            fi
        fi
    fi
    log_success "PHP $version uninstalled"
}

# Get PHP binary path
get_php_path() {
    local current_version
    current_version=$(get_current_version)
    
    if [[ "$current_version" == "system" ]]; then
        command -v php 2>/dev/null || echo ""
    else
        local php_path="$PHPV_VERSIONS_DIR/$current_version/bin/php"
        if [[ -x "$php_path" ]]; then
            echo "$php_path"
        else
            echo ""
        fi
    fi
}

get_lib_paths_for_version() {
    local version="$1"
    local -a lib_paths=()

    if [[ -z "$version" || "$version" == "system" ]]; then
        return 0
    fi

    local version_dir="$PHPV_VERSIONS_DIR/$version"
    local llvm_version_file="$version_dir/.llvm_version"
    local deps_dir=""

    if [[ -f "$llvm_version_file" ]]; then
        local llvm_version
        llvm_version=$(cat "$llvm_version_file" 2>/dev/null || true)
        if [[ -n "$llvm_version" ]]; then
            deps_dir=$(get_deps_dir_for_version "$version" "$llvm_version" 2>/dev/null || true)
        fi
    fi

    if [[ -z "$deps_dir" || ! -d "$deps_dir" ]]; then
        if [[ -d "$PHPV_DEPS_BASE_DIR/$version" ]]; then
            deps_dir="$PHPV_DEPS_BASE_DIR/$version"
        fi
    fi

    if [[ -z "$deps_dir" || ! -d "$deps_dir" ]]; then
        return 0
    fi

    [[ -d "$deps_dir/lib" ]] && lib_paths+=("$deps_dir/lib")
    [[ -d "$deps_dir/lib64" ]] && lib_paths+=("$deps_dir/lib64")

    if (( ${#lib_paths[@]} == 0 )); then
        return 0
    fi

    local IFS=':'
    printf '%s' "${lib_paths[*]}"
}

print_environment_overrides() {
    local current_version
    current_version=$(get_current_version)

    local path_prefix=""
    local ld_prefix=""
    local ld_root=""
    local env_mode="system"

    if [[ -n "$current_version" && "$current_version" != "system" ]]; then
        local version_bin_dir="$PHPV_VERSIONS_DIR/$current_version/bin"
        if [[ -d "$version_bin_dir" ]]; then
            path_prefix="$version_bin_dir"
        fi
        ld_prefix=$(get_lib_paths_for_version "$current_version")
        ld_root="$PHPV_DEPS_BASE_DIR"
        env_mode="$current_version"
    fi

    printf 'PATH_PREFIX=%s\n' "$path_prefix"
    printf 'LD_LIBRARY_PATH_PREFIX=%s\n' "$ld_prefix"
    printf 'LD_LIBRARY_PATH_ROOT=%s\n' "$ld_root"
    printf 'ENV_MODE=%s\n' "$env_mode"
}

# Execute PHP with current version
exec_php() {
    local php_path
    php_path=$(get_php_path)
    
    if [[ -z "$php_path" ]]; then
        log_error "PHP is not available"
        return 1
    fi

    local current_version
    current_version=$(get_current_version)
    if [[ -n "$current_version" && "$current_version" != "system" ]]; then
        local lib_joined
        lib_joined=$(get_lib_paths_for_version "$current_version")
        if [[ -n "$lib_joined" ]]; then
            if [[ -n "${LD_LIBRARY_PATH:-}" ]]; then
                export LD_LIBRARY_PATH="$lib_joined:$LD_LIBRARY_PATH"
            else
                export LD_LIBRARY_PATH="$lib_joined"
            fi
        fi
    fi
    
    exec "$php_path" "$@"
}