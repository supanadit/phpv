# PHPV - PHP Version Manager

A simple PHP version manager for Linux/Unix systems, similar to `pyenv` and `nvm`. PHPV allows you to install, manage, and switch between multiple PHP versions in user space without requiring root privileges for version management.

## Features

- 🚀 Install multiple PHP versions from source
- 🔄 Switch between installed versions instantly
- 🏠 User space installation (no root required for version management)
- 💾 Automatic PATH management
- 🧹 Clean uninstallation of versions
- 📦 Automatic dependency detection and guidance
- 🎨 Colored output for better readability
- 🔙 Backward version support up to PHP 4.0
- 🛠️ Shell integration for bash, zsh, and fish
- 📅 Day one latest PHP support
- 🆓 Open source (MIT License)
- 🐳 Compatible with Docker and CI environments
- 🐧 Works on Linux (All Distros) and macOS
- 🧩 Easily extensible for custom configurations
- 🧱 Isolated dependency management for each PHP version

## Installation

### Quick Install

```bash
# Clone or download the repository
git clone <your-repo-url> ~/.phpv-installer
cd ~/.phpv-installer

# Run the setup script
./setup.sh
```

### Manual Install

1. Copy `phpv.sh` to a directory in your PATH or to `~/.phpv/bin/phpv`
2. Make it executable: `chmod +x ~/.phpv/bin/phpv`
3. Add `~/.phpv/bin` to your PATH
4. Source the shell integration script

## Usage

### Basic Commands

```bash
# Install a PHP version
phpv install 8.3.12

# Switch to a specific version
phpv use 8.3.12

# Switch back to system PHP
phpv use system

# Show current version
phpv current

# List installed versions
phpv list

# List available versions for download
phpv list-available

# Execute PHP command with current version
phpv exec -v
phpv exec composer install

# Show path to current PHP binary
phpv which

# Uninstall a version
phpv uninstall 8.3.12

# Show help
phpv help
```

### Example Workflow

```bash
# Install latest PHP 8.3
phpv install 8.3.12

# Switch to it
phpv use 8.3.12

# Verify installation
phpv current
php -v

# Install another version
phpv install 8.2.24

# List all installed versions
phpv list

# Switch between versions
phpv use 8.2.24
phpv use 8.3.12
phpv use system  # Use system PHP
```

## Configuration

### Environment Variables

- `PHPV_ROOT`: Root directory for phpv installations (default: `~/.phpv`)
- `PHPV_LLVM_VERSION_MAP`: Comma-separated overrides that map PHP versions (supports glob patterns) to specific LLVM releases, e.g. `7.4.*=16.0.6,8.0.*=17.0.6`
- `PHPV_LLVM_VERSION_PHP7`: Fallback LLVM version used for PHP 7.x installs when no explicit map entry matches (default: `16.0.6`)

### Directory Structure

```
~/.phpv/
├── bin/           # phpv executable
├── versions/      # Installed PHP versions
│   ├── 8.3.12/
│   ├── 8.2.24/
│   └── ...
├── cache/         # Downloaded source tarballs and build artifacts
├── version        # Current active version
└── phpv.sh       # Shell integration script
```

## PHP Configuration

Each installed PHP version includes:

- **Location**: `~/.phpv/versions/{version}/`
- **Binaries**: `bin/php`, `bin/php-cgi`, `bin/php-fpm`
- **Config**: `etc/php.ini` with sensible defaults
- **Extensions**: Common extensions pre-enabled

### Default Extensions

The following extensions are compiled by default:

- mbstring (multi-byte string support)
- opcache (opcode caching)
- curl (HTTP client)
- openssl (cryptography)
- zlib (compression)
- bcmath (arbitrary precision math)
- calendar
- exif (image metadata)
- ftp
- gd (image processing) - if available
- intl (internationalization) - if available
- soap
- sockets
- mysqli & pdo_mysql (MySQL support)
- pcntl (process control)
- shared memory extensions

## Troubleshooting

### Common Issues

#### 1. Build Failures
```bash
# Check for missing dependencies
phpv install 8.3.12

# If it fails, create Github Issue with the error message for help
```

#### 2. PHP Not Found After Switch
```bash
# Reload shell configuration
source ~/.zshrc  # or ~/.bashrc

# Or restart your terminal
```

#### 3. Permission Errors

PHPV installs everything in user space (`~/.phpv`), so no root permissions should be needed for version management. If you get permission errors, check that `~/.phpv` is writable by your user.

#### 4. Missing Extensions

If you need additional extensions, you can:
1. Modify the configure options in the `install_php_version()` function
2. Compile extensions separately after installation
3. Use package managers like PECL with the specific PHP version

Or you can request support for additional extensions via a GitHub issue.

I'm happy to help add more extensions with flexibility. 😇

### Debug Mode

For troubleshooting installation issues, you can enable debug output:

```bash
# Enable bash debug mode
bash -x ~/.phpv/bin/phpv install 8.3.12
```

## Comparison with Other Tools

| Feature | PHPV | phpenv | phpbrew |
|---------|------|--------|---------|
| User space | ✅ | ✅ | ✅ |
| Source compilation | ✅ | ✅ | ✅ |
| Easy setup | ✅ | ❌ | ❌ |
| Shell integration | ✅ | ✅ | ✅ |
| Automatic PATH | ✅ | ✅ | ❌ |
| No external deps | ✅ | ❌ | ❌ |
| Clean uninstall | ✅ | ❌ | ✅ |
| Backward support | ✅ | ❌ | ❌ |
| Latest PHP support | ✅ | ❌ | ❌ |
| Docker/CI friendly | ✅ | ❌ | ❌ |
| Works on Linux/macOS | ✅ | ✅ | ✅ |
| Extensible | ✅ | ❌ | ❌ |
| Automatic dependency management | ✅ | ❌ | ❌ |
| Open source (MIT) | ✅ | ✅ | ✅ |
| Clean builds | ✅ | ❌ | ❌ |
| Isolated dependencies | ✅ | ❌ | ❌ |


## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with multiple PHP versions
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Roadmap

- [ ] Support install custom extension with `phpv install-ext <extension-name>`
- [ ] Support uninstall custom extension with `phpv uninstall-ext <extension-name>`
- [ ] Support list installed extensions with `phpv list-ext`
- [ ] Support list available extensions with `phpv list-available-ext`
- [ ] Support enable/disable extension with `phpv enable-ext <extension-name>` and `phpv disable-ext <extension-name>`
- [ ] Support isolated build system to be used by user. For example user might download certain extension source code and automatically build it for specific PHP version using command `phpv build-ext <extension-name> <path-to-extension-source>`
- [ ] Support custom configuration file per version
- [ ] CI/CD for automated testing
- [ ] Automatic detection of available PHP versions from php.net
- [ ] Support for PHP extensions management
- [ ] Integration with Apache
- [ ] Integration with Nginx
- [ ] Integration with Caddy
- [ ] Integration with composer for project-specific PHP versions
- [ ] PECL extension management per PHP version
- [ ] Pre-compiled binary downloads for common distributions