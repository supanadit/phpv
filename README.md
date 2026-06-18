![Logo](assets/logo.png)

# phpv — The PHP Version Manager That Actually Compiles

[![Go Version](https://img.shields.io/github/go-mod/go-version/supanadit/phpv)](https://github.com/supanadit/phpv)
[![License](https://img.shields.io/github/license/supanadit/phpv)](https://github.com/supanadit/phpv/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/supanadit/phpv)](https://github.com/supanadit/phpv/releases)

PHP has no pre-built Linux binaries. Every other major language does. phpv resolves the full transitive dependency graph — OpenSSL, libxml2, curl, zlib, oniguruma, ICU — checks what's already on your system, builds what's missing from source, then compiles PHP with the correct `--with-*` flags. `phpv install 8.4` actually works.

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash
```

Or download a binary from [releases](https://github.com/supanadit/phpv/releases):

```bash
curl -fsSL https://github.com/supanadit/phpv/releases/latest/download/phpv-linux-amd64 -o phpv
chmod +x phpv && sudo mv phpv /usr/local/bin/
```

From source:

```bash
git clone https://github.com/supanadit/phpv.git
cd phpv && go build -o phpv ./app/phpv.go
```

---

## Quick Start

```bash
# Initialize your shell (add to ~/.bashrc, ~/.zshrc, or fish config)
eval "$(phpv init bash)"

# Install PHP — deps auto-resolved, built from source if needed
phpv install 7.2 --ext curl,openssl,intl,gd,mbstring,pdo_mysql
phpv install 8.3 --ext curl,openssl,mbstring,phar,pdo
phpv install 5.6 --compiler zig              # Old PHP with Zig fallback

# Switch versions
phpv use 8.3                                  # Current shell
phpv use system                               # Use system PHP
phpv default 8.3                              # Global default
phpv versions                                 # List installed
phpv which                                    # Path to current PHP

# Rebuild with different extensions (smart — only rebuilds PHP, keeps deps)
phpv rebuild 7.2 --ext phar,iconv,filter,fileinfo,dom,session

# Per-project version via .phpvrc
echo "7.2" > .phpvrc                          # Auto-switch on cd

# List available extensions for any PHP version
phpv extensions --php 7.2

# Manage PHAR tools (per-version — each PHP version has its own phars)
phpv phar install composer                    # Auto-detects latest compatible
phpv phar install pie                         # Install PIE
phpv phar install wp-cli                      # Install WP-CLI
phpv phar update composer                     # Update to latest
phpv phar list                                # List for current PHP version
phpv phar which composer                      # Show phar path

# PECL extensions
phpv pecl install /path/to/ext-1.0.0.tgz
phpv pecl list
phpv pecl uninstall ext

# Diagnose issues
phpv doctor                                   # System readiness check
phpv doctor 8.4                               # Extension analysis for PHP 8.4
phpv install 8.4 --fresh --verbose
```

---

## Features

- **Full dependency resolution** — Transitive dependency graph with version constraints. Missing libraries? Built from source automatically.
- **Bundled PHP extensions** — Each mapped to the correct `./configure` flag, system library, and compatible version range.
- **Dependency-sorted configure flags** — `sortByDependency()` ensures `--enable-hash` always precedes `--enable-phar`, enabling native SHA-256/SHA-512 phar signatures on all PHP versions. Composer 2.x works out of the box on PHP 7.x.
- **Smart `phpv rebuild`** — Runs all three validation gates (unknown extensions, conflicts, implied deps auto-expansion). Only builds missing dependencies, reuses cached ones. No more "configure failed" from missing deps.
- **Per-version PHAR management** — Each PHP version gets its own isolated phar directory at `versions/<ver>/phar/`. Different versions can have different versions of composer, pie, or wp-cli.
- **Auto-regenerating shims** — Every `phpv init zsh` call regenerates all shims (php, phpize, composer, pie, wp) to match the current binary. No stale templates.
- **PECL extension management** — Install, list, and uninstall with full build orchestration.
- **System library detection** — Discovers installed dev packages via `pkg-config` and header checks. Uses system libs when available, builds from source when not.
- **Parallel dependency builds** — Dependencies at the same graph level compile concurrently.
- **Zig compiler fallback** — Old PHP versions that fail with modern GCC get auto-provisioned Zig as a drop-in C compiler.
- **Multi-version support** — PHP 4.x through 8.x side by side, each with isolated dependencies.
- **Smart version resolution** — `8` → latest 8.x, `8.4` → latest 8.4.x, `8.4.5` → exact version.
- **Per-version extension flags (FlagVersions)** — Same extension can use different configure flags per PHP version range (libxml: `--enable-libxml` for <8.0, `--with-libxml` for 8.0+).
- **Compiler flag probing** — CFLAGS are tested against the actual compiler at runtime. Unsupported flags are silently dropped instead of breaking builds.
- **ICU version matrix** — `intl` extension auto-selects the right ICU version per PHP version (57.2 for <7.4, 63.1 for 7.4, 74.2 for 8.0+).
- **Three-tier extension gating** — Unknown extensions halt, conflicting extensions halt, missing implied deps auto-expand with a warning.
- **`.phpvrc` support** — Per-project PHP version auto-switching.
- **`phpv init` regenerates shims** — Every shell init call ensures all shims match the current binary.
- **Doctor command** — Checks system readiness, analyzes extension availability per PHP version, suggests install commands.
- **Single binary** — No runtime dependencies. Just Go.
>>>>>>> ve/refactor

---

## PHPV_ROOT Structure

```
$PHPV_ROOT/
├── bin/                    # Shims (php, php-cgi, phpize, php-config, composer, pie, wp)
├── build-tools/            # Shared build tools (autoconf, bison, re2c, etc.)
│   └── {pkg}/{ver}/
├── cache/                  # Downloaded archives (with resume support)
├── default                 # Default PHP version file
├── sources/                # Extracted source code
│   └── {pkg}/{ver}/
└── versions/               # Installed PHP versions
    └── {php-version}/
        ├── dependency/      # Isolated dependencies per PHP version
        │   └── {pkg}/{ver}/
        ├── phar/            # Per-version PHAR binaries (composer.phar, etc.)
        └── output/          # PHP installation prefix
```

Each PHP version gets its own isolated dependency tree and phar directory — no conflicts between versions.

---

## Requirements

### Linux

- `build-essential` or equivalent development tools
- `gcc` or `zig` compiler (phpv auto-provisions zig)
- `make`, `pkg-config`, `xz-utils`

System libraries are auto-detected. For faster installs, pre-install dev packages:

```bash
# Debian/Ubuntu
sudo apt-get install build-essential libssl-dev libcurl4-openssl-dev \
    libxml2-dev libonig-dev libzip-dev libsqlite3-dev libicu-dev pkg-config \
    cmake perl m4 autoconf automake libtool re2c bison xz-utils

# Fedora/RHEL
sudo dnf install @development-tools openssl-devel libcurl-devel \
    libxml2-devel oniguruma-devel libzip-devel sqlite-devel \
    pkg-config cmake perl m4 autoconf automake libtool re2c bison xz
```

### macOS

- Xcode Command Line Tools
- Homebrew packages may be required for some dependencies

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, architecture, and commit conventions.

## License

[MIT](LICENSE) — Copyright (c) 2025 Supan Adit Pratama
