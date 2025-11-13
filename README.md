# PHPV - PHP Version Manager

**IMPORTANT: This project is currently being rewritten in Go. The bash version is deprecated.**

A simple PHP version manager for Linux/Unix systems, similar to `pyenv` and `nvm`. PHPV allows you to install, manage, and switch between multiple PHP versions in user space without requiring root privileges for version management.

## Go Implementation (Current Development)

The project is being actively developed in Go for better performance, reliability, and maintainability.

### Building from Source

```bash
git clone https://github.com/supanadit/phpv.git
cd phpv
go build -o phpv app/main.go
```

### Demo

Run the demo application:

```bash
# Fast simulated builds (for testing)
go run app/main.go

# Real PHP compilation (takes 10-30 minutes)
go run app/main.go --simulate=false
```

### Supported PHP Versions

PHPV supports PHP versions from 4.x through 8.x:

- **PHP 8.1+**: Full support with modern GCC (recommended)
- **PHP 8.0**: Supported but may have compatibility issues with GCC 15.x
- **PHP 7.4+**: Supported with modern GCC
- **PHP 7.0-7.3**: May have compatibility issues with GCC 15.x
- **PHP 5.x**: Very old versions, may not compile with modern GCC
- **PHP 4.x**: Legacy versions, likely incompatible with modern GCC

### Compatibility Notes

- **GCC Compatibility**: Older PHP versions (4.x, 5.x, 7.0-7.3) may not compile with GCC 15.x due to deprecated functions and stricter compilation standards.
- **Download Sources**: 
  - PHP 4.x downloads from `museum.php.net/php4`
  - PHP 5.x downloads from `museum.php.net/php5` 
  - PHP 7.x+ downloads from `php.net/distributions`
- **Build Time**: Real compilation can take 10-30 minutes depending on your system
- **System Requirements**: Requires build tools (gcc, make, etc.) and development libraries

### Troubleshooting

If compilation fails for older PHP versions:
1. Use an older GCC version (11-14 recommended)
2. Use Docker with an older base image
3. Check PHP's official documentation for version-specific build requirements

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with multiple PHP versions
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

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
