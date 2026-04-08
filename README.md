![Logo](assets/logo.png)

# phpv - PHP Version Manager

[![Go Version](https://img.shields.io/github/go-mod/go-version/supanadit/phpv)](https://github.com/supanadit/phpv)
[![License](https://img.shields.io/github/license/supanadit/phpv)](https://github.com/supanadit/phpv/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/supanadit/phpv)](https://github.com/supanadit/phpv/releases)
[![Test PHP Versions](https://github.com/supanadit/phpv/actions/workflows/test-php-versions.yml/badge.svg)](https://github.com/supanadit/phpv/actions/workflows/test-php-versions.yml)

phpv is a PHP Version Manager written in Go. It downloads, compiles, and manages multiple PHP versions from source on the same system, similar to [phpbrew](https://github.com/phpbrew/phpbrew) or [nvm](https://github.com/nvm-sh/nvm).

## Features

- **Multi-version support**: Install and manage multiple PHP versions (4.x through 8.x)
- **Source compilation**: Build PHP from source with custom configure options
- **Dependency management**: Automatic dependency resolution and compilation
- **Cross-platform**: Linux and macOS support
- **Shell integration**: Bash, Zsh, and Fish support
- **Shim system**: Dynamic version switching without PATH conflicts
- **Parallel builds**: Fast installation with parallel dependency builds
- **Clean architecture**: Well-structured Go codebase with dependency injection

## Requirements

### Linux

- `build-essential` or similar development tools
- `gcc` or `zig` compiler
- `make`, `autoconf`, `automake`, `libtool`
- `bison`, `flex`, `re2c`, `m4`, `perl`
- `pkg-config`
- `xz-utils` (for .tar.xz extraction)
- `libssl-dev`, `libcurl4-openssl-dev`, `libxml2-dev`, `libonig-dev`, `libzip-dev`

### macOS

- Xcode Command Line Tools
- Homebrew packages may be required for dependencies

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash
```

### Install Specific Version

```bash
INSTALL_VERSION=0.1.0 curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash
```

### From Binary Release

Download the latest binary for your platform from the [releases page](https://github.com/supanadit/phpv/releases):

```bash
# Linux amd64
curl -fsSL https://github.com/supanadit/phpv/releases/download/v0.1.0/phpv-v0.1.0-linux-amd64 -o phpv
chmod +x phpv
sudo mv phpv /usr/local/bin/phpv

# Linux arm64
curl -fsSL https://github.com/supanadit/phpv/releases/download/v0.1.0/phpv-v0.1.0-linux-arm64 -o phpv
chmod +x phpv
sudo mv phpv /usr/local/bin/phpv
```

### From Source

```bash
git clone https://github.com/supanadit/phpv.git
cd phpv
go build -o phpv ./app/phpv.go
sudo mv phpv /usr/local/bin/
```

## Shell Initialization

After installation, add phpv to your shell:

### Bash

Add to `~/.bashrc`:

```bash
eval "$(phpv init bash)"
```

### Zsh

Add to `~/.zshrc`:

```bash
eval "$(phpv init zsh)"
```

### Fish

Add to `~/.config/fish/config.fish`:

```bash
phpv init fish | source
```

Or use the command:

```fish
phpv init fish >> ~/.config/fish/config.fish
```

## Usage

### Install PHP

Install a specific PHP version:

```bash
phpv install 8.4        # Install latest 8.4.x
phpv install 8          # Install latest 8.x.x
phpv install 8.4.0      # Install exact version 8.4.0
```

### Switch PHP Version

Switch to a different PHP version:

```bash
phpv use 8.4
```

Make it the default:

```bash
phpv default 8.4
```

### List Versions

List installed PHP versions:

```bash
phpv versions
```

List available versions from remote:

```bash
phpv list
```

### Other Commands

```bash
phpv which              # Show path to current PHP
phpv uninstall 8.2      # Uninstall PHP 8.2
phpv upgrade            # Upgrade default PHP to latest
phpv upgrade 8.4        # Upgrade PHP 8.4 to latest 8.4.x
phpv doctor             # Check system dependencies
phpv build-tools clean   # Remove unused build tools
```

### Build Options

```bash
# Verbose output
phpv install 8.4 --verbose

# Force rebuild (clean install)
phpv install 8.4 --force

# Fresh build (remove existing first)
phpv install 8.4 --fresh

# Use specific compiler
phpv install 8.4 --compiler zig

# Dry run (preview)
phpv install 8.4 --dry-run

# JSON output
phpv install 8.4 --json

# Quiet mode
phpv install 8.4 --quiet
```

## Shell Completions

### Bash

```bash
source <(phpv completion bash)
# Or install system-wide:
sudo phpv completion bash > /etc/bash_completion.d/phpv
```

### Zsh

```bash
# Add to ~/.zshrc:
autoload -U compinit
compinit

# Install completion:
phpv completion zsh > "${fpath[1]}/_phpv"
```

### Fish

```bash
phpv completion fish > ~/.config/fish/completions/phpv.fish
```

### PowerShell

```powershell
phpv completion powershell >> $PROFILE
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PHPV_ROOT` | Root directory for phpv data | `~/.phpv` |
| `PHPV_CURRENT` | Override current PHP version | (none) |

## PHPV_ROOT Structure

```
$PHPV_ROOT/
├── bin/                    # Shim binaries (php, php-cgi, etc.)
├── build-tools/            # Shared build tools
│   └── {pkg}/{ver}/
├── cache/                  # Downloaded archives
├── default                  # Default PHP version
├── sources/                # Extracted source code
│   └── {pkg}/{ver}/
└── versions/               # Installed PHP versions
    └── {php-version}/
        ├── dependency/      # Isolated dependencies
        └── output/        # PHP installation prefix
```

## Troubleshooting

### doctor command

Run `phpv doctor` to check for missing system dependencies:

```bash
phpv doctor
```

### Common Issues

#### Missing system packages

On Debian/Ubuntu:

```bash
sudo apt-get install \
    build-essential \
    libssl-dev \
    libcurl4-openssl-dev \
    libxml2-dev \
    libonig-dev \
    libzip-dev \
    libsqlite3-dev \
    pkg-config \
    cmake \
    perl \
    m4 \
    autoconf \
    automake \
    libtool \
    re2c \
    bison \
    xz-utils
```

On Fedora/RHEL:

```bash
sudo dnf install \
    @development-tools \
    openssl-devel \
    libcurl-devel \
    libxml2-devel \
    oniguruma-devel \
    libzip-devel \
    sqlite-devel \
    pkg-config \
    cmake \
    perl \
    m4 \
    autoconf \
    automake \
    libtool \
    re2c \
    bison \
    xz
```

On macOS:

```bash
brew install autoconf automake libtool m4 pkg-config re2c bison xz
```

#### Compilation fails

Try with `--fresh` to clean and rebuild:

```bash
phpv install 8.4 --fresh --verbose
```

#### Using system libraries

phpv automatically detects system-installed libraries. If you want to use system libraries instead of building from source, ensure the development packages are installed (e.g., `libxml2-dev` instead of just `libxml2`).

## Comparison with Alternatives

| Feature | phpv | phpbrew | nvm |
|---------|------|---------|-----|
| Language | Go | PHP | Shell |
| Installation | Single binary | Composer | curl script |
| Source builds | Yes | Yes | N/A |
| Binary installs | Future | No | Yes |
| Shell integration | Bash/Zsh/Fish | Bash | Bash/Zsh |
| Auto-switching | Planned | No | .nvmrc |
| Plugin system | Planned | No | Plugins |

## Development

### Running Tests

```bash
go test ./... -v
```

### Running Tests with Coverage

```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Building

```bash
go build -o phpv ./app/phpv.go
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

phpv is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Links

- [Documentation](docs/)
- [Issue Tracker](https://github.com/supanadit/phpv/issues)
- [Release Notes](CHANGELOG.md)
