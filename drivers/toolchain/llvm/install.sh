#!/usr/bin/env bash

# Install LLVM/Clang toolchain without relying on system packages
install_llvm_toolchain() {
    local requested_version="${1:-$PHPV_LLVM_VERSION}"
    local machine
    machine=$(uname -m)
    local os
    os=$(uname -s)

    if [[ "$os" != "Linux" ]]; then
        log_error "Automatic LLVM installation currently supports Linux only"
        return 1
    fi

    local install_dir
    local selected_version=""
    local asset_url="${PHPV_LLVM_ARCHIVE_URL:-}"

    local candidates=()
    append_unique candidates "$requested_version"

    if [[ -z "$PHPV_LLVM_ARCHIVE_URL" ]]; then
        if [[ "$requested_version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
            local requested_major="${BASH_REMATCH[1]}"
            local requested_minor="${BASH_REMATCH[2]}"
            local requested_patch="${BASH_REMATCH[3]}"

            local patch_candidate=$((requested_patch - 1))
            while (( patch_candidate >= 0 )); do
                append_unique candidates "${requested_major}.${requested_minor}.${patch_candidate}"
                ((patch_candidate--))
            done
        fi

        local fallback_versions=("17.0.6" "17.0.5" "16.0.6" "16.0.0" "15.0.7")
        for v in "${fallback_versions[@]}"; do
            append_unique candidates "$v"
        done
    fi

    for candidate_version in "${candidates[@]}"; do
        install_dir="$PHPV_DEPS_DIR/llvm-$candidate_version"
        if [[ -x "$install_dir/bin/clang" ]]; then
            selected_version="$candidate_version"
            asset_url=""
            break
        fi

        if [[ -n "$PHPV_LLVM_ARCHIVE_URL" && "$candidate_version" != "$requested_version" ]]; then
            continue
        fi

        local resolved_url
        if [[ -n "$PHPV_LLVM_ARCHIVE_URL" ]]; then
            resolved_url="$PHPV_LLVM_ARCHIVE_URL"
        else
            if ! resolved_url=$(resolve_llvm_asset_url "$candidate_version" "$machine" "$PHPV_LLVM_TARGET_SUFFIX"); then
                log_warning "No compatible LLVM archive found for $candidate_version ($machine)"
                continue
            fi
        fi

        selected_version="$candidate_version"
        asset_url="$resolved_url"
        break
    done

    if [[ -z "$selected_version" ]]; then
        log_error "Could not locate a suitable LLVM archive. Set PHPV_LLVM_ARCHIVE_URL to a downloadable asset."
        return 1
    fi

    install_dir="$PHPV_DEPS_DIR/llvm-$selected_version"

    if [[ -z "$asset_url" ]]; then
        if [[ "$selected_version" != "$requested_version" ]]; then
            log_warning "Using LLVM $selected_version because binaries for $requested_version were not found."
        fi
        PHPV_ACTIVE_LLVM_VERSION="$selected_version"

        if [[ "$selected_version" != "$requested_version" ]]; then
            log_info "Using existing LLVM $selected_version installation"
        fi
        return 0
    fi

    if [[ "$selected_version" != "$requested_version" ]]; then
        log_warning "Falling back to LLVM $selected_version because binaries for $requested_version were not found."
    fi

    log_info "Installing LLVM/Clang $selected_version..."

    local archive
    archive="${asset_url##*/}"
    local cache_file="$PHPV_CACHE_DIR/$archive"
    log_info "Selected LLVM asset: $archive"

    if [[ ! -f "$cache_file" ]]; then
        log_info "Downloading $archive"
        local download_result=1
        if command -v curl &> /dev/null; then
            if with_system_tool_env curl -fsSL "$asset_url" -o "$cache_file"; then
                download_result=0
            fi
        fi

        # If curl failed or not available, try wget
        if [[ $download_result -ne 0 ]] && command -v wget &> /dev/null; then
            if with_system_tool_env wget -q "$asset_url" -O "$cache_file"; then
                download_result=0
            fi
        fi

        if [[ $download_result -ne 0 ]]; then
            rm -f "$cache_file"
            log_error "Failed to download LLVM from $asset_url"
            return 1
        fi
    fi

    local extract_dir="$PHPV_CACHE_DIR/llvm-$selected_version-extract"
    rm -rf "$extract_dir"
    mkdir -p "$extract_dir"
    if ! tar -xJf "$cache_file" -C "$extract_dir"; then
        rm -rf "$extract_dir"
        rm -f "$cache_file"
        log_error "Failed to unpack LLVM archive"
        return 1
    fi

    local unpacked
    unpacked=$(find "$extract_dir" -maxdepth 1 -mindepth 1 -type d -name "clang+llvm-${selected_version}*" | head -n1)
    if [[ -z "$unpacked" ]]; then
        log_error "Failed to locate LLVM directory after extraction"
        return 1
    fi

    rm -rf "$install_dir"
    mv "$unpacked" "$install_dir"
    rm -rf "$extract_dir"
    PHPV_ACTIVE_LLVM_VERSION="$selected_version"
}