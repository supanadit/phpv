#!/usr/bin/env bash

# Install libzip from source
install_libzip_from_source() {
    build_with_cmake "libzip" "1.10.1" "https://libzip.org/download/libzip-1.10.1.tar.gz"
}