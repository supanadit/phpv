#!/usr/bin/env bash
#
# PHPV Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash
#        INSTALL_VERSION=0.1.1 curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash
#

set -e

PHPV_INSTALLER_VERSION="1.0.0"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}==>${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}✓${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $*" >&2
}

log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       echo "unknown" ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)       echo "$arch" ;;
    esac
}

detect_shell() {
    local shell

    # First, try to get the login shell from /etc/passwd
    if command -v getent >/dev/null 2>&1; then
        shell="$(getent passwd "$(whoami)" 2>/dev/null | cut -d: -f7)"
        shell="$(basename "$shell" 2>/dev/null)"
    fi

    # Fallback to SHELL environment variable
    if [ -z "$shell" ] && [ -n "$SHELL" ]; then
        shell="$(basename "$SHELL")"
    fi

    # Normalize shell name
    case "$shell" in
        bash) echo "bash" ;;
        zsh)  echo "zsh" ;;
        fish) echo "fish" ;;
        pwsh|powershell) echo "pwsh" ;;
        ksh|ksh93|mksh) echo "ksh" ;;
        *)    echo "bash" ;;
    esac
}

get_shell_config() {
    local shell="$1"
    case "$shell" in
        bash) echo "${HOME}/.bashrc" ;;
        zsh)  echo "${HOME}/.zshrc" ;;
        fish) echo "${HOME}/.config/fish/config.fish" ;;
        *)    echo "${HOME}/.bashrc" ;;
    esac
}

check_curl() {
    if command -v curl >/dev/null 2>&1; then
        echo "curl"
    elif command -v wget >/dev/null 2>&1; then
        echo "wget"
    else
        log_error "Neither curl nor wget found. Please install curl or wget first."
        exit 1
    fi
}

download_file() {
    local url="$1"
    local output="$2"
    local downloader

    downloader="$(check_curl)"

    if [ "$downloader" = "curl" ]; then
        curl -fSL "$url" -o "$output"
    else
        wget -O "$output" "$url"
    fi
}

get_latest_version() {
    local downloader
    downloader="$(check_curl)"
    local version

    if [ "$downloader" = "curl" ]; then
        version=$(curl -fsSL https://api.github.com/repos/supanadit/phpv/releases 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/')
    else
        version=$(wget -qO- https://api.github.com/repos/supanadit/phpv/releases 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/')
    fi

    if [ -z "$version" ]; then
        log_error "Failed to detect latest version. Please specify INSTALL_VERSION explicitly."
        exit 1
    fi

    echo "$version"
}

get_download_url() {
    local version="$1"
    local os="$2"
    local arch="$3"

    local url="https://github.com/supanadit/phpv/releases/download/${version}/phpv-${version}-${os}-${arch}"

    echo "$url"
}

get_installed_version() {
    local phpv_root="${PHPV_ROOT:-$HOME/.phpv}"
    local version_file="${phpv_root}/version"

    if [ -f "$version_file" ]; then
        cat "$version_file"
    fi
}

write_installed_version() {
    local version="$1"
    local phpv_root="${PHPV_ROOT:-$HOME/.phpv}"

    mkdir -p "$phpv_root"
    echo "$version" > "${phpv_root}/version"
}

compare_versions() {
    local current="$1"
    local target="$2"

    if [ -z "$current" ] || [ -z "$target" ]; then
        echo "unknown"
        return
    fi

    if [ "$current" = "$target" ]; then
        echo "same"
        return
    fi

    local lowest
    lowest="$(printf '%s\n%s\n' "$current" "$target" | sort -V | head -n1)"

    if [ "$lowest" = "$current" ]; then
        echo "upgrade"
    else
        echo "downgrade"
    fi
}

install_phpv() {
    local version="$1"
    local action="$2"
    local install_dir="$3"
    local bin_dir="${install_dir}/bin"
    local bin_path="${bin_dir}/phpv"

    mkdir -p "$bin_dir"

    local os
    os="$(detect_os)"
    local arch
    arch="$(detect_arch)"

    if [ "$os" = "unknown" ] || [ "$arch" != "amd64" ]; then
        log_error "Unsupported platform: ${os}-${arch}. Currently only linux-amd64 is supported."
        exit 1
    fi

    case "$action" in
        skip)
            log_info "PHPV v${version} is already the latest version"
            echo "$bin_path"
            write_installed_version "$version"
            return 0
            ;;
        downgrade)
            log_info "Downgrading PHPV to v${version}..."
            ;;
        *)
            log_info "Installing PHPV v${version}..."
            ;;
    esac

    if [ -f "$bin_path" ]; then
        log_info "Backing up existing binary..."
        mv "$bin_path" "${bin_path}.backup.$(date +%s)"
    fi

    local download_url
    download_url="$(get_download_url "$version" "$os" "$arch")"

    log_info "Downloading from ${download_url}..."

    local tmp_file
    tmp_file="$(mktemp)"

    if ! download_file "$download_url" "$tmp_file"; then
        rm -f "$tmp_file"
        log_error "Failed to download PHPV. Please check if version ${version} exists."
        exit 1
    fi

    mv "$tmp_file" "$bin_path"
    chmod +x "$bin_path"

    write_installed_version "$version"

    log_success "Installed PHPV to ${bin_path}"

    echo "$bin_path"
}

setup_shell_integration() {
    local bin_path="$1"
    local shell="$2"

    log_info "Setting up shell integration for ${shell}..."

    local init_line="eval \"\$(${bin_path} init -)\""

    local shell_config
    shell_config="$(get_shell_config "$shell")"

    if [ -z "$shell_config" ]; then
        log_warn "Could not detect shell config file. Skipping automatic setup."
        return 0
    fi

    if [ ! -f "$shell_config" ]; then
        touch "$shell_config"
    fi

    if grep -q "phpv init" "$shell_config" 2>/dev/null; then
        log_success "PHPV shell integration already configured in ${shell_config}"
        return 0
    fi

    log_info "Adding PHPV initialization to ${shell_config}..."

    {
        echo ""
        echo "# PHPV - PHP Version Manager"
        echo "${init_line}"
    } >> "$shell_config"

    log_success "Added shell integration to ${shell_config}"
}

main() {
    log_info "PHPV Installer v${PHPV_INSTALLER_VERSION}"
    echo

    local version="${INSTALL_VERSION:-}"
    local phpv_root="${PHPV_ROOT:-$HOME/.phpv}"
    local action="install"

    local os
    os="$(detect_os)"
    local arch
    arch="$(detect_arch)"

    log_info "Platform: ${os}-${arch}"
    log_info "Installation directory: ${phpv_root}"

    if [ -z "$version" ]; then
        log_info "Detecting latest version..."
        version="$(get_latest_version)"
        log_info "Latest version: v${version}"
    else
        log_info "Installing specified version: v${version}"
    fi

    local installed_version
    installed_version="$(get_installed_version)"

    if [ -n "$installed_version" ]; then
        local comparison
        comparison="$(compare_versions "$installed_version" "$version")"

        case "$comparison" in
            same)
                log_info "PHPV v${version} is already installed"
                action="skip"
                ;;
            upgrade)
                log_info "Current version: v${installed_version}"
                log_info "Upgrading to v${version}..."
                action="upgrade"
                ;;
            downgrade)
                log_info "Current version: v${installed_version}"
                log_info "Downgrading to v${version}..."
                action="downgrade"
                ;;
        esac
    else
        log_info "No PHPV installation found, installing..."
    fi

    echo

    local bin_path
    bin_path="$(install_phpv "$version" "$action" "$phpv_root")"

    local shell
    shell="$(detect_shell "")"
    setup_shell_integration "$bin_path" "$shell"

    echo
    case "$action" in
        skip)
            log_success "PHPV v${version} is already the latest version!"
            ;;
        upgrade)
            log_success "PHPV upgraded to v${version} successfully!"
            ;;
        downgrade)
            log_success "PHPV downgraded to v${version} successfully!"
            ;;
        *)
            log_success "PHPV v${version} installed successfully!"
            ;;
    esac
    echo
    echo "Next steps:"
    echo "  1. Restart your shell or run: source ~/.bashrc  # or ~/.zshrc"
    echo "  2. Verify installation: phpv --version"
    echo "  3. List available versions: phpv list"
    echo "  4. Install PHP: phpv install 8.4"
    echo
    echo "For more information, visit: https://github.com/supanadit/phpv"
}

main "$@"
