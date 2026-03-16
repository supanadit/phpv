# 🚀 PHPV - PHP Version Manager

**The PHP version manager PHP developers deserve.**

> _"NVM manages Node. Pyenv manages Python. **PHPV manages PHP.**"_

---

## Quickstart

```bash
# Quick Install (Recommended)
curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | bash

# Specific Version
# curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/install.sh | INSTALL_VERSION=0.1.4 bash

# Restart shell or source config
source ~/.zshrc  # or ~/.bashrc

# Use it
phpv list             # See available versions
phpv download 8.5     # For now we need download source, It might be removed after stable version which enough with `install` command
phpv install 8.5      # Install latest PHP 8.5
phpv use 8.5          # Switch to PHP 8.5 in current shell
phpv default 8.5      # Set PHP 8.5 as default
phpv versions         # List installed versions
phpv which            # Show current PHP path
```

### Alternative: Install from Source

```bash
# Install from source (requires Go 1.21+)
git clone https://github.com/supanadit/phpv.git
cd phpv
go install ./app/phpv.go
```

## The Problem: Why PHP Version Management Is Fundamentally Different

If you've worked with Node.js or Python, you know the luxury: `nvm install 20`, `pyenv install 3.12`, done. Binaries download, paths get wired, you're productive in seconds.
**PHP is not that simple.**
PHP doesn't just need a binary. It needs:

- **Version-specific compilers** — PHP 8.3 requires different LLVM toolchains than PHP 8.0
- **Dependency chains** — libxml2, openssl, curl, onigurama, each with precise version requirements
- **Static linking** — For portable binaries that work across distros
- **PECL extensions** — Compiled per-version, with their own dependency graphs
- **Web server integration** — Apache modules, PHP-FPM configs, NGINX wiring
  This is why **no true PHP version manager has existed** — until now. The bash scripts and Docker containers you've been using? They're workarounds, not solutions. They don't handle the complexity. They don't work across distros. They require root. They break when you need that one extension.

---

## The Vision: What PHPV Does Differently

**PHPV is not a script. It's not a container. It's a version manager.**
Built from the ground up in Go (not bash, not Python) because PHP ecosystem complexity demands a real programming language. Here's what makes it different:

### 🧠 **Complexity Handled, Not Hidden**

Most tools hide complexity. PHPV handles it.

- **Automatic dependency resolution** — Each PHP version knows exactly which libraries it needs and builds them from source
- **Transitive dependency chains** — When curl needs openssl and zlib, PHPV knows. When dependencies have dependencies, PHPV tracks it all
- **Version-specific LLVM toolchains** — PHP 8.3+? LLVM 21.1.6. PHP 8.0-8.2? LLVM 18.1.8. Older PHP? LLVM 15.0.6. **Automatically downloaded and configured.**

### 🏠 **True User-Space Installation**

```bash
# No sudo. No root. No system pollution.
phpv download 8.4
phpv install 8.4
phpv use 8.4
```

Your PHP versions live in `~/.phpv/`. Your system PHP stays untouched. Need five versions for testing different projects? They coexist peacefully.

### 🚀 **Day-One PHP Support**

PHP 9.x in dev? PHPV has it. New PHP version released yesterday? PHPV supports it today.
No waiting for:

- Ubuntu PPA updates
- Homebrew bottling
- Distribution package freezes
  **You're always on current PHP.**

---

## How PHPV Works: A Peek Under the Hood

The architecture is intentionally layered, following clean/hexagonal architecture principles:

```
┌─────────────────────────────────────────────────────────┐
│  CLI Layer (internal/terminal)                          │
│  Command handlers, argument parsing, user interaction   │
├─────────────────────────────────────────────────────────┤
│  Service Layer (*/service.go)                           │
│  Business logic, orchestration, build coordination      │
├─────────────────────────────────────────────────────────┤
│  Domain Layer (domain/)                                 │
│  Version structs, dependency models, toolchain config  │
├─────────────────────────────────────────────────────────┤
│  Repository Layer (internal/repository)                 │
│  Version data, source manifests, release tracking       │
└─────────────────────────────────────────────────────────┘
```

### Key Technical Decisions That Matter

| Challenge                                                   | PHPV's Solution                                                                                 |
| ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| **Different PHP versions need different compiler versions** | Automatic LLVM toolchain download per major PHP version (8.3+: LLVM 21, 8.0-8.2: LLVM 18, etc.) |
| **Dependency version conflicts**                            | Isolated dependency builds per PHP version in `~/.phpv/dependencies/`                           |
| **"Works on my machine" syndrome**                          | Statically-linked builds with `--enable-static --disable-shared` (In Progress)                  |
| **Custom compilers (Zig, custom GCC)**                      | Full toolchain override via `PHPV_TOOLCHAIN_*` environment variables                            |
| **PECL extensions**                                         | _[Planned]_ Per-version extension management with automatic dependency resolution               |

---

### First Steps

```bash
# See what PHP versions are available
phpv list
# Install a specific version
phpv install 8.4
# Install latest from a major version
phpv install 8
# See installed versions
phpv versions
```

---

## The Roadmap: Where PHPV Is Going

PHPV is already functional for core version management. But this is just the beginning:

**✅ Available Now:**

- [x] Install PHP from source (8.0+ fully supported)
- [x] Multi-compiler support (LLVM, GCC, custom toolchains)
- [x] Smart compiling strategy, use system GCC by default and fallback to LLVM if necessary
- [x] Automatic dependency resolution
- [x] Version-specific LLVM toolchains
- [x] User-space installation (no root)
- [x] Shell integration (bash, zsh, fish)
- [x] **Version switching commands** (`use`, `shell`, `default`, `versions`, `which`)
- [x] Linux support (distro-agnostic)

**🛠️ In Progress:**

- [ ] Compaction build size to reduce and delete unecessary file
- [ ] PECL extension management
- [ ] [PIE](https://github.com/php/pie) support out of the box
- [ ] PHP-FPM and web server integration
- [ ] `php.ini` management per version
- [ ] Static linking for portable binaries

**🔮 Future:**

- [ ] Cross-compilation with Zig
- [ ] Truly portable binary builds
- [ ] macOS support
- [ ] Windows support
- [ ] Pre-built binary cache
- [ ] `.phpvrc` project-level configuration

---

## Why This Matters

**The PHP ecosystem has been underserved.**
Node developers have had NVM since 2014. Python developers have had Pyenv since 2012. PHP developers? We've been manually compiling, using brittle bash scripts, or locked into platform-specific solutions.
**PHPV changes that.**
This isn't about convenience. It's about:

- **Developer productivity** — Switching PHP versions in seconds, not hours
- **Team consistency** — Same PHP version across development and production
- **Modern PHP adoption** — Day-one access to new PHP features and security fixes
- **Freedom** — Not locked into a specific distro or Docker image

---

## Philosophy

PHPV is built on these principles:

1. **Complexity should be handled, not hidden** — Users shouldn't need to understand LLVM versions, dependency chains, or static linking. But the tool should handle them correctly.
2. **User-space is sovereign** — Root access is for system administrators. Developer tools should never require `sudo`.
3. **Portability over convenience** — A static binary that works everywhere beats a shared library build that's faster to compile but fragile.
4. **Day-one support matters** — PHP releases should be available immediately, not months later through package managers.
5. **Real architecture scales** — Go over bash. Clean architecture over quick scripts. Long-term maintainability over temporary fixes.

---

## PHP Versions Supported

| Version Range     | Status           | Notes                                                  |
| ----------------- | ---------------- | ------------------------------------------------------ |
| **PHP 8.5.x**     | ✅ Stable        | Latest production PHP                                  |
| **PHP 8.4.x**     | ✅ Stable        | —                                                      |
| **PHP 8.3.x**     | ✅ Stable        | —                                                      |
| **PHP 8.2.x**     | ✅ Stable        | —                                                      |
| **PHP 8.1.x**     | ✅ Stable        | —                                                      |
| **PHP 8.0.x**     | ✅ Stable        | —                                                      |
| **PHP 7.x.x**     | 🟡 SimiSupported | End-of-life, but functional                            |
| **PHP 5.x / 4.x** | ⚠️ Legacy        | Dependencies difficult to source, use at your own risk |

**Older versions (PHP 4.x, 5.x):** These can be built, but some dependencies are no longer available from original sources. PHPV will do its best, but expect some manual intervention. Create an issue if you need these — we can find solutions together.

## Architecture Details

For those who want to understand the internals:

### Directory Structure

```
~/.phpv/
├── cache/
│   ├── sources/          # Downloaded PHP source archives (tarballs)
│   └── toolchains/       # Downloaded LLVM releases
├── sources/              # Extracted PHP source ready to build
├── versions/             # Compiled PHP installations (the binaries)
├── dependencies/         # Built libraries for each PHP version
├── dependencies-src/     # Source code for all dependencies
└── toolchains/           # Extracted LLVM toolchains
```

### Dependency Management

Each PHP version has a **dependency manifest** (`dependency/mapping.go`). For example, PHP 8.4.0 needs:

```
Toolchain:     LLVM 21.1.6
Build Tools:   cmake 3.30.2, perl 5.40.0, m4 1.4.19, autoconf 2.72,
               automake 1.17, libtool 2.4.7, re2c 3.1
Libraries:     zlib 1.3.1, libxml2 2.13.4, openssl 3.3.2,
               curl 8.10.1, oniguruma 6.9.9
```

Dependencies have **transitive relationships** (curl needs openssl, openssl needs zlib). PHPV resolves the entire graph and builds in correct order.

### Custom Toolchain Support

Want to use your own compiler? Set environment variables:

```bash
export PHPV_TOOLCHAIN_CC=zig cc
export PHPV_TOOLCHAIN_CXX=zig c++
export PHPV_TOOLCHAIN_LDFLAGS="-static"
phpv install 8.4
```

Full override capability for experimentation with Zig, GCC, Clang, or any compiler.

## Contributing

PHPV is early in its journey. There's much to build:

- **Extension management** — Design the PECL integration
- **Web server integration** — Create Apache/NGINX/Caddy modules
- **Platform support** — Help with macOS and Windows ports
- **Testing** — Add integration tests, extend coverage
- **Documentation** — Improve guides, add examples

```bash
# Standard Go project setup
git clone https://github.com/supanadit/phpv.git
cd phpv

go run app/phpv.go download 8
go run app/phpv.go build 8
go run app/phpv.go use 8
```

See [AGENTS.md](./AGENTS.md) for development context and architecture details.

## Project Status

**Current Version:** Early development, functional for core use cases  
**License:** MIT License — truly open source  
**Language:** Go (why? because PHP ecosystem complexity demands a real programming language, not bash scripts)  
**Author:** [Supan Adit Pratama](https://github.com/supanadit)

---

## The Bottom Line

**Node developers have NVM. Python developers have Pyenv. Ruby developers have Rbenv. PHP developers deserve PHPV.**
This is the PHP version manager that handles the complexity PHP ecosystem demands. Not a workaround. Not a script. Not a container. A real version manager.
**Built for PHP developers who deserve better.**

---

<p align="center">
  <strong>Star the repo if you believe PHP developers deserve better tooling.</strong>
</p>
<p align="center">
  <a href="https://github.com/supanadit/phpv/issues">Report Bug</a> •
  <a href="https://github.com/supanadit/phpv/issues">Request Feature</a> •
  <a href="https://github.com/supanadit/phpv/blob/main/AGENTS.md">Architecture Deep Dive</a>
</p>
