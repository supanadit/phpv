#!/usr/bin/env bash

get_lib_paths_for_version() {
    local version="$1"
    local -a lib_paths=()

    if [[ -z "$version" || "$version" == "system" ]]; then
        return 0
    fi

    local version_dir="$PHPV_VERSIONS_DIR/$version"
    local llvm_version_file="$version_dir/.llvm_version"
    local deps_dir=""

    if [[ -f "$llvm_version_file" ]]; then
        local llvm_version
        llvm_version=$(cat "$llvm_version_file" 2>/dev/null || true)
        if [[ -n "$llvm_version" ]]; then
            deps_dir=$(get_deps_dir_for_version "$version" "$llvm_version" 2>/dev/null || true)
        fi
    fi

    if [[ -z "$deps_dir" || ! -d "$deps_dir" ]]; then
        if [[ -d "$PHPV_DEPS_BASE_DIR/$version" ]]; then
            deps_dir="$PHPV_DEPS_BASE_DIR/$version"
        fi
    fi

    if [[ -z "$deps_dir" || ! -d "$deps_dir" ]]; then
        return 0
    fi

    [[ -d "$deps_dir/lib" ]] && lib_paths+=("$deps_dir/lib")
    [[ -d "$deps_dir/lib64" ]] && lib_paths+=("$deps_dir/lib64")

    if (( ${#lib_paths[@]} == 0 )); then
        return 0
    fi

    local IFS=':'
    printf '%s' "${lib_paths[*]}"
}