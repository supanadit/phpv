#!/usr/bin/env bash

print_environment_overrides() {
    local current_version
    current_version=$(get_current_version)

    local path_prefix=""
    local ld_prefix=""
    local ld_root=""
    local env_mode="system"

    if [[ -n "$current_version" && "$current_version" != "system" ]]; then
        local version_bin_dir="$PHPV_VERSIONS_DIR/$current_version/bin"
        if [[ -d "$version_bin_dir" ]]; then
            path_prefix="$version_bin_dir"
        fi
        ld_prefix=$(get_lib_paths_for_version "$current_version")
        ld_root="$PHPV_DEPS_BASE_DIR"
        env_mode="$current_version"
    fi

    printf 'PATH_PREFIX=%s\n' "$path_prefix"
    printf 'LD_LIBRARY_PATH_PREFIX=%s\n' "$ld_prefix"
    printf 'LD_LIBRARY_PATH_ROOT=%s\n' "$ld_root"
    printf 'ENV_MODE=%s\n' "$env_mode"
}