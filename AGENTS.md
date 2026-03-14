# AI Tooling Context for PHPV

## Project Overview

PHPV is a sophisticated PHP version manager written in Go, similar to NVM (Node) or Pyenv (Python). It enables users to install, manage, and switch between multiple PHP versions in user space without root privileges. Unlike simpler version managers, PHPV handles the complexity of PHP's diverse deployment options (Apache, NGINX, etc.) and extension management (PECL, source builds).

### Key Differentiators

- **Multi-compiler support**: LLVM, GCC, Zig Compiler, Docker-isolated builds (statically linked)
- **Automatic dependency management**: Downloads and builds all required dependencies per PHP version
- **Version-specific LLVM toolchain**: Different LLVM versions for different PHP versions (PHP 8.3+ uses LLVM 21.1.6, 8.0-8.2 uses 18.1.8, 5.x/7.x uses 15.0.6)
- **Static linking**: Produces portable, statically-linked PHP binaries
- **User-space installation**: No root privileges required

## Architecture

### Layered Architecture Pattern

```
┌─────────────────────────────────────────┐
│  Terminal/CLI Layer (internal/terminal) │
│  - Command handlers, flag parsing       │
├─────────────────────────────────────────┤
│  Service Layer (*/service.go)           │
│  - Business logic, orchestration        │
├─────────────────────────────────────────┤
│  Domain Layer (domain/)                 │
│  - Entities, value objects              │
├─────────────────────────────────────────┤
│  Repository Layer (internal/repository) │
│  - Data access abstraction              │
└─────────────────────────────────────────┘
```

### Directory Structure

```
/home/supanadit/Workspaces/Personal/Go/phpv/
├── app/phpv.go              # Main CLI entry point
├── build/                   # Build service - compiles PHP from source
├── dependency/              # Dependency management (987 lines - most complex)
│   ├── service.go          # Builds all dependencies
│   ├── mapping.go          # PHP version → dependency mappings (711 lines)
│   └── toolchain.go        # LLVM toolchain management
├── domain/                  # Domain models
│   ├── version.go          # Version struct (Major, Minor, Patch, Extra)
│   ├── dependency.go       # Dependency definitions
│   ├── toolchain.go        # ToolchainConfig for custom compilers
│   └── llvm.go             # LLVM version mappings per PHP version
├── download/                # Download service
├── internal/
│   ├── repository/memory/   # In-memory version storage (hardcoded lists)
│   ├── terminal/           # CLI handlers (version, download, build, prune)
│   └── util/               # Command execution utilities
├── prune/                   # Cleanup service
└── version/                 # Version retrieval service
```

## Key Concepts

### 1. Version Struct

```go
type Version struct {
    Major int
    Minor int
    Patch int
    Extra string  // e.g., "RC1", "alpha1"
}
```

### 2. Dependency Management

Each PHP version has specific dependency requirements defined in `dependency/mapping.go`. Dependencies include:
- **Toolchain**: LLVM/Clang (automatically downloaded)
- **Build tools**: cmake, perl, m4, autoconf, automake, libtool, re2c
- **Libraries**: zlib, libxml2, openssl, curl, oniguruma

Dependencies have transitive relationships (e.g., curl needs openssl, zlib).

### 3. Runtime Directory Structure

```
~/.phpv/
├── cache/
│   ├── sources/          # Downloaded PHP source archives
│   └── toolchains/       # Downloaded LLVM archives
├── sources/              # Extracted PHP source code
├── versions/             # Compiled PHP installations
├── dependencies/         # Built dependencies per PHP version
├── dependencies-src/     # Dependency source code
└── toolchains/           # LLVM toolchain installations
```

### 4. Custom Toolchain Support

Users can override the default LLVM toolchain via environment variables:
- `PHPV_TOOLCHAIN_CC` - C compiler
- `PHPV_TOOLCHAIN_CXX` - C++ compiler
- `PHPV_TOOLCHAIN_SYSROOT` - System root
- `PHPV_TOOLCHAIN_PATH` - Additional PATH entries
- `PHPV_TOOLCHAIN_CFLAGS` - C flags
- `PHPV_TOOLCHAIN_CPPFLAGS` - C preprocessor flags
- `PHPV_TOOLCHAIN_LDFLAGS` - Linker flags

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `list [major[.minor]]` | List available PHP versions | `phpv list`, `phpv list 8.3` |
| `download <version>` | Download PHP source code | `phpv download 8.4.14` |
| `build` or `install <version>` | Build PHP from source | `phpv build 8.3` |
| `prune` | Remove all build artifacts | `phpv prune` |
| `help`, `-h` | Show help message | `phpv help` |

**Version formats supported**: `8` (major only), `8.3` (major.minor), `8.4.14` (full)

## Development Guidelines

### Code Style

1. **Interface Segregation**: Define interfaces in the consumer package (e.g., `terminal/download.go` defines `DownloadService` interface)

2. **Dependency Injection**: Services receive their dependencies via constructors

3. **Context Usage**: All long-running operations accept `context.Context` for cancellation

4. **Error Handling**: Wrap errors with context using `fmt.Errorf("...: %w", err)`

5. **Environment Variables**: Use Viper for configuration with `PHPV_` prefix

### Adding a New Command

1. Create handler in `internal/terminal/<command>.go`:
```go
package terminal

type NewService interface {
    // Define service interface
}

type NewHandler struct {
    service NewService
}

func NewNewHandler(ctx context.Context, svc NewService) bool {
    // Parse args, check if this command matches
    // Return true if handled, false otherwise
}
```

2. Register in `app/phpv.go`:
```go
if !terminal.NewNewHandler(ctx, newSvc) {
    // Next handler
}
```

### Adding PHP Version Support

1. Update `internal/repository/memory/version.go` with new version numbers
2. If new dependency requirements, update `dependency/mapping.go`
3. If new LLVM version needed, update `domain/llvm.go`

### Adding Dependencies

Edit `dependency/mapping.go`:
1. Add dependency definition to `DependencySet`
2. Define download URL and version in dependency struct
3. Add to `GetDependenciesForVersion()` function
4. Add build logic in `dependency/service.go` if special handling needed

## Testing

Run all tests:
```bash
go test ./...
```

Run specific package:
```bash
go test ./build
go test ./download
go test ./dependency
```

Test files follow `*_test.go` pattern alongside source files.

## Build and Development

### Build the CLI

```bash
go build ./app/phpv.go
```

### Install to $GOPATH/bin

```bash
go install ./app/phpv.go
```

### Run for Development

```bash
go run app/phpv.go [command]
```

### Dependencies

Only external dependency: `github.com/spf13/viper v1.21.0` (configuration management)

## Important Files

| File | Purpose | Size |
|------|---------|------|
| `dependency/service.go` | Builds all dependencies for PHP | 987 lines |
| `dependency/mapping.go` | PHP version → dependency mappings | 711 lines |
| `internal/repository/memory/version.go` | Hardcoded PHP version lists | - |
| `build/service.go` | PHP compilation orchestration | 319 lines |
| `domain/llvm.go` | LLVM version mappings | - |

## Common Tasks for AI Assistants

### 1. Adding Support for a New PHP Version

When a new PHP version is released:
1. Add version to `internal/repository/memory/version.go`
2. Check if dependency versions in `dependency/mapping.go` need updates
3. Verify LLVM version compatibility in `domain/llvm.go`

### 2. Fixing Build Issues

If a PHP version fails to build:
1. Check `dependency/mapping.go` for correct dependency versions
2. Verify configure flags in `build/service.go` (version-specific switch statement)
3. Check environment setup in `dependency/service.go`

### 3. Adding Extension Support

To add PHP extension support:
1. Add configure flags in `build/service.go` Build() method
2. May need to add new dependencies to `dependency/mapping.go`
3. Update dependency build logic if extension requires special libraries

### 4. Custom Compiler Support

When adding compiler support:
1. Update `domain/toolchain.go` with new config options
2. Modify `build/service.go` compiler detection
3. Update `dependency/service.go` to use custom toolchain

## Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `PHPV_ROOT` | Root directory for phpv | `~/.phpv` |
| `PHP_SOURCE` | Download source (`github` or `official`) | `github` |
| `PHPV_TOOLCHAIN_CC` | Custom C compiler | (none) |
| `PHPV_TOOLCHAIN_CXX` | Custom C++ compiler | (none) |
| `PHPV_TOOLCHAIN_*` | See Custom Toolchain Support section | (none) |

## LLM-Specific Notes

1. **Version Matching**: The system uses fuzzy matching - `phpv build 8` finds the latest 8.x version. See `FindMatchingVersion()` in services.

2. **Dependency Transitivity**: When modifying dependencies, remember they can have transitive dependencies. Check the full dependency chain in `dependency/mapping.go`.

3. **LLVM Version Mapping**: PHP versions map to specific LLVM versions for compatibility. Always check `domain/llvm.go` when adding new PHP versions.

4. **Static Linking**: All builds use `--enable-static --disable-shared` for portability.

5. **Error Propagation**: Errors are wrapped with context at each layer. When debugging, trace through the error chain.

6. **Testing Strategy**: Unit tests exist for service logic. Integration testing requires actual PHP builds (time-consuming).

7. **Caching**: Downloads are cached in `~/.phpv/cache/` to avoid redundant network requests. When testing download changes, clear the cache.

8. **Architecture Pattern**: The codebase uses a clean/hexagonal architecture. Domain logic is in `domain/`, application logic in service packages, and infrastructure concerns in `internal/`.

## Questions?

For issues specific to this codebase, check:
1. The `dependency/mapping.go` for dependency-related questions
2. The `build/service.go` for compilation questions
3. The `domain/` package for data model questions
4. Existing command handlers in `internal/terminal/` for CLI pattern examples
