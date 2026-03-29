# phpv - PHP Version Manager

phpv is a PHP Version Manager written in Go. It downloads, compiles, and manages multiple PHP versions from source on the same system, similar to phpbrew or nvm.

## Project Status: IN DEVELOPMENT

**Current Focus:** Terminal interface - CLI commands for user interaction

### What's Working
- fx dependency injection wiring in `app/phpv.go`
- Bundler orchestration service (interface in `bundler/service.go`, implementation in `internal/repository/disk/bundler*.go`)
- Multi-strategy build support in forge (configure/make, cmake, make-only, autogen)
- System package detection for executables via `which` and libraries via `pkg-config`/header checks
- HTTP download with resume support
- Dependency graph resolution via assembler
- **Terminal interface with cobra CLI** (`internal/terminal/`)

### Known Issues / In Progress
- **libxml2 download fails**: gnome.org may return redirect/HTML instead of tar.xz
- **System fallback for libraries**: When download/build fails, system fallback uses pkg-config/header detection

### Planned Features
- **Multi-platform URL patterns**: Different download URLs per OS (Linux, macOS, Windows) for packages like PHP. Currently only Linux is supported.

## Project Structure

- `app/` - Main entry point (phpv.go) with fx wiring and cobra CLI delivery layer
- `domain/` - Domain entities (Forge, Source, Version, Download, URLPattern, Silo, Dependency, DependencyGraph, Package, VersionConstraint, Bundler types)
- `bundler/` - BundlerRepository interface and BundlerServiceConfig
- `assembler/` - Assembler service - dependency graph resolution
- `forge/` - Build service - orchestrates PHP compilation from source (multi-strategy support)
- `download/` - Download service - HTTP downloads with resume support
- `source/` - Source version management - retrieves available PHP versions
- `unload/` - Archive extraction service (tar.gz, tar.xz, zip)
- `pattern/` - URL pattern registry - maps package names/versions to download URLs
- `advisor/` - Advisory service (determines system vs build-from-source)
- `flagresolver/` - Configure flag resolver service
- `silo/` - Silo repository service
- `internal/terminal/` - Terminal usecase layer (business logic, no UI)
  - `service.go` - TerminalService interface
  - `handler.go` - TerminalService implementation
- `shim/` - Shim script generator for PHP binaries
- `internal/utils/` - Utility functions (constraint matching, version parsing)
- `internal/repository/` - Data access implementations
  - `memory/` - In-memory repository (package definitions, source data)
  - `disk/` - Disk-based implementations (forge, bundler, advisor, silo, unloader)
  - `http/` - HTTP download implementation

## Key Technologies

- Go 1.25.3
- uber-go/fx for dependency injection
- spf13/cobra for CLI command handling
- Dependencies: viper (config), afero (filesystem abstraction), xz (compression), mapstructure, go-toml

## Architecture

Follow Clean Architecture / Hexagonal Architecture patterns with three main layers:

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Repository** | `internal/repository/*` | Data access implementations |
| **Usecase** | `internal/terminal/*` | Business logic (no UI) |
| **Delivery** | `app/*` | CLI commands + UI output |

### Layer Responsibilities

| Layer | Purpose | Examples |
|-------|---------|----------|
| `domain/` | Pure data types, no logic | `Dependency`, `Package`, `Version` structs |
| `internal/utils/` | Pure utility functions | `ParseVersion()`, `SortVersions()`, `MatchVersionRange()` |
| `internal/repository/` | Data access implementations | `disk/silo.go`, `http/download.go` |
| `bundler/` | Orchestrator service (interface) | `BundlerService.Install()`, `Orchestrate()` |
| `assembler/` | Service + interface | `AssemblerService`, `AssemblerRepository` |
| `forge/` | Build service | `ForgeService`, multi-strategy `BuildWithStrategy()` |
| `internal/terminal/` | Usecase layer (business logic only) | `TerminalHandler.Install()`, `Use()`, `ListInstalled()` |
| `app/` | Delivery layer (UI + CLI) | Cobra commands, `fmt.Print*`, `printBox()` |

### Data Flow

```
CLI Command (e.g., "phpv install 8.4")
       ↓
app/phpv.go (Delivery layer - cobra command)
       ↓
terminal.Handler.Install() (Usecase layer - business logic)
       ↓
bundlerRepo.Install() → bundler.Install()
       ↓
resolvePHPVersion("8.4") → "8.4.19" (latest 8.4.y)
       ↓
assembler.GetGraph("php", "8.4.19") → full transitive dependency graph
       ↓
For each dependency (in order):
  advisor.Check() → system available or build-from-source?
  buildPackage() → download → extract → compile OR use system package
       ↓
buildPHP() → configure → make → make install
       ↓
Forge{Prefix: "...", Env: {"LD_LIBRARY_PATH": "..."}}
       ↓
app/phpv.go prints UI output
```

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
    Name       string
    Version    string
    Constraint string
    Optional   bool
}

// ❌ WRONG - Domain entity with business logic
func (d Dependency) Matches(version string) bool {
    // BUSINESS LOGIC IN DOMAIN - FORBIDDEN
}
```

Exception: `Silo` struct contains path methods (`RootPath()`, `CachePath()`, etc.) for PHPV_ROOT structure.

Business logic belongs in:
- `internal/utils/` - Pure utility functions
- Service packages (`assembler/`, `forge/`, `bundler/`, etc.)
- `internal/terminal/` - Usecase layer (NO UI logic)

### Interface Pattern

Each service defines its repository interface in the service package:

```go
type ForgeRepository interface {
    Build(config domain.ForgeConfig) (domain.Forge, error)
    BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error)
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
    packages map[string]domain.Package
    repo     AssemblerRepository  // optional delegate
}
```

Key features:
- Resolves all transitive dependencies recursively
- Detects and prevents circular dependencies
- Supports optional dependencies via `Dependency.Optional`
- Version constraint format: `"recommendation|constraint"` (e.g., `"3.3.2|>=3.0.0,<4.0.0"`)

The `memory/` repository contains predefined package dependencies for:
- PHP (5.6 through 8.5)
- OpenSSL 1.1.x (for PHP 7.x) and 3.x (for PHP 8.x)
- Build tools: autoconf, automake, bison, flex, libtool, m4, perl, re2c
- Libraries: libxml2, oniguruma, zlib, curl

### Bundler Orchestration

The `bundler/` package provides the interface; implementation is in `internal/repository/disk/bundler*.go`:

```go
type BundlerRepository interface {
    Install(version string) (domain.Forge, error)
    Orchestrate(name, exactVersion string) (domain.Forge, error)
}

type BundlerServiceConfig struct {
    Assembler assembler.AssemblerRepository
    Advisor   advisor.AdvisorRepository
    Forge     forge.ForgeRepository
    Download  download.DownloadRepository
    Unload    unload.UnloadRepository
    Source    source.SourceRepository
    Silo      *domain.Silo
    Jobs      int
    Verbose   bool
}
```

Key behaviors:
- Version resolution: `"8"` → latest 8.x.y, `"8.4"` → latest 8.4.y, `"8.4.3"` → exact
- System package detection for executables via `which` command
- System library detection via `pkg-config` and header file checks
- Failed build steps halt entirely (no continue-on-error)

### Terminal Interface

The `internal/terminal/` package provides the usecase layer for CLI commands. It contains **business logic only, NO UI output**.

```go
type TerminalService interface {
    Install(version string, verbose bool) (domain.Forge, error)
    Use(constraint string) (*UseResult, error)
    SetDefault(constraint string) error
    GetDefault() (string, error)
    ListInstalled() ([]string, error)
    ListAvailable() ([]domain.Source, error)
    Which() (string, error)
}

type TerminalHandler struct {
    BundlerRepo bundler.BundlerRepository
    Silo        *disk.SiloRepository
    Source      source.SourceRepository
}
```

**Important**: All `fmt.Print*` statements belong in `app/phpv.go` (Delivery layer), NOT in `internal/terminal/` (Usecase layer).

### URL Pattern Registry

The `pattern/` package provides URL templates for downloading software:

- PHP (all versions from 4.x to 8.x)
- zlib, re2c, perl, autoconf, automake, bison
- cmake, curl, flex, libtool, libxml2, m4
- oniguruma, openssl (1.x and 3.x)

### Build Strategies

The forge service supports multiple build strategies (via `domain.BuildStrategy`):

| Strategy | Packages | Method |
|----------|----------|--------|
| `StrategyCMake` | cmake | cmake → make → make install |
| `StrategyConfigureMake` | openssl, php, libxml2, oniguruma, curl | ./configure → make → make install |
| `StrategyMakeOnly` | zlib, m4, autoconf, automake, bison, flex, libtool, perl | make → make install |
| `StrategyAutogen` | autoreconf packages | ./autogen.sh → ./configure → make → make install |

## CLI Commands

```bash
phpv install <version>   # Install PHP version (e.g., 8.5, 8.4, 8.4.0)
phpv use <version>       # Generate shims for PHP version
phpv default <version>   # Set default PHP version
phpv versions            # List installed PHP versions
phpv list                # List available PHP versions from source
phpv which               # Show path to current (default) PHP
```

### Use Command and Shim System

The `use` command generates shim scripts in `$PHPV_ROOT/bin/`:

- Shim scripts wrap PHP binaries and set environment variables
- Supported shims: `php`, `phpize`, `php-config`, `php-cgi`
- After running `phpv use <version>`, add `$PHPV_ROOT/bin` to your PATH

## Build Commands

```bash
# Build binary
go build -o phpv ./app/phpv.go

# Run CLI
./phpv install 8.4
./phpv use 8.4
./phpv versions
./phpv list
./phpv which

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test -v ./domain/...

# Run go fmt and vet
go fmt ./... && go vet ./...
```

## Configuration

- Default root directory: `$HOME/.phpv`
- Environment variable: `PHPV_ROOT`
- Viper is used for configuration management with automatic env reading
- fx provides dependency injection with `-x` flag for verbose logging

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
├── bin/              # Shim binaries (php, php-cgi, phpize, etc.)
├── build-tools/      # Shared build tools (m4, autoconf, etc.) across PHP versions
│   └── {pkg}/{ver}/
├── cache/            # Downloaded archives
│   └── {pkg}-{ver}.tar.{gz|xz}
├── default           # Default PHP version file
├── sources/          # Extracted source code
│   └── {pkg}/{ver}/
└── versions/         # Installed PHP versions
    └── {php-version}/
        ├── dependency/  # Isolated dependencies for this PHP version
        │   └── {pkg}/{ver}/
        └── output/      # PHP installation prefix
```

## fx Wiring (app/phpv.go)

The main entry point uses uber-go/fx for dependency injection:

```go
opts := []fx.Option{
    fx.WithLogger(func() fxevent.Logger { return &silentLogger{} }),  // or fxevent.DefaultLogger for -x
    fx.Provide(
        NewSiloRepository,       // *disk.SiloRepository
        NewSourceRepository,      // source.SourceRepository (memory)
        NewDownloadRepository,    // download.DownloadRepository (http)
        NewUnloadRepository,      // unload.UnloadRepository (disk)
        NewAdvisorRepository,     // advisor.AdvisorRepository (disk)
        NewAssemblerRepository,   // assembler.AssemblerRepository (memory)
        NewForgeRepository,       // forge.ForgeRepository (disk)
        NewFlagResolverRepository, // domain.FlagResolverRepository (memory)
        NewBundlerServiceConfig,  // bundler.BundlerServiceConfig
    ),
    fx.Invoke(run),
}
```

The `BundlerRepository` and `TerminalHandler` are created manually in the `run()` function, not through fx injection.

## Repository Implementations

| Repository | Interface Location | Implementation Location |
|------------|-------------------|------------------------|
| `SiloRepository` | `silo/service.go` | `internal/repository/disk/` |
| `SourceRepository` | `source/service.go` | `internal/repository/memory/` |
| `DownloadRepository` | `download/service.go` | `internal/repository/http/` |
| `UnloadRepository` | `unload/service.go` | `internal/repository/disk/` |
| `AdvisorRepository` | `advisor/service.go` | `internal/repository/disk/` |
| `AssemblerRepository` | `assembler/service.go` | `internal/repository/memory/` |
| `ForgeRepository` | `forge/service.go` | `internal/repository/disk/` |
| `BundlerRepository` | `bundler/service.go` | `internal/repository/disk/` |
| `FlagResolverRepository` | `domain/flagresolver.go` | `internal/repository/memory/` |
| `TerminalService` | `internal/terminal/service.go` | `internal/terminal/handler.go` |

## Common Development Tasks

### Adding a new CLI command

1. Add method to `internal/terminal/service.go` (usecase interface)
2. Implement method in `internal/terminal/handler.go` (business logic only, NO `fmt.Print*`)
3. Add cobra command to `app/phpv.go` in `run()` function (UI output + call handler)

Example:
```go
// internal/terminal/handler.go (usecase - business logic only)
func (h *TerminalHandler) NewCommand() (Result, error) {
    // business logic here
    return result, nil  // Return data, don't print
}

// app/phpv.go (delivery - UI output)
newCmd := &cobra.Command{
    Use:   "new",
    RunE: func(cmd *cobra.Command, args []string) error {
        result, err := handler.NewCommand()
        if err != nil {
            return err
        }
        fmt.Println(result)  // UI output here
        return nil
    },
}
```

### Adding a new URL pattern

1. Add the pattern to `pattern/defaults.go` or `pattern/registry.go`
2. Follow the existing pattern structure with Constraint and Template
3. Test with a specific version

### Adding a new package to assembler

1. Add package data to `internal/repository/memory/assembler.go`
2. Follow the existing structure with Package, Default, and Constraints
3. Use version constraint format: `"recommendation|constraint"`
4. Mark optional dependencies with `Optional: true`

### Adding multi-platform URL support

When adding support for a new platform (Linux, macOS, Windows):

1. Add new URL patterns for the package in `pattern/defaults.go` with platform-specific constraints
2. Update `pattern/registry.go` to detect OS and return appropriate URL patterns
3. Update `bundler/` package to handle platform-specific build requirements if needed

### Adding system detection for a library

1. Update `internal/repository/disk/advisor.go:checkSystemLibrary()`
2. Uses `pkg-config` for library detection
3. Uses header file checks for additional validation
4. Examples:
   - libxml2: check `/usr/include/libxml2/` or `pkg-config --exists libxml-2.0`
   - openssl: check `/usr/include/openssl/` or `pkg-config --exists openssl`

### Debugging

Use `-x` flag to see full fx dependency graph:
```bash
go run app/phpv.go -x
```
