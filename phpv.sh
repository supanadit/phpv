#!/usr/bin/env bash

# PHPV - PHP Version Manager
# Similar to pyenv and nvm but for PHP
# Manages multiple PHP versions in user space

# Source all script modules
PHPV_SCRIPT_DRIVER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/drivers" && pwd)"
PHPV_SCRIPT_DEPENDENCY_DIR="$PHPV_SCRIPT_DRIVER_DIR/dependencies"

PHPV_SCRIPT_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/lib" && pwd)"
PHPV_SCRIPT_BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/bin" && pwd)"

source "$PHPV_SCRIPT_LIB_DIR/common.sh"
source "$PHPV_SCRIPT_LIB_DIR/build.sh"

# Dependencies
source "$PHPV_SCRIPT_DEPENDENCY_DIR/cmake.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/curl.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/freetype.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/icu.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/libjpeg.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/libpng.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/libxml2.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/libzip.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/mariadb.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/mysql.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/odbc.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/oniguruma.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/openssl.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/postgresql.sh"
source "$PHPV_SCRIPT_DEPENDENCY_DIR/zlib.sh"


source "$PHPV_SCRIPT_LIB_DIR/versions.sh"
source "$PHPV_DRIVER_DIR/llvm.sh"
source "$PHPV_SCRIPT_LIB_DIR/commands.sh"
source "$PHPV_SCRIPT_BIN_DIR/main.sh"

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
else
    # Script is being sourced - set up shell integration
    # Instead of exporting functions, we'll define them in the calling scope
    :
fi