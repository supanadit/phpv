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

# Install PHP — sensible defaults, deps auto-resolved, built from source if needed
phpv install 8.4                              # 25 default extensions, works out of the box
phpv install 7.4                              # Same defaults, auto-bundles OpenSSL 1.1.1w

# Customize
phpv install 8.4 --ext openssl,curl,apcu     # Build exactly this list (no defaults)
phpv install 8.4 --minimal                    # Bare build (--disable-all --enable-cli only)
phpv install 8.4 --jobs 4                     # Parallel make with 4 jobs
phpv install 8.4 --fresh                      # Clean rebuild (delete prefix, keep cached source)
phpv install 8.4 --verbose                    # See full build output

# Switch versions
phpv use 8.3                                  # Current shell
phpv use system                               # Use system PHP
phpv use 8.3 --global                         # Global default
phpv default 8.3                              # Set global default
phpv versions                                 # List installed
phpv which                                    # Path to current PHP

# Per-project version via .php-version
echo "7.2" > .php-version                     # Auto-switch on cd

# Rebuild with different extensions (smart — only rebuilds PHP, keeps deps)
phpv rebuild 7.2 --ext phar,iconv,filter,fileinfo,dom,session

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

# Export and import PHP builds
phpv share 8.4                                # Export as portable tar.gz bundle
phpv install 8.4 --from bundle.tar.gz         # Install from bundle

# Self-update
phpv update

# Uninstall
phpv uninstall 8.3

# Shell completion
phpv completion bash                          # Generate shell completion
```

---

Each PHP version gets its own isolated dependency tree and phar directory — no conflicts between versions. State files track build progress for resume support.

---

## Commands Reference

| Command | Description |
| --------- | ------------- |
| `phpv install <ver>` | Install a PHP version with default extensions |
| `phpv rebuild <ver>` | Rebuild PHP with different extensions (keeps deps) |
| `phpv uninstall <ver>` | Remove an installed PHP version |
| `phpv use <ver>` | Switch PHP version for current shell |
| `phpv default <ver>` | Set global default PHP version |
| `phpv versions` | List installed PHP versions |
| `phpv which` | Show path to current PHP binary |
| `phpv init <shell>` | Generate shell integration (bash/zsh/fish/pwsh/ksh) |
| `phpv rehash` | Regenerate all shims |
| `phpv doctor [ver]` | System readiness check + extension analysis |
| `phpv update` | Self-update phpv |
| `phpv config` | View and manage configuration |
| `phpv completion <shell>` | Generate shell completion |
| `phpv share <ver>` | Export PHP as portable bundle |
| `phpv extensions [--php <ver>]` | List available/installed extensions |
| `phpv extension add <ver> <name>` | Install an extension post-build |
| `phpv extension remove <ver> <name>` | Remove an extension |
| `phpv phar install <name>` | Install a PHAR tool (composer/pie/wp-cli/phpunit) |
| `phpv phar list` | List installed PHAR tools |
| `phpv phar update <name>` | Update a PHAR tool |
| `phpv phar which <name>` | Show path to a PHAR tool |
| `phpv pecl install <archive>` | Install a PECL extension |
| `phpv pecl list` | List installed PECL extensions |
| `phpv pecl uninstall <name>` | Remove a PECL extension |

### Install Flags

| Flag | Description |
| ------ | ------------- |
| `--ext <list>` | Comma-separated extension list (replaces defaults) |
| `--minimal` | Bare build (--disable-all --enable-cli only) |
| `--fresh` | Delete prefix, keep cached source |
| `--clean` | Delete prefix + source + state |
| `--force` | Force reinstall even if already installed |
| `--static` | Fully static build |
| `--jobs <n>` | Parallel make jobs |
| `--verbose` | Show full build output |
| `--auto-deps` | Auto-install system dependencies |
| `--no-system` | Build all deps from source, skip system packages |
| `--dry-run` | Show what would be done without doing it |
| `--from <bundle>` | Install from a bundle file |

---

## Supported PHP Versions

| PHP | Default extensions | Notes |
| ----- | -------------------- | ------- |
| 8.x | 25 | Full default set |
| 7.0+ | 25 | Full default set |
| 5.6 | 24 | `opcache` skipped (requires PHP 7.0+) |
| 5.2–5.5 | 23 | `opcache` + `json` skipped |
| 5.0–5.1 | 23 | Same as 5.2 ( Experimental ) |
| 4.x | 0 | Use `--ext` to pick extensions ( Experimental ) |

Default extensions: `bcmath`, `curl`, `dom`, `fileinfo`, `filter`, `gd`, `iconv`, `intl`, `json`, `mbstring`, `openssl`, `opcache`, `pdo`, `pdo_mysql`, `pdo_sqlite`, `phar`, `session`, `simplexml`, `sqlite3`, `tokenizer`, `xml`, `xmlreader`, `xmlwriter`, `zip`, `zlib`

Use `--minimal` for a bare build (`--disable-all --enable-cli` only), or `--ext` to specify your own list.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, architecture, and commit conventions.

## License

[MIT](LICENSE) — Copyright (c) 2025 Supan Adit Pratama
