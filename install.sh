#!/usr/bin/env bash

# PHPV Installation Script
# Makes PHPV as easy to install as NVM

set -e

PHPV_REPO_URL="https://raw.githubusercontent.com/supanadit/phpv/main"
PHPV_DIR="${PHPV_DIR:-$HOME/.phpv}"
PHPV_SCRIPT="$PHPV_DIR/phpv.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Detect shell
detect_shell() {
    if [[ -n "$ZSH_VERSION" ]]; then
        echo "zsh"
    elif [[ -n "$BASH_VERSION" ]]; then
        echo "bash"
    else
        echo "unknown"
    fi
}

# Get shell config file
get_shell_config() {
    local shell_type="$1"
    case "$shell_type" in
        zsh)
            echo "$HOME/.zshrc"
            ;;
        bash)
            if [[ -f "$HOME/.bashrc" ]]; then
                echo "$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.bashrc"
            fi
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

# Download PHPV
download_phpv() {
    log_info "Setting up PHPV..."

    mkdir -p "$PHPV_DIR/bin"

    # For development/testing, copy local files instead of downloading
    if [[ -f "./phpv.sh" ]]; then
        cp "./phpv.sh" "$PHPV_SCRIPT"
        log_success "Copied local PHPV script to $PHPV_SCRIPT"
    else
        # Production: download from GitHub
        if command -v curl >/dev/null 2>&1; then
            curl -fsSL "$PHPV_REPO_URL/phpv.sh" -o "$PHPV_SCRIPT"
        elif command -v wget >/dev/null 2>&1; then
            wget -q "$PHPV_REPO_URL/phpv.sh" -O "$PHPV_SCRIPT"
        else
            log_error "Neither curl nor wget found. Please install one of them."
            exit 1
        fi
        log_success "Downloaded PHPV to $PHPV_SCRIPT"
    fi

    chmod +x "$PHPV_SCRIPT"
}

# Create phpv init script
create_init_script() {
    local init_script="$PHPV_DIR/init.sh"

    cat > "$init_script" << 'EOF'
# PHPV Initialization Script
# This script is sourced to set up PHPV in your shell

export PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"
export PATH="$PHPV_ROOT/bin:$PATH"

# PHPV function for shell integration
phpv() {
    local command="$1"
    shift

    case "$command" in
        use)
            # Call the original phpv use command
            "$PHPV_ROOT/phpv.sh" use "$@"
            # Update PATH for current session
            phpv_update_path
            ;;
        *)
            "$PHPV_ROOT/phpv.sh" "$command" "$@"
            ;;
    esac
}

# Update PATH based on current PHP version
phpv_update_path() {
    local current_version
    current_version=$(cat "$PHPV_ROOT/version" 2>/dev/null || echo "system")

    # Remove any existing PHPV version from PATH
    PATH=$(echo "$PATH" | sed 's|:$PHPV_ROOT/versions/[^:]*||g' | sed 's|^$PHPV_ROOT/versions/[^:]*:||g' | sed 's|:$PHPV_ROOT/versions/[^:]*:||g')

    if [[ "$current_version" != "system" && -d "$PHPV_ROOT/versions/$current_version/bin" ]]; then
        export PATH="$PHPV_ROOT/versions/$current_version/bin:$PATH"
    fi
}

# Initialize PATH on shell start
phpv_update_path
EOF

    chmod +x "$init_script"
    log_success "Created PHPV init script at $init_script"
}

# Create phpv binary
create_phpv_binary() {
    local phpv_binary="$PHPV_DIR/bin/phpv"

    cat > "$phpv_binary" << EOF
#!/usr/bin/env bash

# PHPV binary wrapper
# This allows 'phpv' command to work from anywhere

PHPV_ROOT="\${PHPV_ROOT:-\$HOME/.phpv}"
exec "\$PHPV_ROOT/phpv.sh" "\$@"
EOF

    chmod +x "$phpv_binary"
    log_success "Created phpv binary at $phpv_binary"
}

# Update shell config
update_shell_config() {
    local shell_type
    shell_type=$(detect_shell)

    local config_file
    config_file=$(get_shell_config "$shell_type")

    log_info "Detected shell: $shell_type"
    log_info "Shell config file: $config_file"

    # Backup original config
    if [[ -f "$config_file" ]]; then
        cp "$config_file" "${config_file}.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "Backed up $config_file"
    fi

    # Add PHPV initialization
    if ! grep -q "source.*phpv.*init" "$config_file" 2>/dev/null; then
        echo "" >> "$config_file"
        echo "# PHPV - PHP Version Manager" >> "$config_file"
        echo "export PHPV_ROOT=\"\$HOME/.phpv\"" >> "$config_file"
        echo "[ -s \"\$PHPV_ROOT/init.sh\" ] && source \"\$PHPV_ROOT/init.sh\"" >> "$config_file"
        log_success "Added PHPV initialization to $config_file"
    else
        log_warning "PHPV initialization already exists in $config_file"
    fi
}

# Main installation
main() {
    log_info "Installing PHPV - PHP Version Manager"
    log_info "This will set up PHPV similar to how NVM works"
    echo

    # Check if already installed
    if [[ -f "$PHPV_SCRIPT" ]]; then
        log_warning "PHPV is already installed at $PHPV_DIR"
        read -p "Do you want to reinstall? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Installation cancelled"
            exit 0
        fi
    fi

    download_phpv
    create_init_script
    create_phpv_binary
    update_shell_config

    echo
    log_success "PHPV installation completed!"
    echo
    log_info "Please restart your terminal or run: source $(get_shell_config "$shell_type")"
    echo
    log_info "Then you can use PHPV:"
    echo "  phpv install 8.3.12    # Install PHP 8.3.12"
    echo "  phpv use 8.3.12        # Switch to PHP 8.3.12"
    echo "  phpv current           # Show current version"
    echo "  phpv list              # List installed versions"
}

main "$@"