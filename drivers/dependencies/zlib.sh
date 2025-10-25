#!/usr/bin/env bash

# Install zlib from source
install_zlib_from_source() {
    build_from_source "zlib" "1.3.1" "https://zlib.net/zlib-1.3.1.tar.gz" "--shared"
}
