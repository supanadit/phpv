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
    [[ "$version" == "system" ]] || [[ -d "$PHPV_VERSIONS_DIR/$version" ]]
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

    echo "$default_version"
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
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install OpenSSL from source
install_openssl_from_source() {
    local version="3.0.13"
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
    ./config --prefix="$PHPV_DEPS_DIR" --openssldir="$PHPV_DEPS_DIR/ssl"
    make -j$(nproc)
    make install
}

# Install libxml2 from source
install_libxml2_from_source() {
    local version="2.11.5"
    local url="https://download.gnome.org/sources/libxml2/2.11/libxml2-$version.tar.xz"
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

# Install oniguruma from source
install_oniguruma_from_source() {
    local version="6.9.9"
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
    local version="73.2"
    local url="https://github.com/unicode-org/icu/releases/download/release-$(echo $version | tr . -)/icu4c-$(echo $version | tr . _)-src.tgz"
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
    local version="8.5.0"
    local url="https://curl.se/download/curl-$version.tar.gz"
    local cache_file="$PHPV_CACHE_DIR/curl-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/curl-$version"
    
    # Download if not cached
    if [[ ! -f "$cache_file" ]]; then
        safe_download "$url" "$cache_file" || return 1
    fi
    
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    tar -xzf "$cache_file" --strip-components=1
    ./configure --prefix="$PHPV_DEPS_DIR" --with-openssl="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
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
    local old_ld_library_path="$LD_LIBRARY_PATH"
    local old_pkg_config_path="$PKG_CONFIG_PATH"
    local old_ldflags="$LDFLAGS"
    local old_cppflags="$CPPFLAGS"
    
    # Use only system paths for cmake
    export PATH="/usr/local/bin:/usr/bin:/bin"
    unset LD_LIBRARY_PATH
    unset PKG_CONFIG_PATH
    unset LDFLAGS
    unset CPPFLAGS
    
    cmake -DCMAKE_INSTALL_PREFIX="$PHPV_DEPS_DIR" ..
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

# Get installed versions
get_installed_versions() {
    echo "system"
    if [[ -d "$PHPV_VERSIONS_DIR" ]]; then
        find "$PHPV_VERSIONS_DIR" -maxdepth 1 -type d -exec basename {} \; | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V
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
install_php_version() {
    local version="$1"
    
    if [[ -z "$version" ]]; then
        log_error "Please specify a version to install"
        return 1
    fi
    
    local install_dir="$PHPV_VERSIONS_DIR/$version"
    local cache_file="$PHPV_CACHE_DIR/php-$version.tar.gz"
    
    if is_version_installed "$version"; then
        log_warning "PHP $version is already installed"
        return 0
    fi
    
    log_info "Installing PHP $version..."

    local resolved_llvm
    resolved_llvm=$(resolve_llvm_version_for_php "$version")
    [[ -z "$resolved_llvm" ]] && resolved_llvm="$PHPV_LLVM_VERSION"

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
    
    # Install required dependencies from source if not present
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libz.so" ]]; then
        log_info "Installing zlib from source..."
        install_zlib_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib64/libssl.so" ]] && [[ ! -f "$PHPV_DEPS_DIR/lib/libssl.so" ]]; then
        log_info "Installing OpenSSL from source..."
        install_openssl_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libxml2.so" ]]; then
        log_info "Installing libxml2 from source..."
        install_libxml2_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libonig.so" ]]; then
        log_info "Installing oniguruma from source..."
        install_oniguruma_from_source || return 1
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
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libicuuc.so" ]]; then
        log_info "Installing ICU from source..."
        install_icu_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libcurl.so" ]]; then
        log_info "Installing curl from source..."
        install_curl_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libzip.so" ]]; then
        log_info "Installing libzip from source..."
        install_libzip_from_source || return 1
    fi
    
    # Download PHP source if not cached
    if [[ ! -f "$cache_file" ]]; then
        log_info "Downloading PHP $version source..."
        local download_url="https://www.php.net/distributions/php-$version.tar.gz"
        safe_download "$download_url" "$cache_file" || return 1
    fi
    
    # Extract and build
    local build_dir="$PHPV_CACHE_DIR/php-$version-build"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    
    log_info "Extracting PHP $version..."
    tar -xzf "$cache_file" -C "$build_dir" --strip-components=1
    
    cd "$build_dir"
    
    log_info "Configuring PHP $version..."
    # Basic configuration - can be customized
    ./configure \
        --prefix="$install_dir" \
        --enable-cli \
        --enable-cgi \
        --enable-fpm \
        --with-config-file-path="$install_dir/etc" \
        --with-config-file-scan-dir="$install_dir/etc/conf.d" \
        --enable-mbstring \
        --enable-opcache \
        --with-openssl="$PHPV_DEPS_DIR" \
        --with-zlib="$PHPV_DEPS_DIR" \
        --with-libxml-dir="$PHPV_DEPS_DIR" \
        --with-onig="$PHPV_DEPS_DIR" \
        --with-libzip="$PHPV_DEPS_DIR" \
        --enable-bcmath \
        --enable-calendar \
        --enable-exif \
        --enable-ftp \
        --with-curl="$PHPV_DEPS_DIR" \
        --enable-gd \
        --with-png-dir="$PHPV_DEPS_DIR" \
        --with-jpeg-dir="$PHPV_DEPS_DIR" \
        --with-freetype-dir="$PHPV_DEPS_DIR" \
        --enable-intl \
        --with-icu-dir="$PHPV_DEPS_DIR" \
        --enable-soap \
        --enable-sockets \
        --with-mysqli \
        --with-pdo-mysql \
        --enable-pcntl \
        --enable-shmop \
        --enable-sysvmsg \
        --enable-sysvsem \
        --enable-sysvshm \
        2>/dev/null || {
        log_error "Configuration failed. You may need to install development packages:"
        log_info "Ubuntu/Debian: sudo apt-get install libxml2-dev libssl-dev libcurl4-openssl-dev libonig-dev libzip-dev"
        log_info "CentOS/RHEL: sudo yum install libxml2-devel openssl-devel curl-devel oniguruma-devel libzip-devel"
        return 1
    }
    
    log_info "Building PHP $version (this may take a while)..."
    make -j$(nproc) 2>/dev/null || {
        log_error "Build failed"
        return 1
    }
    
    log_info "Installing PHP $version..."
    make install 2>/dev/null
    
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
            local version_info
            version_info=$("$php_path" -v | head -n1)
            echo "Current: $current_version ($version_info)"
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
    echo "Available versions for download:"
    get_available_versions | sed 's/^/  /'
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
    rm -rf "$PHPV_VERSIONS_DIR/$version"
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

# Execute PHP with current version
exec_php() {
    local php_path
    php_path=$(get_php_path)
    
    if [[ -z "$php_path" ]]; then
        log_error "PHP is not available"
        return 1
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
    install <version>    Install a specific PHP version
    uninstall <version>  Uninstall a specific PHP version
    use <version>        Switch to a specific PHP version
    current             Show the current PHP version
    list                List installed PHP versions
    list-available      List available PHP versions for download
    exec <command>      Execute command with current PHP version
    which               Show path to current PHP binary
    help                Show this help message

EXAMPLES:
    phpv install 8.3.12         # Install PHP 8.3.12
    phpv use 8.3.12             # Switch to PHP 8.3.12
    phpv use system             # Switch to system PHP
    phpv current                # Show current version
    phpv list                   # List installed versions
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
            list_available
            ;;
        "exec")
            exec_php "$@"
            ;;
        "which")
            get_php_path
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
fi
