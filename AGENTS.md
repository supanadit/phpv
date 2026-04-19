# Agents

## Build & Test

```bash
go build -o phpv ./app/phpv.go   # Build binary
go test ./...                    # Run all tests
go fmt ./... && go vet ./...      # Format and lint
```

Entry point: `app/phpv.go` uses `go.uber.org/fx` for dependency injection.

## Architecture

Clean Architecture layers:
- `app/` — CLI entry point, DI wiring (fx)
- `internal/terminal/` — Usecase/business logic (NO `fmt.Print*` here)
- `domain/` — Pure data types (no logic)
- `internal/repository/` — Data access (disk/, memory/, http/)
- Service interfaces live in root packages (`bundler/`, `assembler/`, `forge/`, etc.)

## Conventions

- Return `(value, err)` for all fallible functions
- Domain types must be pure data
- Business logic goes in service packages, not in `internal/terminal/`
- Tests: `*_test.go` alongside the code they test

## Test Failures

Some tests may fail on a fresh machine (e.g., `flex` not installed, no PHP default set). These are environmental, not code issues. Run tests with `-v` to see which package fails.

## Running

```bash
go run ./app/phpv.go --help       # Direct run
./phpv --help                     # Built binary
```