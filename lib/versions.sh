#!/usr/bin/env bash

# PHPV - Version management functions

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

# Get available PHP versions for download
get_available_versions() {
    # This would typically fetch from PHP's release API
    # For now, we'll provide a hardcoded list of common versions
    cat << 'EOF'
8.4.13
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

    if [[ "$php_version" == 5.* ]]; then
        echo "${PHPV_LLVM_VERSION_PHP5:-15.0.6}"
        return
    fi

    if [[ "$php_version" == 7.* ]]; then
        echo "${PHPV_LLVM_VERSION_PHP7:-15.0.6}"
        return
    fi

    echo "$default_version"
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

# Resolve an installed version from partial input (e.g., "7.4" -> "7.4.33")
resolve_installed_version() {
    local input_version="$1"
    
    if [[ -z "$input_version" ]]; then
        return 1
    fi
    
    # Special case: system version
    if [[ "$input_version" == "system" ]]; then
        echo "system"
        return 0
    fi
    
    # If it's already a full version (x.y.z) and installed, return as-is
    if [[ "$input_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] && is_version_installed "$input_version"; then
        echo "$input_version"
        return 0
    fi
    
    # Build filter pattern for partial version
    local filter_pattern="$input_version"
    if [[ "$filter_pattern" != *"." ]]; then
        filter_pattern="$filter_pattern."
    fi
    
    # Get matching installed versions and find the latest one
    local latest_version
    latest_version=$(get_installed_versions | grep -v "^system$" | grep "^$filter_pattern" | sort -V | tail -n1)
    
    if [[ -z "$latest_version" ]]; then
        return 1
    fi
    
    echo "$latest_version"
}