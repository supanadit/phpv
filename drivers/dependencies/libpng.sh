#!/usr/bin/env bash

# Install libpng from source
install_libpng_from_source() {
    build_from_source "libpng" "1.6.40" "https://download.sourceforge.net/libpng/libpng-1.6.40.tar.gz"
}