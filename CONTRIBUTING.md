# Contributing to phpv

Thank you for your interest in contributing to phpv!

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Git
- A working Go development environment

### Development Setup

1. Fork the repository
2. Clone your fork:

```bash
git clone https://github.com/YOUR_USERNAME/phpv.git
cd phpv
```

3. Add the upstream remote:

```bash
git remote add upstream https://github.com/supanadit/phpv.git
```

4. Create a feature branch:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix-name
```

## Development Workflow

### 1. Code Style

- Follow Go's standard formatting (`go fmt`)
- Run `go vet` before committing
- Follow existing naming conventions in the codebase

### 2. Writing Code

phpv follows Clean Architecture with these layers:

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **Delivery** | `app/` | CLI commands, UI output |
| **Usecase** | `internal/terminal/` | Business logic, no UI |
| **Domain** | `domain/` | Pure data types |
| **Repository** | `internal/repository/` | Data access |

Key rules:
- Domain types must be pure data (no business logic)
- Business logic goes in service packages
- `internal/terminal/` must NOT contain `fmt.Print*`
- Return `(value, err)` for all functions that can fail

### 3. Testing

All new features should include tests:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test -v ./internal/terminal/...

# Run tests with race detection
go test -race ./...
```

Test coverage targets:
- Core packages: 60%+ coverage
- New features: Add tests alongside implementation

### 4. Running the Application

```bash
# Build
go build -o phpv ./app/phpv.go

# Run directly
go run ./app/phpv.go --help

# Run with verbose logging
go run ./app/phpv.go -x install 8.4
```

### 5. Committing

We use conventional commits. Format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test changes
- `refactor`: Code refactoring
- `chore`: Build/dependency changes

Examples:

```
feat(bundler): add parallel dependency resolution
fix(terminal): handle missing version gracefully
docs(readme): add troubleshooting section
test(handler): add tests for Use command
```

### 6. Submitting Changes

1. Ensure all tests pass:

```bash
go fmt ./...
go vet ./...
go test ./...
```

2. Commit your changes with a clear message

3. Push to your fork:

```bash
git push origin feature/your-feature-name
```

4. Open a Pull Request against `main`

## Project Structure

```
phpv/
├── app/                    # Main entry point (CLI + DI wiring)
├── bundler/               # Bundler orchestrator (interface)
├── assembler/             # Dependency resolution service
├── forge/                 # Build service (interface)
├── download/              # Download service (interface)
├── source/                # Source version management (interface)
├── unload/                # Archive extraction (interface)
├── advisor/               # System package detection (interface)
├── flagresolver/          # Configure flag resolver
├── silo/                  # Storage repository (interface)
├── pattern/               # URL pattern registry
├── shim/                   # Shim script generator
├── domain/                 # Domain entities
├── internal/
│   ├── terminal/          # Usecase layer (business logic)
│   ├── repository/        # Data access implementations
│   │   ├── memory/        # In-memory repos
│   │   ├── disk/          # Disk-based repos
│   │   └── http/          # HTTP repos
│   └── utils/             # Utility functions
└── .github/
    └── workflows/         # CI/CD workflows
```

## Adding New Features

### Adding a New CLI Command

1. Add method to `internal/terminal/service.go` (usecase interface)
2. Implement method in `internal/terminal/handler.go` (business logic only)
3. Add cobra command to `app/phpv.go` (UI output)
4. Add tests in `internal/terminal/handler_test.go`

### Adding a New Package

1. Define interface in service package (e.g., `mypackage/service.go`)
2. Implement in `internal/repository/memory/` or `internal/repository/disk/`
3. Register in fx wiring in `app/phpv.go`
4. Add URL patterns to `pattern/defaults.go` if applicable

### Adding Tests

Create `*_test.go` files alongside the code they test:

```go
package mypackage

import (
    "testing"
)

func TestMyFunction(t *testing.T) {
    // test code
}
```

## Reporting Issues

When reporting issues, please include:

- phpv version (`phpv version`)
- Operating system and version
- Go version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs (use `--verbose` flag)

## Questions?

Feel free to:
- Open an issue for questions
- Check existing issues and discussions
