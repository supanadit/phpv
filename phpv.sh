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

# Install zlib from source
install_zlib_from_source() {
    local version="1.3.1"
    local url="https://zlib.net/zlib-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/zlib-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install OpenSSL from source
install_openssl_from_source() {
    local version="3.0.13"
    local url="https://www.openssl.org/source/openssl-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/openssl-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./config --prefix="$PHPV_DEPS_DIR" --openssldir="$PHPV_DEPS_DIR/ssl"
    make -j$(nproc)
    make install
}

# Install libxml2 from source
install_libxml2_from_source() {
    local version="2.11.5"
    local url="https://download.gnome.org/sources/libxml2/2.11/libxml2-$version.tar.xz"
    local build_dir="$PHPV_CACHE_DIR/libxml2-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xJ --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xJ --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./configure --prefix="$PHPV_DEPS_DIR" --without-python
    make -j$(nproc)
    make install
}

# Install oniguruma from source
install_oniguruma_from_source() {
    local version="6.9.8"
    local url="https://github.com/kkos/oniguruma/releases/download/v$version/onig-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/onig-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    export CFLAGS="-Wno-incompatible-pointer-types $CFLAGS"
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install libpng from source
install_libpng_from_source() {
    local version="1.6.40"
    local url="https://download.sourceforge.net/libpng/libpng-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/libpng-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install libjpeg from source
install_libjpeg_from_source() {
    local version="9e"
    local url="https://www.ijg.org/files/jpegsrc.v$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/jpeg-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install freetype from source
install_freetype_from_source() {
    local version="2.13.2"
    local url="https://download.savannah.gnu.org/releases/freetype/freetype-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/freetype-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install ICU from source
install_icu_from_source() {
    local version="73.2"
    local url="https://github.com/unicode-org/icu/releases/download/release-$(echo $version | tr . -)/icu4c-$(echo $version | tr . _)_src.tgz"
    local build_dir="$PHPV_CACHE_DIR/icu-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    cd source
    ./configure --prefix="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install curl from source
install_curl_from_source() {
    local version="8.5.0"
    local url="https://curl.se/download/curl-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/curl-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./configure --prefix="$PHPV_DEPS_DIR" --with-openssl="$PHPV_DEPS_DIR"
    make -j$(nproc)
    make install
}

# Install libzip from source
install_libzip_from_source() {
    local version="1.10.1"
    local url="https://libzip.org/download/libzip-$version.tar.gz"
    local build_dir="$PHPV_CACHE_DIR/libzip-$version"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" | tar -xz --strip-components=1
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O - | tar -xz --strip-components=1
    else
        log_error "curl or wget required to download dependencies"
        return 1
    fi
    ./configure --prefix="$PHPV_DEPS_DIR"
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
    
    # Check for required build dependencies
    if ! command -v gcc &> /dev/null || ! command -v make &> /dev/null; then
        log_error "Build tools (gcc, make) are required but not installed"
        log_info "On Ubuntu/Debian: sudo apt-get install build-essential"
        log_info "On CentOS/RHEL: sudo yum groupinstall 'Development Tools'"
        return 1
    fi
    
    # Set environment for custom dependencies
    export PKG_CONFIG_PATH="$PHPV_DEPS_DIR/lib/pkgconfig:$PKG_CONFIG_PATH"
    export LDFLAGS="-L$PHPV_DEPS_DIR/lib $LDFLAGS"
    export CPPFLAGS="-I$PHPV_DEPS_DIR/include $CPPFLAGS"
    export LD_LIBRARY_PATH="$PHPV_DEPS_DIR/lib:$LD_LIBRARY_PATH"
    
    # Install required dependencies from source if not present
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libz.so" ]]; then
        log_info "Installing zlib from source..."
        install_zlib_from_source || return 1
    fi
    if [[ ! -f "$PHPV_DEPS_DIR/lib/libssl.so" ]]; then
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
        
        if command -v curl &> /dev/null; then
            curl -fsSL "$download_url" -o "$cache_file"
        elif command -v wget &> /dev/null; then
            wget -q "$download_url" -O "$cache_file"
        else
            log_error "Either curl or wget is required to download PHP"
            return 1
        fi
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
        --with-curl="$PHPV_DEPS_DIR" \
        --with-openssl="$PHPV_DEPS_DIR" \
        --with-zlib="$PHPV_DEPS_DIR" \
        --with-libxml-dir="$PHPV_DEPS_DIR" \
        --with-onig="$PHPV_DEPS_DIR" \
        --with-libzip="$PHPV_DEPS_DIR" \
        --enable-bcmath \
        --enable-calendar \
        --enable-exif \
        --enable-ftp \
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
    cat > "$install_dir/etc/php.ini" << EOF
; Basic PHP configuration
memory_limit = 256M
max_execution_time = 30
upload_max_filesize = 64M
post_max_size = 64M
date.timezone = UTC

; Extensions
extension_dir = "$install_dir/lib/php/extensions"

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
