# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive README.md documentation
- CONTRIBUTING.md with contribution guidelines
- CODE_OF_CONDUCT.md for community guidelines
- Improved CI/CD workflows with multi-platform support
- Test coverage improvements for terminal handler
- Test coverage improvements for shim package
- Test coverage improvements for bundler package
- Test coverage improvements for advisor package
- Test coverage improvements for utils package

### Changed
- Updated release workflow to build for multiple platforms (Linux amd64/arm64, macOS amd64/arm64)
- Updated test workflow to run on pull requests and main branch
- Added coverage tracking workflow

### Fixed
- (None yet)

### Deprecated
- (None yet)

### Removed
- (None yet)

### Security
- (None yet)

## [0.1.0] - 2024-01-01

### Added
- Initial release
- PHP version installation from source (4.x through 8.x)
- Dependency resolution and automatic compilation
- Support for multiple build strategies (configure/make, cmake, autogen, make-only)
- System package detection (pkg-config, header checks)
- Shim system for version switching
- Shell initialization for Bash, Zsh, and Fish
- Shell completions for Bash, Zsh, Fish, and PowerShell
- Commands: install, use, default, versions, list, which, uninstall, doctor, upgrade
- Build-tools management with cleanup functionality
- HTTP download with resume support
- Archive extraction (tar.gz, tar.xz, zip)
- URL pattern registry for package downloads
- Zig compiler support for PHP < 8.0

### Features
- Multi-version PHP management from single system
- Parallel dependency builds
- Fresh install option (clean rebuild)
- Verbose and quiet output modes
- Dry-run mode for installation preview
- JSON output for scripting
- Installation script with automatic updates
