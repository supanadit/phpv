# PHPV - PHP Version Manager

A simple PHP version manager for Linux/Unix systems, similar to `pyenv` and `nvm`. PHPV allows you to install, manage, and switch between multiple PHP versions in user space without requiring root privileges for version management.

## Installation

PHPV is designed to be as easy to install as NVM. Just run this command:

```bash
curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash
```

Or if you prefer wget:

```bash
wget -qO- https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash
```

This will:
- Download PHPV to `~/.phpv/`
- Set up shell integration automatically
- Add the `phpv` command to your PATH
- Configure your shell profile (`.bashrc`, `.zshrc`, etc.)

After installation, restart your terminal or run `source ~/.bashrc` (or `source ~/.zshrc` for ZSH).

### Manual Installation

If you prefer to install manually or are testing from source:

```bash
git clone https://github.com/supanadit/phpv.git
cd phpv
./install.sh
```

## Features

- 🚀 Install multiple PHP versions from source
- 🔄 Switch between installed versions instantly
- 🏠 User space installation (no root required for version management)
- 💾 Automatic PATH management
- 🧹 Clean uninstallation of versions
- 📦 Automatic dependency detection and guidance
- 🎨 Colored output for better readability
- 🛠️ Shell integration for bash, and zsh
- 📅 Day one latest PHP support
- 🆓 Open source (MIT License)
- 🐳 Compatible with Docker and CI environments
- 🐧 Support all linux distros (Distro agnostic)
- 🧩 Easily extensible for custom configurations
- 🧱 Isolated dependency management for each PHP version

## PHP Versions Supported

- PHP 8.5.x ( Dev Preview, Coming Soon )
- PHP 8.4.x ( Coming Soon )
- PHP 8.3.x
- PHP 8.2.x
- PHP 8.1.x
- PHP 8.0.x
- PHP 7.x.x

### Deprecated PHP Versions Supported ( Not Recommended )

These versions are deprecated and not recommended for use in production environments. They are provided for legacy support and testing purposes only. Most of dependencies for these versions are no longer maintained and even not available to be downloaded. 

I understand that some users may still need these versions for specific use cases, such as maintaining legacy applications or testing compatibility. So I will try my best to make it work. If you need any of these versions, please create a Github Issue.

- PHP 5.0.x below ( EOL, not recommended )
- PHP 4.x.x below ( EOL, not recommended )
- PHP 3.x.x below ( EOL, not recommended )
- PHP 2.x.x below ( EOL, not recommended )
- PHP 1.x.x below ( EOL, not recommended )

## How It Works

PHPV works similarly to NVM (Node Version Manager). After installation:

1. **Automatic PATH Management**: When you run `phpv use <version>`, PHPV automatically updates your PATH to use the specified PHP version
2. **Shell Integration**: The `phpv` command is available in all your shell sessions
3. **No Manual Configuration**: Unlike some version managers, you don't need to manually source scripts or modify your PATH

### Example Session

```bash
# Install a PHP version
phpv install 8.3.12

# Switch to it (PATH is automatically updated)
phpv use 8.3.12

# Check current version
phpv current
# Output: Current: 8.3.12 (PHP 8.3.12)

# Verify PHP binary
which php
# Output: /home/user/.phpv/versions/8.3.12/bin/php

php -v
# Output: PHP 8.3.12 (cli) ...

# Switch back to system PHP
phpv use system
```

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

## PHP Configuration

Each installed PHP version includes:

- **Location**: `~/.phpv/versions/{version}/`
- **Binaries**: `bin/php`, `bin/php-cgi`, `bin/php-fpm`
- **Config**: `etc/php.ini` with sensible defaults
- **Extensions**: Common extensions pre-enabled

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
- [ ] Support MacOS (Apple Silicon and Intel, I don't have Mac to test it, so PR is welcome. Sorry 😅)