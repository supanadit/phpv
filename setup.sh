#!/usr/bin/env bash

# PHPV Setup Script
# This script sets up phpv for easy use

set -e

PHPV_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHPV_SCRIPT="$PHPV_SCRIPT_DIR/phpv.sh"
PHPV_ROOT="${PHPV_ROOT:-$HOME/.phpv}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}PHPV Setup${NC}"
echo "Setting up PHP Version Manager..."

# Create phpv directory
mkdir -p "$PHPV_ROOT"
mkdir -p "$PHPV_ROOT/bin"

# Copy the main script
cp "$PHPV_SCRIPT" "$PHPV_ROOT/bin/phpv"
chmod +x "$PHPV_ROOT/bin/phpv"

# Create shell integration script
cat > "$PHPV_ROOT/phpv.sh" << 'EOF'
# PHPV Shell Integration

export PHPV_ROOT="$HOME/.phpv"
export PATH="$PHPV_ROOT/bin:$PATH"

# Function to update PATH with current PHP version
phpv_update_path() {
    # Remove old phpv PHP paths
    export PATH=$(echo "$PATH" | sed -E 's|:[^:]*\.phpv/versions/[^:]*/(bin|sbin)||g' | sed 's|^[^:]*\.phpv/versions/[^:]*/bin:||')
    
    local current_version_file="$PHPV_ROOT/version"
    if [[ -f "$current_version_file" ]]; then
        local current_version=$(cat "$current_version_file")
        if [[ "$current_version" != "system" && -d "$PHPV_ROOT/versions/$current_version" ]]; then
            export PATH="$PHPV_ROOT/versions/$current_version/bin:$PHPV_ROOT/versions/$current_version/sbin:$PATH"
        fi
    fi
}

# Auto-update PATH on shell startup
phpv_update_path

# Override phpv use command to update PATH
phpv() {
    "$PHPV_ROOT/bin/phpv" "$@"
    local exit_code=$?
    
    # Update PATH after 'use' command
    if [[ "$1" == "use" && $exit_code -eq 0 ]]; then
        phpv_update_path
    fi
    
    return $exit_code
}
EOF

echo -e "${GREEN}✓${NC} PHPV installed to $PHPV_ROOT"

# Detect shell and provide setup instructions
SHELL_NAME=$(basename "$SHELL")
SHELL_RC=""

case "$SHELL_NAME" in
    bash)
        if [[ -f "$HOME/.bashrc" ]]; then
            SHELL_RC="$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            SHELL_RC="$HOME/.bash_profile"
        fi
        ;;
    zsh)
        SHELL_RC="$HOME/.zshrc"
        ;;
    fish)
        SHELL_RC="$HOME/.config/fish/config.fish"
        ;;
esac

echo
echo -e "${YELLOW}Setup Instructions:${NC}"
echo "To complete the setup, add the following line to your shell configuration file:"
echo
echo -e "${BLUE}source $PHPV_ROOT/phpv.sh${NC}"
echo

if [[ -n "$SHELL_RC" && -f "$SHELL_RC" ]]; then
    echo "For your current shell ($SHELL_NAME), add it to: $SHELL_RC"
    echo
    read -p "Would you like me to add it automatically? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo >> "$SHELL_RC"
        echo "# PHPV" >> "$SHELL_RC"
        echo "source $PHPV_ROOT/phpv.sh" >> "$SHELL_RC"
        echo -e "${GREEN}✓${NC} Added to $SHELL_RC"
        echo "Please restart your shell or run: source $SHELL_RC"
    else
        echo "Manual setup required. Add the source line to your shell config file."
    fi
else
    echo "Please add it to your shell configuration file manually."
fi

echo
echo -e "${BLUE}Quick Start:${NC}"
echo "1. Restart your shell or run: source $PHPV_ROOT/phpv.sh"
echo "2. Install a PHP version: phpv install 8.3.12"
echo "3. Switch to it: phpv use 8.3.12"
echo "4. Verify: phpv current"
echo
echo "Run 'phpv help' for more commands."