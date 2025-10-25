#!/usr/bin/env bash

# PHPV - LLVM toolchain management
resolve_llvm_asset_url() {
    local version="$1"
    local machine="$2"
    local target_suffix="$3"

    local api_url="https://api.github.com/repos/llvm/llvm-project/releases/tags/llvmorg-${version}"
    local release_json

    if command -v curl &> /dev/null; then
        if ! release_json=$(with_system_tool_env curl -fsSL "$api_url"); then
            return 1
        fi
    elif command -v wget &> /dev/null; then
        if ! release_json=$(with_system_tool_env wget -qO- "$api_url"); then
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
    urls=$(echo "$release_json" | grep -o '"browser_download_url": *"[^\"]*"' | sed -E 's/.*"browser_download_url": *"([^\"]*)"/\1/' | sed 's/%2B/+/g' | tr -d '\r')

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

    printf '%s' "$chosen"
}