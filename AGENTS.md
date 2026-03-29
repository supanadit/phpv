# phpv - PHP Version Manager

phpv is a PHP Version Manager written in Go. It downloads, compiles, and manages multiple PHP versions from source on the same system, similar to phpbrew or nvm.

## Project Structure

- `app/` - Main entry point (phpv.go)
- `domain/` - Domain entities (Forge, Source, Version, Download, URLPattern, Silo, Dependency, DependencyGraph, Package, VersionConstraint)
- `assembler/` - Assembler service - dependency graph resolution
- `forge/` - Build service - orchestrates PHP compilation from source
- `download/` - Download service - HTTP downloads with resume support
- `source/` - Source version management - retrieves available PHP versions
- `unload/` - Archive extraction service (tar.gz, tar.xz, zip)
- `pattern/` - URL pattern registry - maps package names/versions to download URLs
- `advisor/` - Advisory service (determines system vs build-from-source)
- `internal/utils/` - Utility functions (constraint matching, version parsing)
- `internal/repository/` - Data access implementations
  - `memory/` - In-memory repository (orchestrates the full build process)
  - `disk/` - Archive extraction implementation
  - `http/` - HTTP download implementation

## Key Technologies

- Go 1.25.3
- Dependencies: viper (config), afero (filesystem abstraction), xz (compression), mapstructure, go-toml

## Architecture

Follow Clean Architecture / Hexagonal Architecture patterns:

- `domain/` layer has NO business logic - pure data types only
- `internal/utils/` contains pure utility functions (no external dependencies)
- Service layers (`forge/`, `assembler/`, `download/`, `source/`, `unload/`) contain business logic
- Repository implementations in `internal/repository/`

### Layer Responsibilities

| Layer | Purpose | Examples |
|-------|---------|----------|
| `domain/` | Pure data types, no logic | `Dependency`, `Package`, `Version` structs |
| `internal/utils/` | Pure utility functions | `MatchVersionRange()` |
| `assembler/` | Service + interface | `AssemblerService`, `AssemblerRepository` |
| `internal/repository/` | Data access implementations | `memory/assembler.go`, `http/assembler.go` |

### Data Flow

```
ForgeConfig{Name: "php", Version: "8.3.0"}
       ↓
AssemblerService.GetGraph("php", "8.3.0")  →  full transitive dependency graph
       ↓
Advisor.Check(package)  →  system package or build from source?
       ↓
Forge.Build(package)  →  builds each package in topological order
```

### Domain Entities

- `Forge` - Represents a built PHP installation
- `ForgeConfig` - Configuration for building PHP (name, version, configure flags)
- `Source` - A software source with name, version, and download URL
- `Version` - Parsed version (major, minor, patch, suffix)
- `URLPattern` - Pattern template for generating download URLs
- `Download` - Download record with URL and destination
- `Unload` - Unpacking result (source, destination, extracted count)
- `Silo` - Cache manager
- `Dependency` - A package dependency (Name, Version, Constraint, Optional)
- `DependencyGraph` - Collection of resolved dependencies with build order
- `Package` - A package definition with name, source URL, and dependencies
- `VersionConstraint` - Version requirement string (e.g., `">=3.0.0,<4.0.0"`)

## Code Standards

### Naming Conventions

- Types: PascalCase (Forge, Source, Version)
- Interfaces: PascalCase with "Repository" suffix (ForgeRepository, BuildRepository)
- Exported functions: PascalCase
- Unexported functions: camelCase
- Variables: camelCase
- Constants: PascalCase or camelCase depending on scope
- Package names: short, lowercase, no underscores

### Error Handling

- Return `(value, err)` for all functions that can fail
- Use `fmt.Errorf("context: %w", err)` for wrapped errors
- Only use `panic` for truly unrecoverable errors (e.g., missing home directory)

### Domain Layer Rules (CRITICAL)

The `domain/` package MUST contain ONLY pure data types with NO business logic:

```go
// ✅ CORRECT - Domain entity is pure data
type Dependency struct {
    Name      string
    Version   string
    Constraint string
    Optional  bool
}

// ❌ WRONG - Domain entity with business logic
type Dependency struct {
    Name      string
    Version   string
}

func (d Dependency) Matches(version string) bool {
    // BUSINESS LOGIC IN DOMAIN - FORBIDDEN
}
```

Business logic belongs in:
- `internal/utils/` - Pure utility functions
- Service packages (`assembler/`, `forge/`, etc.)

### Interface Pattern

Each service defines its repository interface in the service package:

```go
type ForgeRepository interface {
    Build(config domain.ForgeConfig) (domain.Forge, error)
}
```

Implementations are in `internal/repository/<type>/`.

### Assembler / Dependency Graph System

The `assembler/` package provides transitive dependency resolution:

```go
type AssemblerRepository interface {
    GetGraph(name string, version string) (domain.DependencyGraph, error)
    GetDependencies(name string, version string) ([]domain.Dependency, error)
    RegisterPackage(pkg domain.Package)
}

type AssemblerService struct {
    repo AssemblerRepository
}
```

Key features:
- Resolves all transitive dependencies recursively
- Detects and prevents circular dependencies
- Supports optional dependencies via `Dependency.Optional`
- Version constraint format: `"recommendation|constraint"` (e.g., `"3.3.2|>=3.0.0,<4.0.0"`)

The `memory/` repository contains predefined package dependencies for:
- PHP (5.6 through 8.4)
- OpenSSL 1.1.x (for PHP 7.x) and 3.x (for PHP 8.x)
- Build tools: autoconf, automake, bison, flex, libtool, m4, perl, re2c
- Libraries: libxml2, oniguruma, zlib

### URL Pattern Registry

The `pattern/` package provides URL templates for downloading software:

- PHP (all versions from 4.x to 8.x)
- zlib, re2c, perl, autoconf, automake, bison
- cmake, curl, flex, libtool, libxml2, m4
- oniguruma, openssl (1.x and 3.x)

## Build Commands

```bash
# Build binary
go build -o phpv ./app/phpv.go

# Build with version flag
go build -ldflags "-X github.com/supanadit/phpv/domain.AppVersion=v1.0.0" -o phpv ./app/phpv.go

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test -v ./domain/...

# Run specific test
go test -v ./domain/... -run TestVersion

# Run go fmt
go fmt ./...

# Run go vet
go vet ./...
```

## Configuration

- Default root directory: `$HOME/.phpv`
- Environment variable: `PHPV_ROOT`
- Viper is used for configuration management with automatic env reading

## Testing

Tests exist for core packages:

- `source/service_test.go`
- `unload/service_test.go`
- `download/service_test.go`
- `internal/repository/disk/unloader_test.go`
- `internal/repository/http/download_test.go`
- `internal/repository/memory/assembler_test.go`

Use standard `go test` commands to run tests.

## PHPV_ROOT Structure

```
$PHPV_ROOT/
├── bin/          # Shim binaries (php, php-cgi, phpize, etc.)
├── cache/        # Downloaded archives
├── sources/      # Extracted source code
└── versions/     # Installed PHP versions
```

## Common Development Tasks

### Adding a new URL pattern

1. Add the pattern to `pattern/registry.go` or create a new patterns file
2. Follow the existing pattern structure with Constraint and Template
3. Test with a specific version

### Adding a new domain entity

1. Create type in `domain/` package
2. Add repository interface to relevant service package
3. Implement in `internal/repository/<type>/`

### Adding a new archive format support

1. Add extraction logic to `internal/repository/disk/`
2. Update `Unload` type if needed
3. Add tests for the new format

### Adding a new package to assembler

1. Add package data to `internal/repository/memory/assembler.go`
2. Follow the existing structure with Name, Version, URL, and Dependencies
3. Use version constraint format: `"recommendation|constraint"`
4. Mark optional dependencies with `Optional: true`
5. Add tests for transitive dependency resolution
