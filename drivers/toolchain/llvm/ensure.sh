#!/usr/bin/env bash

ensure_llvm_toolchain() {
    local requested_version="${1:-$PHPV_LLVM_VERSION}"

    install_llvm_toolchain "$requested_version" || return 1

    local active_version="${PHPV_ACTIVE_LLVM_VERSION:-$requested_version}"
    local llvm_dir="$PHPV_DEPS_DIR/llvm-$active_version"
    local clang_path="$llvm_dir/bin/clang"
    local clangxx_path="$llvm_dir/bin/clang++"

    if [[ ! -x "$clang_path" || ! -x "$clangxx_path" ]]; then
        log_error "LLVM toolchain installation failed"
        return 1
    fi

    prepend_path "$llvm_dir/bin"
    export CC="$clang_path"
    export CXX="$clangxx_path"
    export AR="$llvm_dir/bin/llvm-ar"
    export NM="$llvm_dir/bin/llvm-nm"
    export RANLIB="$llvm_dir/bin/llvm-ranlib"
    export LLVM_HOME="$llvm_dir"
}