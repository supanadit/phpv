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

### Build Strategies for Legacy Versions

PHPV automatically selects the best build strategy for each PHP version:

#### For Very Old Versions (PHP 4.x, 5.x):
- **Recommended**: Docker-based builds with compatible base images
- **Docker Images**: `ubuntu:16.04` (GCC 5.4) for PHP 4.x/5.x
- **Fallback**: Native build with extensive compatibility warnings

#### For PHP 7.0-8.0:
- **Recommended**: GCC version-specific builds
- **GCC Versions**: GCC 9 for PHP 7.0-7.3 and PHP 8.0
- **Docker Images**: `ubuntu:18.04` (GCC 7.5) as alternative

#### For Modern Versions (PHP 8.1+):
- **Recommended**: Native system compiler (GCC 11+)
- **Docker Images**: `ubuntu:22.04` or `ubuntu:24.04`

### Manual Build Options

If automatic builds fail, you can manually set up compatible environments:

#### Using Docker:
```bash
# For PHP 5.6
docker run -it -v $(pwd):/workspace ubuntu:16.04 bash
# Inside container:
apt-get update && apt-get install -y build-essential wget tar gzip
# Then run PHPV from within the container

# For PHP 7.0-7.3
docker run -it -v $(pwd):/workspace ubuntu:18.04 bash
```

#### Using Specific GCC Versions:
```bash
# Install multiple GCC versions (Ubuntu/Debian)
sudo apt-get install gcc-9 g++-9
# Set environment variables before building
export CC=gcc-9 CXX=g++-9
```

#### Cross-Compilation Considerations:
- Use older Linux distributions in VMs or containers
- Consider using distcc for distributed compilation
- Apply PHP-specific patches for known compatibility issues

### Advanced: Implementing Full Docker/GCC Support

For production use, you can extend PHPV with full Docker and GCC version management:

#### Docker Implementation:
```go
// In a full implementation, DockerBuilder would:
// 1. Generate Dockerfile with appropriate base image
// 2. Copy PHP source into container context
// 3. Run build commands inside container
// 4. Extract compiled binaries back to host
```

#### GCC Version Management:
```go
// GCCVersionBuilder would:
// 1. Download/install specific GCC versions
// 2. Set CC/CXX environment variables
// 3. Configure PHP with specific compiler flags
// 4. Handle compiler-specific compatibility issues
```

#### Pre-compiled Binaries:
For organizations with many developers, consider:
- Pre-building PHP versions in CI/CD pipelines
- Storing binaries in private artifact repositories
- Using tools like `php-build` or `phpenv` for binary distribution

### Legacy PHP Project Support

For maintaining unmaintained PHP projects:

1. **Containerization**: Use Docker to create reproducible build environments
2. **Version Pinning**: Lock specific PHP/compiler versions for consistency
3. **Patch Management**: Maintain patches for security issues in old versions
4. **Gradual Migration**: Use PHPV to test compatibility with newer versions
5. **Isolated Testing**: Run legacy applications in contained environments

### Best Practices for Legacy Code

- **Security**: Apply security patches even to old versions when possible
- **Testing**: Use PHPV to test applications across multiple PHP versions
- **Documentation**: Document which PHP versions are supported and why
- **Deprecation**: Plan migration paths away from unsupported versions
- **Monitoring**: Track usage of legacy PHP versions for migration planning

## Troubleshooting

If compilation fails for older PHP versions:
1. Use an older GCC version (11-14 recommended)
2. Use Docker with an older base image
3. Check PHP's official documentation for version-specific build requirements
4. Use the provided `legacy-build-helper.sh` script for Docker-based builds

### Legacy Build Helper

A helper script is provided for manual Docker-based builds of legacy PHP versions:

```bash
# Generate Docker setup for PHP 5.6.0
./legacy-build-helper.sh 5.6.0

# This creates a Dockerfile and build script for the specified PHP version
# Follow the instructions to build PHP in an isolated container
```

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
