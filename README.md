# PHPV - PHP Version Manager

**IMPORTANT: This project is currently being rewritten in Go. The bash version is deprecated.**

A simple PHP version manager for Linux/Unix systems, similar to `pyenv` and `nvm`. PHPV allows you to install, manage, and switch between multiple PHP versions in user space without requiring root privileges for version management.

## Planned Features

- 🚀 Install multiple PHP versions from source
- 🔄 Switch between installed versions instantly
- 🏠 User space installation (no root required for version management)
- 💾 Automatic PATH management
- 🧹 Clean uninstallation of versions
- 📦 Automatic dependency detection and guidance
- 🛠️ Shell integration for bash, and zsh
- 📅 Day one latest PHP support
- 🆓 True open source ([MIT License](LICENSE))
- 🐳 Compatible with Docker and CI environments
- 🐧 Support all linux distros (Distro agnostic)
- 🧩 Easily extensible for custom configurations
  - Adding and removing PHP extensions per version from PECL or source
  - Enabling and disabling extensions per version on the fly
  - Managing `php.ini` from this tool
  - Integration with web servers (Apache, Nginx, Caddy)
- 🧱 Isolated dependency management for each PHP version
- 🪟 Windows support
- 🍏 MacOS support

## Go Implementation (Current Development)

The project is being actively developed in Go for better performance, reliability, and maintainability.

### Building from Source

```bash
git clone https://github.com/supanadit/phpv.git
cd phpv
go install ./app/phpv.go
```

### Demo

Run the demo application:

```bash
# Fast simulated builds (for testing)
go run app/phpv.go
```

## PHP Versions Supported

After this project released in stable state, each new PHP version will be supported as soon as possible. So you don't need to wait the operating system package manager to provide the latest PHP version.

- PHP 8.5.x ( Dev Preview, Coming Soon )
- PHP 8.4.x
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

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with multiple PHP versions
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.
