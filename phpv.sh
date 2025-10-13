#!/usr/bin/env bash

# PHPV - PHP Version Manager
# Similar to pyenv and nvm but for PHP
# Manages multiple PHP versions in user space

set -e

# Configuration
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
PHPV_VERSIONS_DIR="$PHPV_ROOT/versions"
PHPV_CACHE_DIR="$PHPV_ROOT/cache"
PHPV_CURRENT_FILE="$PHPV_ROOT/version"
PHPV_DEPS_DIR="$PHPV_ROOT/deps"
PHPV_DEPS_BASE_DIR="$PHPV_DEPS_DIR"
PHPV_LLVM_VERSION="${PHPV_LLVM_VERSION:-17.0.6}"
PHPV_DEFAULT_VERSION="system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

prepend_path() {
    local dir="$1"
    case ":$PATH:" in
        *":$dir:"*) ;;
        *) PATH="$dir:$PATH" ;;
    esac
}

append_unique() {
    local -n __phpv_target_array="$1"
    local __phpv_value="$2"
    local __phpv_existing

    [[ -z "$__phpv_value" ]] && return

    for __phpv_existing in "${__phpv_target_array[@]}"; do
        if [[ "$__phpv_existing" == "$__phpv_value" ]]; then
            return
        fi
    done

    __phpv_target_array+=("$__phpv_value")
}

normalize_mysql_config() {
    local config_path="$1"

    [[ -f "$config_path" ]] || return 0

    if grep -q 'libs="$libs -l "' "$config_path"; then
        sed -i \
            -e 's/libs="\$libs -l "/libs="$libs -lmysqlclient -lpthread -lz -lm -lssl -lcrypto"/' \
            -e 's/embedded_libs="\$embedded_libs -l "/embedded_libs="$embedded_libs -lmysqlclient"/' \
            "$config_path"
    fi

    chmod +x "$config_path"
}

# Download helper that uses system libraries (not custom built ones)
safe_download() {
    local url="$1"
    local output="$2"
    
    # Completely isolate from custom environment
    local old_ld_library_path="$LD_LIBRARY_PATH"
    local old_path="$PATH"
    
    unset LD_LIBRARY_PATH
    export PATH="/usr/local/bin:/usr/bin:/bin"
    
    local result=1
    
    # Try wget first (more reliable)
    if command -v wget &> /dev/null; then
        wget -q "$url" -O "$output"
        result=$?
    fi
    
    # If wget failed or not available, try curl
    if [[ $result -ne 0 ]] && command -v curl &> /dev/null; then
        curl -fsSL "$url" -o "$output" 2>/dev/null
        result=$?
    fi
    
    if [[ $result -ne 0 ]]; then
        log_error "Failed to download $url"
    fi
    
    # Restore environment
    export LD_LIBRARY_PATH="$old_ld_library_path"
    export PATH="$old_path"
    return $result
}

# Initialize phpv directory structure
init_phpv() {
    mkdir -p "$PHPV_VERSIONS_DIR"
    mkdir -p "$PHPV_CACHE_DIR"
    mkdir -p "$PHPV_DEPS_DIR"
    
    if [[ ! -f "$PHPV_CURRENT_FILE" ]]; then
        echo "$PHPV_DEFAULT_VERSION" > "$PHPV_CURRENT_FILE"
    fi
}

get_deps_dir_for_version() {
    local version="$1"
    local llvm_version="$2"
    if [[ -z "$version" || "$version" == "system" || -z "$llvm_version" ]]; then
        return 1
    fi

    printf '%s\n' "$PHPV_DEPS_BASE_DIR/$llvm_version/$version"
}

# Get current PHP version
get_current_version() {
    if [[ -f "$PHPV_CURRENT_FILE" ]]; then
        cat "$PHPV_CURRENT_FILE"
    else
        echo "$PHPV_DEFAULT_VERSION"
    fi
}

# Set current PHP version
set_current_version() {
    local version="$1"
    echo "$version" > "$PHPV_CURRENT_FILE"
}

# Check if version is installed
is_version_installed() {
    local version="$1"
    [[ "$version" == "system" ]] || [[ -x "$PHPV_VERSIONS_DIR/$version/bin/php" ]]
}

# Get available PHP versions for download
get_available_versions() {
    # This would typically fetch from PHP's release API
    # For now, we'll provide a hardcoded list of common versions
    cat << 'EOF'
8.3.12
8.3.11
8.3.10
8.3.9
8.3.8
8.3.7
8.3.6
8.3.4
8.3.3
8.3.2
8.3.1
8.3.0
8.2.24
8.2.23
8.2.22
8.2.21
8.2.20
8.2.19
8.2.18
8.2.17
8.2.16
8.2.15
8.2.14
8.2.13
8.2.12
8.2.11
8.2.10
8.2.9
8.2.8
8.2.7
8.2.6
8.2.5
8.2.4
8.2.3
8.2.2
8.2.1
8.2.0
8.1.29
8.1.28
8.1.27
8.1.26
8.1.25
8.1.24
8.1.23
8.1.22
8.1.21
8.1.20
8.1.19
8.1.18
8.1.17
8.1.16
8.1.15
8.1.14
8.1.13
8.1.12
8.1.11
8.1.10
8.1.9
8.1.8
8.1.7
8.1.6
8.1.5
8.1.4
8.1.3
8.1.2
8.1.1
8.1.0
8.0.30
8.0.29
8.0.28
8.0.27
8.0.26
8.0.25
8.0.24
8.0.23
8.0.22
8.0.21
8.0.20
8.0.19
8.0.18
8.0.17
8.0.16
8.0.15
8.0.14
8.0.13
8.0.12
8.0.11
8.0.10
8.0.9
8.0.8
8.0.7
8.0.6
8.0.5
8.0.3
8.0.2
8.0.1
8.0.0
7.4.33
7.4.32
7.4.30
7.4.29
7.4.28
7.4.27
7.4.26
7.4.25
7.4.24
7.4.23
7.4.22
7.4.21
7.4.20
7.4.19
7.4.18
7.4.16
7.4.15
7.4.14
7.4.13
7.4.12
7.4.11
7.4.10
7.4.9
7.4.8
7.4.7
7.4.6
7.4.5
7.4.4
7.4.3
7.4.2
7.4.1
7.4.0
7.3.33
7.3.32
7.3.31
7.3.30
7.3.29
7.3.28
7.3.27
7.3.26
7.3.25
7.3.24
7.3.23
7.3.22
7.3.21
7.3.20
7.3.19
7.3.18
7.3.17
7.3.16
7.3.15
7.3.14
7.3.13
7.3.12
7.3.11
7.3.10
7.3.9
7.3.8
7.3.7
7.3.6
7.3.5
7.3.4
7.3.3
7.3.2
7.3.1
7.3.0
7.2.34
7.2.33
7.2.32
7.2.31
7.2.30
7.2.29
7.2.28
7.2.27
7.2.26
7.2.25
7.2.24
7.2.23
7.2.22
7.2.21
7.2.20
7.2.19
7.2.18
7.2.17
7.2.16
7.2.15
7.2.14
7.2.13
7.2.12
7.2.11
7.2.10
7.2.9
7.2.8
7.2.7
7.2.6
7.2.5
7.2.4
7.2.3
7.2.2
7.2.1
7.2.0
7.1.33
7.1.32
7.1.31
7.1.30
7.1.29
7.1.28
7.1.27
7.1.26
7.1.25
7.1.24
7.1.23
7.1.22
7.1.21
7.1.20
7.1.19
7.1.18
7.1.17
7.1.16
7.1.15
7.1.14
7.1.13
7.1.12
7.1.11
7.1.10
7.1.9
7.1.8
7.1.7
7.1.6
7.1.5
7.1.4
7.1.3
7.1.2
7.1.1
7.1.0
7.0.33
7.0.32
7.0.31
7.0.30
7.0.29
7.0.28
7.0.27
7.0.26
7.0.25
7.0.24
7.0.23
7.0.22
7.0.21
7.0.20
7.0.19
7.0.18
7.0.17
7.0.16
7.0.15
7.0.14
7.0.13
7.0.12
7.0.11
7.0.10
7.0.9
7.0.8
7.0.7
7.0.6
7.0.5
7.0.4
7.0.3
7.0.2
7.0.1
7.0.0
5.6.40
5.6.39
5.6.38
5.6.37
5.6.36
5.6.35
5.6.34
5.6.33
5.6.32
5.6.31
5.6.30
5.6.29
5.6.28
5.6.27
5.6.26
5.6.25
5.6.24
5.6.23
5.6.22
5.6.21
5.6.20
5.6.19
5.6.18
5.6.17
5.6.16
5.6.15
5.6.14
5.6.13
5.6.12
5.6.11
5.6.10
5.6.9
5.6.8
5.6.7
5.6.6
5.6.5
5.6.4
5.6.3
5.6.2
5.6.1
5.6.0
5.5.38
5.5.37
5.5.36
5.5.35
5.5.34
5.5.33
5.5.32
5.5.31
5.5.30
5.5.29
5.5.28
5.5.27
5.5.26
5.5.25
5.5.24
5.5.23
5.5.22
5.5.21
5.5.20
5.5.19
5.5.18
5.5.17
5.5.16
5.5.15
5.5.14
5.5.13
5.5.12
5.5.11
5.5.10
5.5.9
5.5.8
5.5.7
5.5.6
5.5.5
5.5.4
5.5.3
5.5.2
5.5.1
5.5.0
5.4.45
5.4.44
5.4.43
5.4.42
5.4.41
5.4.40
5.4.39
5.4.38
5.4.37
5.4.36
5.4.35
5.4.34
5.4.33
5.4.32
5.4.31
5.4.30
5.4.29
5.4.28
5.4.27
5.4.26
5.4.25
5.4.24
5.4.23
5.4.22
5.4.21
5.4.20
5.4.19
5.4.18
5.4.17
5.4.16
5.4.15
5.4.14
5.4.13
5.4.12
5.4.11
5.4.10
5.4.9
5.4.8
5.4.7
5.4.6
5.4.5
5.4.4
5.4.3
5.4.2
5.4.1
5.4.0
5.3.29
5.3.28
5.3.27
5.3.26
5.3.25
5.3.24
5.3.23
5.3.22
5.3.21
5.3.20
5.3.19
5.3.18
5.3.17
5.3.16
5.3.15
5.3.14
5.3.13
5.3.12
5.3.11
5.3.10
5.3.9
5.3.8
5.3.7
5.3.6
5.3.5
5.3.4
5.3.3
5.3.2
5.3.1
5.3.0
5.2.17
5.2.16
5.2.15
5.2.14
5.2.13
5.2.12
5.2.11
5.2.10
5.2.9
5.2.8
5.2.7
5.2.6
5.2.5
5.2.4
5.2.3
5.2.2
5.2.1
5.2.0
5.1.6
5.1.5
5.1.4
5.1.3
5.1.2
5.1.1
5.1.0
5.0.5
5.0.4
5.0.3
5.0.2
5.0.1
5.0.0
4.4.9
4.4.8
4.4.7
4.4.6
4.4.5
4.4.4
4.4.3
4.4.2
4.4.1
4.4.0
4.3.11
4.3.10
4.3.9
4.3.8
4.3.7
4.3.6
4.3.5
4.3.4
4.3.3
4.3.2
4.3.1
4.3.0
4.2.3
4.2.2
4.2.1
4.2.0
4.1.2
4.1.1
4.1.0
4.0.6
4.0.5
4.0.4
4.0.3
4.0.2
4.0.1
4.0.0
EOF
}

# Determine which LLVM toolchain version should be used for a given PHP version.
# Users can override the defaults by exporting PHPV_LLVM_VERSION_MAP with entries
# like "7.4.*=16.0.6,8.0.*=17.0.6". The first matching pattern wins.
resolve_llvm_version_for_php() {
    local php_version="$1"
    local default_version="${PHPV_LLVM_VERSION:-17.0.6}"

    if [[ -z "$php_version" ]]; then
        echo "$default_version"
        return
    fi

    if [[ -n "$PHPV_LLVM_VERSION_MAP" ]]; then
        local -a __phpv_llvm_entries=()
        IFS=',' read -ra __phpv_llvm_entries <<< "$PHPV_LLVM_VERSION_MAP"
        for entry in "${__phpv_llvm_entries[@]}"; do
            entry="${entry//[[:space:]]/}"
            [[ -z "$entry" || "$entry" != *"="* ]] && continue

            local pattern="${entry%%=*}"
            local llvm_version="${entry#*=}"

            [[ -z "$pattern" || -z "$llvm_version" ]] && continue

            local glob="$pattern"
            case "$glob" in
                *\**)
                    :
                    ;;
                *.*)
                    glob="${glob}*"
                    ;;
                *)
                    glob="${glob}.*"
                    ;;
            esac

            case "$php_version" in
                $glob)
                    echo "$llvm_version"
                    return
                    ;;
            esac
        done
    fi

    if [[ "$php_version" == 7.* ]]; then
        echo "${PHPV_LLVM_VERSION_PHP7:-16.0.4}"
        return
    fi

    if [[ "$php_version" == 5.* ]]; then
        echo "${PHPV_LLVM_VERSION_PHP5:-15.0.6}"
        return
    fi

    echo "$default_version"
}

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
    ./configure --prefix="$PHPV_DEPS_DIR" --without-python
    make -j$(nproc)
    make install
}

# Install zlib from source
install_zlib_from_source() {
    local version="1.3.1"
    local url="https://zlib.net/zlib-$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/zlib-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/zlib-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"

    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1

    ./configure --prefix="$PHPV_DEPS_DIR" --shared
    make -j$(nproc)
    make install
}

# Install OpenSSL from source
install_openssl_from_source() {
    local php_version="${1:-}"
    local version="3.0.13"
    
    # Use OpenSSL 1.0.x for PHP 5.x versions for compatibility
    if [[ -n "$php_version" && "$php_version" == 5.* ]]; then
        version="1.0.1u"
    # Use OpenSSL 1.1.x for PHP 7.x versions for compatibility
    elif [[ -n "$php_version" && "$php_version" == 7.* ]]; then
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
    ./config --prefix="$PHPV_DEPS_DIR" --openssldir="$PHPV_DEPS_DIR/ssl" shared
    make -j$(nproc)
    make install
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
    local cache_file="$PHPV_CACHE_DIR/onig-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/onig-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    ensure_llvm_toolchain || return 1
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install libpng from source
install_libpng_from_source() {
    local version="1.6.40"
    local url="https://download.sourceforge.net/libpng/libpng-$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/libpng-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/libpng-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
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
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install freetype from source
install_freetype_from_source() {
    local version="2.13.2"
    local url="https://download.savannah.gnu.org/releases/freetype/freetype-$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/freetype-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/freetype-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
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
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
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
    eval "$configure_cmd"
    if [[ "$restore_select_cache" == true ]]; then
        unset ac_cv_func_select ac_cv_func_socket
    fi
    make -j$(nproc)
    make install
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

    if ! ./bootstrap --prefix="$PHPV_DEPS_DIR" --parallel="$jobs" -- -DCMake_ENABLE_DEBUGGER=OFF -DBUILD_TESTING=OFF; then
        cd "$old_cwd"
        return 1
    fi

    if ! make -j"$jobs"; then
        cd "$old_cwd"
        return 1
    fi

    if ! make install; then
        cd "$old_cwd"
        return 1
    fi

    cd "$old_cwd"
    rm -rf "$source_dir"
}

# Install libzip from source
install_libzip_from_source() {
    local version="1.10.1"
    local url="https://libzip.org/download/libzip-$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/libzip-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/libzip-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    
    # libzip uses CMake, not autotools
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

    "$cmake_cmd" \
        -DCMAKE_INSTALL_PREFIX="$PHPV_DEPS_DIR" \
        -DCMAKE_PREFIX_PATH="$PHPV_DEPS_DIR" ..
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
    make install
}

# Install unixODBC from source
install_unixodbc_from_source() {
    local version="2.3.12"
    local url="https://www.unixodbc.org/unixODBC-$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/unixODBC-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/unixODBC-$version"
    
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
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
    ./configure --prefix="$PHPV_DEPS_DIR" \
                --with-unixodbc="$PHPV_DEPS_DIR" \
                --with-openssl="$PHPV_DEPS_DIR" \
                --enable-sybase-compat \
                --disable-dependency-tracking
    make -j$(nproc)
    make install
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

    "$cmake_cmd" .. \
    -DCMAKE_INSTALL_PREFIX="$PHPV_DEPS_DIR" \
    -DCMAKE_PREFIX_PATH="$PHPV_DEPS_DIR" \
        -DWITH_UNIT_TESTS=OFF \
        -DWITH_SSL=ON \
        -DOPENSSL_ROOT_DIR="$PHPV_DEPS_DIR" \
        -DZLIB_ROOT="$PHPV_DEPS_DIR" \
        -DWITH_EXTERNAL_ZLIB=ON \
        -DWITH_CURL=OFF \
        -DCMAKE_POLICY_VERSION=3.5 \
        -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
        -DCMAKE_BUILD_TYPE=Release || {
        export PATH="$old_path"
        export LD_LIBRARY_PATH="$old_ld_library_path"
        export PKG_CONFIG_PATH="$old_pkg_config"
        export LDFLAGS="$old_ldflags"
        export CPPFLAGS="$old_cppflags"
        cd "$old_cwd"
        return 1
    }

    make -j$(nproc) || {
        export PATH="$old_path"
        export LD_LIBRARY_PATH="$old_ld_library_path"
        export PKG_CONFIG_PATH="$old_pkg_config"
        export LDFLAGS="$old_ldflags"
        export CPPFLAGS="$old_cppflags"
        cd "$old_cwd"
        return 1
    }

    make install || {
        export PATH="$old_path"
        export LD_LIBRARY_PATH="$old_ld_library_path"
        export PKG_CONFIG_PATH="$old_pkg_config"
        export LDFLAGS="$old_ldflags"
        export CPPFLAGS="$old_cppflags"
        cd "$old_cwd"
        return 1
    }

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
    local mysql_url="https://downloads.mysql.com/archives/mysql-5.0/mysql-${mysql_version}.tar.gz"
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
        CC="$PHPV_DEPS_DIR/llvm-17.0.6/bin/clang" \
        CXX="$PHPV_DEPS_DIR/llvm-17.0.6/bin/clang++" \
        CFLAGS="-I$PHPV_DEPS_DIR/include -Wno-implicit-int -Wno-implicit-function-declaration" \
        CXXFLAGS="-I$PHPV_DEPS_DIR/include" \
        LDFLAGS="-L$PHPV_DEPS_DIR/lib -L$PHPV_DEPS_DIR/lib64" || {
        log_error "MySQL configure failed"
        cd "$old_cwd" 2>/dev/null || true
        rm -rf "$mysql_extract_dir"
        return 1
    }

    make -j$(nproc) || {
        log_error "MySQL build failed"
        cd "$old_cwd" 2>/dev/null || true
        rm -rf "$mysql_extract_dir"
        return 1
    }

    make install || {
        log_error "MySQL install failed"
        cd "$old_cwd" 2>/dev/null || true
        rm -rf "$mysql_extract_dir"
        return 1
    }

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

    make install || {
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

# Get installed versions
get_installed_versions() {
    echo "system"
    if [[ -d "$PHPV_VERSIONS_DIR" ]]; then
        for dir in "$PHPV_VERSIONS_DIR"/*/; do
            if [[ -d "$dir" ]]; then
                local version
                version=$(basename "$dir")
                if [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] && [[ -x "$dir/bin/php" ]]; then
                    echo "$version"
                fi
            fi
        done | sort -V
    fi
}

resolve_llvm_asset_url() {
    local version="$1"
    local machine="$2"
    local target_suffix="$3"

    local api_url="https://api.github.com/repos/llvm/llvm-project/releases/tags/llvmorg-${version}"
    local release_json

    if command -v curl &> /dev/null; then
        if ! release_json=$(curl -fsSL "$api_url"); then
            return 1
        fi
    elif command -v wget &> /dev/null; then
        if ! release_json=$(wget -qO- "$api_url"); then
            return 1
        fi
    else
        return 1
    fi

    if [[ "$release_json" == *"API rate limit exceeded"* ]]; then
        log_warning "GitHub API rate limit exceeded while fetching LLVM $version metadata"
        return 1
    fi

    local urls
    urls=$(echo "$release_json" | grep -o '"browser_download_url": *"[^\"]*"' | sed -E 's/.*"browser_download_url": *"([^\"]*)"/\1/' | sed 's/%2B/+/g')

    if [[ -z "$urls" ]]; then
        return 1
    fi

    local arch_patterns=()
    case "$machine" in
        x86_64)
            arch_patterns=("x86_64" "x86-64" "amd64")
            ;;
        aarch64|arm64)
            arch_patterns=("aarch64" "arm64")
            ;;
        ppc64le)
            arch_patterns=("ppc64le")
            ;;
        *)
            arch_patterns=("$machine")
            ;;
    esac

    local chosen=""
    while IFS= read -r url; do
        [[ -z "$url" ]] && continue
        [[ "$url" != *"clang+llvm-${version}"* ]] && continue
        [[ "$url" != *.tar.xz ]] && continue

        if [[ -n "$target_suffix" ]]; then
            if [[ "$url" == *"clang+llvm-${version}-${target_suffix}.tar.xz" ]]; then
                chosen="$url"
                break
            fi
            continue
        fi

        local matched=0
        for pattern in "${arch_patterns[@]}"; do
            if [[ "$url" == *"$pattern"* ]]; then
                matched=1
                break
            fi
        done
        [[ $matched -eq 0 ]] && continue
        [[ "$url" != *linux* ]] && continue
        chosen="$url"
        break
    done <<< "$urls"

    if [[ -z "$chosen" ]]; then
        return 1
    fi

    printf '%s\n' "$chosen"
}

# Install LLVM/Clang toolchain without relying on system packages
install_llvm_toolchain() {
    local requested_version="${1:-$PHPV_LLVM_VERSION}"
    local machine
    machine=$(uname -m)
    local os
    os=$(uname -s)

    if [[ "$os" != "Linux" ]]; then
        log_error "Automatic LLVM installation currently supports Linux only"
        return 1
    fi

    local install_dir
    local selected_version=""
    local asset_url="${PHPV_LLVM_ARCHIVE_URL:-}"

    local candidates=()
    append_unique candidates "$requested_version"

    if [[ -z "$PHPV_LLVM_ARCHIVE_URL" ]]; then
        if [[ "$requested_version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
            local requested_major="${BASH_REMATCH[1]}"
            local requested_minor="${BASH_REMATCH[2]}"
            local requested_patch="${BASH_REMATCH[3]}"

            local patch_candidate=$((requested_patch - 1))
            while (( patch_candidate >= 0 )); do
                append_unique candidates "${requested_major}.${requested_minor}.${patch_candidate}"
                ((patch_candidate--))
            done
        fi

        local fallback_versions=("17.0.6" "17.0.5" "16.0.6" "16.0.0" "15.0.7")
        for v in "${fallback_versions[@]}"; do
            append_unique candidates "$v"
        done
    fi

    for candidate_version in "${candidates[@]}"; do
        install_dir="$PHPV_DEPS_DIR/llvm-$candidate_version"
        if [[ -x "$install_dir/bin/clang" ]]; then
            selected_version="$candidate_version"
            asset_url=""
            break
        fi

        if [[ -n "$PHPV_LLVM_ARCHIVE_URL" && "$candidate_version" != "$requested_version" ]]; then
            continue
        fi

        local resolved_url
        if [[ -n "$PHPV_LLVM_ARCHIVE_URL" ]]; then
            resolved_url="$PHPV_LLVM_ARCHIVE_URL"
        else
            if ! resolved_url=$(resolve_llvm_asset_url "$candidate_version" "$machine" "$PHPV_LLVM_TARGET_SUFFIX"); then
                log_warning "No compatible LLVM archive found for $candidate_version ($machine)"
                continue
            fi
        fi

        selected_version="$candidate_version"
        asset_url="$resolved_url"
        break
    done

    if [[ -z "$selected_version" ]]; then
        log_error "Could not locate a suitable LLVM archive. Set PHPV_LLVM_ARCHIVE_URL to a downloadable asset."
        return 1
    fi

    install_dir="$PHPV_DEPS_DIR/llvm-$selected_version"

    if [[ -z "$asset_url" ]]; then
        if [[ "$selected_version" != "$requested_version" ]]; then
            log_warning "Using LLVM $selected_version because binaries for $requested_version were not found."
        fi
        PHPV_ACTIVE_LLVM_VERSION="$selected_version"
        if [[ "$selected_version" != "$requested_version" ]]; then
            log_info "Using existing LLVM $selected_version installation"
        fi
        return 0
    fi

    if [[ "$selected_version" != "$requested_version" ]]; then
        log_warning "Falling back to LLVM $selected_version because binaries for $requested_version were not found."
    fi

    log_info "Installing LLVM/Clang $selected_version..."

    local archive
    archive="${asset_url##*/}"
    local cache_file="$PHPV_CACHE_DIR/$archive"
    log_info "Selected LLVM asset: $archive"

    if [[ ! -f "$cache_file" ]]; then
        log_info "Downloading $archive"
        if command -v curl &> /dev/null; then
            if ! curl -fsSL "$asset_url" -o "$cache_file"; then
                rm -f "$cache_file"
                log_error "Failed to download LLVM from $asset_url"
                return 1
            fi
        else
            if ! wget -q "$asset_url" -O "$cache_file"; then
                rm -f "$cache_file"
                log_error "Failed to download LLVM from $asset_url"
                return 1
            fi
        fi
    fi

    local extract_dir="$PHPV_CACHE_DIR/llvm-$selected_version-extract"
    rm -rf "$extract_dir"
    mkdir -p "$extract_dir"
    if ! tar -xJf "$cache_file" -C "$extract_dir"; then
        rm -rf "$extract_dir"
        rm -f "$cache_file"
        log_error "Failed to unpack LLVM archive"
        return 1
    fi

    local unpacked
    unpacked=$(find "$extract_dir" -maxdepth 1 -mindepth 1 -type d -name "clang+llvm-${selected_version}*" | head -n1)
    if [[ -z "$unpacked" ]]; then
        log_error "Failed to locate LLVM directory after extraction"
        return 1
    fi

    rm -rf "$install_dir"
    mv "$unpacked" "$install_dir"
    rm -rf "$extract_dir"
    PHPV_ACTIVE_LLVM_VERSION="$selected_version"
}

ensure_llvm_toolchain() {
    local requested_version="${1:-$PHPV_LLVM_VERSION}"

    install_llvm_toolchain "$requested_version" || return 1

    local active_version="${PHPV_ACTIVE_LLVM_VERSION:-$requested_version}"
    local llvm_dir="$PHPV_DEPS_DIR/llvm-$active_version"
    local clang_path="$llvm_dir/bin/clang"
    local clangxx_path="$llvm_dir/bin/clang++"

    if [[ ! -x "$clang_path" || ! -x "$clangxx_path" ]]; then
        log_error "LLVM toolchain installation failed"
        return 1
    fi

    prepend_path "$llvm_dir/bin"
    export CC="$clang_path"
    export CXX="$clangxx_path"
    export AR="$llvm_dir/bin/llvm-ar"
    export NM="$llvm_dir/bin/llvm-nm"
    export RANLIB="$llvm_dir/bin/llvm-ranlib"
    export LLVM_HOME="$llvm_dir"
}

# Download and compile PHP
resolve_latest_version() {
    local input_version="$1"
    
    if [[ -z "$input_version" ]]; then
        return 1
    fi
    
    # If it's already a full version (x.y.z), return as-is
    if [[ "$input_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "$input_version"
        return 0
    fi
    
    # Build filter pattern
    local filter_pattern="$input_version"
    if [[ "$filter_pattern" != *"." ]]; then
        filter_pattern="$filter_pattern."
    fi
    
    # Get matching versions and find the latest one
    local latest_version
    latest_version=$(get_available_versions | grep "^$filter_pattern" | sort -V | tail -n1)
    
    if [[ -z "$latest_version" ]]; then
        return 1
    fi
    
    echo "$latest_version"
}

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

    ensure_llvm_toolchain "$resolved_llvm" || return 1

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
    export LD_LIBRARY_PATH="$PHPV_DEPS_DIR/lib:$PHPV_DEPS_DIR/lib64:$LD_LIBRARY_PATH"
    
    # For PHP 5.x, DSA_get_default_method is in libcrypto, not libssl
    # Add libcrypto to LDFLAGS to make the function available during configure checks
    if [[ "$version" == 5.* ]]; then
        php_restore_env=true
        php_extra_ldflags="-lssl -lcrypto"
        export CFLAGS="-Wno-implicit-int -Wno-implicit-function-declaration -Wno-deprecated-declarations -Wno-deprecated-non-prototype -Wno-visibility -Wno-pointer-sign -fcommon $CFLAGS"
        export CXXFLAGS="-Wno-register $CXXFLAGS"
        export CPPFLAGS="-DHAVE_STDARG_PROTOTYPES=1 -D_BSD_SOURCE -D_DEFAULT_SOURCE -D_POSIX_C_SOURCE=200112L -D_XOPEN_SOURCE=600 $CPPFLAGS"
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
        --enable-opcache
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
    if [[ ${#configure_env[@]} -gt 0 ]]; then
        if env "${configure_env[@]}" ./configure "${configure_flags[@]}"; then
            configure_success=true
        fi
    else
        if ./configure "${configure_flags[@]}"; then
            configure_success=true
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
    
    log_info "Building PHP $version (this may take a while)..."
    if ! make -j"$(nproc)"; then
        log_error "Build failed"
        return 1
    fi
    
    log_info "Installing PHP $version..."
    make install
    
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

; OPcache
zend_extension=opcache
opcache.enable=1
opcache.memory_consumption=128
opcache.interned_strings_buffer=8
opcache.max_accelerated_files=4000
opcache.revalidate_freq=2
opcache.fast_shutdown=1
EOF
    
    # Clean up build directory
    rm -rf "$build_dir"
    
    log_success "PHP $version installed successfully"
}

# Switch to a specific PHP version
use_php_version() {
    local version="$1"
    
    if [[ -z "$version" ]]; then
        log_error "Please specify a version"
        return 1
    fi
    
    if ! is_version_installed "$version"; then
        log_error "PHP $version is not installed"
        log_info "Available versions:"
        get_installed_versions | sed 's/^/  /'
        return 1
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
        if command -v php &> /dev/null; then
            local system_version
            system_version=$(php -v | head -n1 | cut -d' ' -f2)
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
    local version="$1"
    
    if [[ -z "$version" ]]; then
        log_error "Please specify a version to uninstall"
        return 1
    fi
    
    if [[ "$version" == "system" ]]; then
        log_error "Cannot uninstall system PHP"
        return 1
    fi
    
    if ! is_version_installed "$version"; then
        log_error "PHP $version is not installed"
        return 1
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

    if [[ -n "$current_version" && "$current_version" != "system" ]]; then
        local version_bin_dir="$PHPV_VERSIONS_DIR/$current_version/bin"
        if [[ -d "$version_bin_dir" ]]; then
            path_prefix="$version_bin_dir"
        fi
        ld_prefix=$(get_lib_paths_for_version "$current_version")
    fi

    printf 'PATH_PREFIX=%s\n' "$path_prefix"
    printf 'LD_LIBRARY_PATH_PREFIX=%s\n' "$ld_prefix"
    printf 'LD_LIBRARY_PATH_ROOT=%s\n' "$PHPV_DEPS_BASE_DIR"
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

# Show help
show_help() {
    cat << 'EOF'
PHPV - PHP Version Manager

USAGE:
    phpv <command> [arguments]

COMMANDS:
    install <version>           Install a specific PHP version (supports partial versions: e.g., 8, 8.3)
    uninstall <version>         Uninstall a specific PHP version
    use <version>               Switch to a specific PHP version
    current                     Show the current PHP version
    list                        List installed PHP versions
    list-available [filter]     List available PHP versions for download (optional filter: e.g., 8, 8.3)
    exec <command>              Execute command with current PHP version
    which                       Show path to current PHP binary
    env                         Print environment overrides for current version
    help                        Show this help message

EXAMPLES:
    phpv install 8.3.12         # Install PHP 8.3.12
    phpv install 8.3            # Install latest 8.3.x version (8.3.12)
    phpv install 8              # Install latest 8.x.x version
    phpv use 8.3.12             # Switch to PHP 8.3.12
    phpv use system             # Switch to system PHP
    phpv current                # Show current version
    phpv list                   # List installed versions
    phpv list-available         # List all available versions
    phpv list-available 8       # List only 8.x versions
    phpv list-available 8.3     # List only 8.3.x versions
    phpv exec -v                # Run 'php -v' with current version
    phpv which                  # Show current PHP binary path

ENVIRONMENT VARIABLES:
    PHPV_ROOT    Root directory for phpv (default: ~/.phpv)
EOF
}

# Main command dispatcher
main() {
    init_phpv
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        "install")
            install_php_version "$1"
            ;;
        "uninstall")
            uninstall_php_version "$1"
            ;;
        "use")
            use_php_version "$1"
            ;;
        "current")
            show_current_version
            ;;
        "list")
            list_versions
            ;;
        "list-available")
            list_available "$@"
            ;;
        "exec")
            exec_php "$@"
            ;;
        "which")
            get_php_path
            ;;
        "env")
            print_environment_overrides
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            echo
            show_help
            exit 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
else
    # Script is being sourced - set up shell integration
    # Instead of exporting functions, we'll define them in the calling scope
    :
fi