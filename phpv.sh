#!/usr/bin/env bash

# PHPV - PHP Version Manager
# Similar to pyenv and nvm but for PHP
# Manages multiple PHP versions in user space

# Source all script modules
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/scripts" && pwd)"
source "$SCRIPT_DIR/common.sh"
source "$SCRIPT_DIR/build.sh"
source "$SCRIPT_DIR/deps.sh"
source "$SCRIPT_DIR/versions.sh"
source "$SCRIPT_DIR/llvm.sh"
source "$SCRIPT_DIR/commands.sh"
source "$SCRIPT_DIR/main.sh"

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
else
    # Script is being sourced - set up shell integration
    # Instead of exporting functions, we'll define them in the calling scope
    :
fi