#!/usr/bin/env bash

# Install freetype from source
install_freetype_from_source() {
    build_from_source "freetype" "2.13.2" "https://download.savannah.gnu.org/releases/freetype/freetype-2.13.2.tar.gz"
}
