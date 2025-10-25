#!/usr/bin/env bash

# PHPV - PHP Version Manager
# Similar to pyenv and nvm but for PHP
# Manages multiple PHP versions in user space

# Source all script modules
PHPV_DRIVER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/drivers" && pwd)"
PHPV_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/lib" && pwd)"
PHPV_BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/bin" && pwd)"

source "$PHPV_LIB_DIR/common.sh"
source "$PHPV_LIB_DIR/build.sh"

# Dependencies
source "$PHPV_DRIVER_DIR/dependencies/cmake.sh"
source "$PHPV_DRIVER_DIR/dependencies/curl.sh"
source "$PHPV_DRIVER_DIR/dependencies/freetype.sh"
source "$PHPV_DRIVER_DIR/dependencies/icu.sh"
source "$PHPV_DRIVER_DIR/dependencies/libjpeg.sh"
source "$PHPV_DRIVER_DIR/dependencies/libpng.sh"
source "$PHPV_DRIVER_DIR/dependencies/libxml2.sh"
source "$PHPV_DRIVER_DIR/dependencies/libzip.sh"
source "$PHPV_DRIVER_DIR/dependencies/mariadb.sh"
source "$PHPV_DRIVER_DIR/dependencies/mysql.sh"
source "$PHPV_DRIVER_DIR/dependencies/odbc.sh"
source "$PHPV_DRIVER_DIR/dependencies/oniguruma.sh"
source "$PHPV_DRIVER_DIR/dependencies/openssl.sh"
source "$PHPV_DRIVER_DIR/dependencies/postgresql.sh"
source "$PHPV_DRIVER_DIR/dependencies/zlib.sh"


source "$PHPV_LIB_DIR/versions.sh"
source "$PHPV_DRIVER_DIR/llvm.sh"
source "$PHPV_LIB_DIR/commands.sh"
source "$PHPV_BIN_DIR/main.sh"

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
else
    # Script is being sourced - set up shell integration
    # Instead of exporting functions, we'll define them in the calling scope
    :
fi