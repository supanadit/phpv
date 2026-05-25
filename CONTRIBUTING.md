# Contributing to phpv

phpv is MIT-licensed and open to contributions — bug reports, build fixes, new PHP version support, dependency resolution improvements, and better distro coverage. If you use phpv and it works on your machine, that's already a contribution worth sharing.

## Before You Code

Open an issue first. Especially for features — phpv compiles PHP from source, which means behavior that looks broken is often a dependency or platform issue, not a bug in the code. A quick discussion saves everyone time.

Check existing issues and PRs before starting. If something's already open, comment on it so we don't duplicate work.

## What's Valuable Here

This project deals with compiling PHP across diverse Linux environments. The most useful contributions aren't always code:

- **Bug reports with environment detail** — your distro, PHP version, `phpv doctor` output, and the full `--verbose` log. Build failures are almost always environment-specific, so detail matters.
- **PHP version support** — adding a new minor or patch version to the source registry.
- **Build fixes for specific distros** — Fedora, Arch, Alpine, etc. I test on Ubuntu. Patches for other distros are welcome.
- **Dependency resolution** — new system library detection, better `./configure` flag mapping, PECL extension fixes.

## Getting Set Up

You need Go 1.25+ and a Linux or macOS system with a C compiler. That's it.

```bash
git clone https://github.com/supanadit/phpv.git
cd phpv
go build -o phpv ./app/phpv.go
./phpv install 8.4
```

Run the tests:

```bash
go test ./...
```

The CI pipeline also runs integration tests — actual PHP builds on GitHub Actions. Those matter more than unit tests for this project, since the real edge cases are in compilation, not logic.

## Architecture

phpv follows Clean Architecture. If you're adding code, know where it goes:

**Domain** (`domain/`) — Pure data types. No business logic, no I/O. If you're adding a new concept (like a new package type or build strategy), start here.

**Services** (`advisor/`, `assembler/`, `bundler/`, `download/`, `extension/`, `flagresolver/`, `forge/`, `pattern/`, `shim/`, `silo/`, `source/`, `unload/`) — Business logic. Each service defines a repository interface that the implementation must satisfy.

**Repositories** (`internal/repository/disk/`, `internal/repository/memory/`, `internal/repository/http/`) — Concrete implementations of the service interfaces. `disk/` is the filesystem layer (build state, config, shims). `memory/` holds package definitions, extension mappings, and flag rules. `http/` handles downloads.

**Delivery** (`app/`, `internal/terminal/`) — CLI commands, argument parsing, output formatting. This is the only place `fmt.Print` should appear. Business logic does not live here.

Dependency injection uses Uber's fx. The wiring is in `app/phpv.go`.

A few conventions:

- Every new service interface needs a matching mock in its test file. Hand-written, no generation tools.
- Table-driven tests with `t.Run` where it makes sense, simple `got/want` where it doesn't.
- `afero` for filesystem tests — no temp directory tricks unless necessary.

## Commits

Follow what's already in the git log:

```
feat: add PHP 8.5.5 support
fix: oniguruma system not detected in fedora
refactor: centralize scattered flag logic into flagresolver
```

Short subject line. Lowercase. Prefix with `feat:`, `fix:`, or `refactor:`. If it's a substantial change, add a brief body explaining why.

## Branches and Pull Requests

- PRs target `development`, not `main`.
- One thing per PR. Small and focused is easier to review and merge.
- Rebase on `development` before opening.
- If CI fails, fix it before requesting review. The integration tests build real PHP versions — if they break, something in your change broke compilation.

## Reporting Bugs

Open a GitHub issue with:

1. Your distro and version (`cat /etc/os-release`)
2. The PHP version you tried to install
3. Full output of `phpv doctor`
4. Full output of the command that failed, run with `--verbose`

Without `--verbose` output, I'm guessing. With it, I can probably fix it.

## Code of Conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). By participating, you agree to it.