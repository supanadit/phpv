#!/usr/bin/env bash

# PHPV - PHP Version Manager
# Common configuration and helper functions

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
PHPV_VERBOSE="${PHPV_VERBOSE:-0}"

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

# Progress bar functions
show_progress() {
    local current=$1
    local total=$2
    local label="${3:-Progress}"
    local width=50
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        return
    fi
    
    local percentage=$((current * 100 / total))
    local filled=$((width * current / total))
    local empty=$((width - filled))
    
    printf "\r${BLUE}[INFO]${NC} %s: [" "$label"
    printf "%${filled}s" | tr ' ' '='
    printf "%${empty}s" | tr ' ' ' '
    printf "] %d%%" "$percentage"
}

complete_progress() {
    local label="${1:-Progress}"
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        return
    fi
    
    printf "\r${GREEN}[SUCCESS]${NC} %s: [" "$label"
    printf "%50s" | tr ' ' '='
    printf "] 100%%\n"
}

run_with_progress() {
    local label="$1"
    local total_steps="${2:-100}"
    shift 2
    local log_file="$PHPV_CACHE_DIR/build.log"
    
    if [[ "$PHPV_VERBOSE" == "1" ]]; then
        "$@"
        return $?
    fi
    
    # Run the command in the background and redirect output
    "$@" > "$log_file" 2>&1 &
    local pid=$!
    
    local step=0
    while kill -0 $pid 2>/dev/null; do
        show_progress $step $total_steps "$label"
        step=$(( (step + 1) % total_steps ))
        if [[ $step -eq 0 ]]; then
            step=$((total_steps - 1))
        fi
        sleep 0.1
    done
    
    wait $pid
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        complete_progress "$label"
    else
        printf "\r${RED}[ERROR]${NC} %s failed. See %s for details.\n" "$label" "$log_file"
    fi
    
    return $exit_code
}

with_system_tool_env() {
    local old_path="$PATH"
    local old_ld=""
    local ld_was_set=0

    if [[ "${LD_LIBRARY_PATH+x}" == "x" ]]; then
        old_ld="$LD_LIBRARY_PATH"
        ld_was_set=1
    fi

    unset LD_LIBRARY_PATH
    PATH="/usr/local/bin:/usr/bin:/bin"

    "$@"
    local status=$?

    PATH="$old_path"
    if (( ld_was_set )); then
        export LD_LIBRARY_PATH="$old_ld"
    else
        unset LD_LIBRARY_PATH
    fi

    return $status
}

safe_download() {
    local url="$1"
    local output="$2"

    local result=1

    if command -v wget &> /dev/null; then
        if with_system_tool_env wget -q "$url" -O "$output"; then
            result=0
        fi
    fi

    if [[ $result -ne 0 ]] && command -v curl &> /dev/null; then
        if with_system_tool_env curl -fsSL "$url" -o "$output" 2>/dev/null; then
            result=0
        fi
    fi

    if [[ $result -ne 0 ]]; then
        log_error "Failed to download $url"
    fi

    return $result
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

version_supports_opcache() {
    local version="$1"
    local major="" minor=""

    IFS='.' read -r major minor _ <<< "$version"

    if [[ -z "$major" || -z "$minor" ]]; then
        return 1
    fi

    if (( major > 5 )); then
        return 0
    fi

    if (( major == 5 && minor >= 5 )); then
        return 0
    fi

    return 1
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