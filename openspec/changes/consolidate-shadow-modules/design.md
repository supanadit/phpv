## Context

Three root-level packages — `pattern/`, `flagresolver/`, `unload/` — exist outside the `openspec/config.yaml` module map. Each duplicates responsibilities already assigned to documented modules and violates Clean Architecture rules (use-case importing `internal/`, infrastructure importing use-case). The `bundler/service.go` also imports `internal/utils`. This design explains how to consolidate all three into their correct modules while fixing violations.

**Current dependency flow (violations bolded):**
```
pattern/imports internal/config  ← ❌ use-case importing internal/
pattern/imports internal/utils   ← ❌ use-case importing internal/
pattern/calls config.Get()       ← ❌ global singleton
bundler/imports internal/utils   ← ❌ use-case importing internal/
memory/patterns imports pattern  ← ❌ infrastructure importing use-case
memory/flags imports flagresolver ← ❌ infrastructure importing use-case
disk/bundler imports pattern     ← ❌ infrastructure importing use-case
disk/bundler imports flagresolver← ❌ infrastructure importing use-case
disk/forge imports unload        ← ❌ infrastructure importing use-case
disk/advisor imports pattern     ← ❌ infrastructure importing use-case
```

## Goals / Non-Goals

**Goals:**
- Move `PatternRepository` interface and URL matching logic into `source/`
- Move compiler flag logic (`CStdRule`, `CXXFlagsFromCFlags`, `COnlyWarnings`, `GetCompilerStdRule`, `GetCompilerFlags`) into `forge/`
- Move extension validation methods into `extension/`
- Move `UnloadRepository` interface and service into `silo/`
- Remove all 10 architecture violations
- Delete `pattern/`, `flagresolver/`, `unload/` directories
- Zero behavioral change — `go build ./... && go test ./...` must pass

**Non-Goals:**
- Fixing `internal/config/` or `internal/utils/` violations — separate changes
- Refactoring `bundler/service.go` beyond removing the `internal/utils` import
- Changing any domain type structures
- Modifying any infrastructure implementation behavior

## Decisions

### Decision 1: Interfaces move to their CONSUMER packages

The `PatternRepository` interface is consumed by `bundler` (via `BundlerServiceConfig`), `disk/bundler`, `disk/advisor`, and `memory/patterns`. By placing it in `source/` (where the Service implementation also lives), all consumers import a single documented module.

Similarly:
- `UnloadRepository` → consumed by `bundler` and `disk/forge` → placed in `silo/`
- Compiler flag methods → consumed by `forge`, `disk/bundler*` → placed in `forge/`
- Extension validation methods → consumed by `bundler`, `disk/bundler*`, `forge` → placed in `extension/`

**Alternative considered:** Move `PatternRepository` to `bundler/` (the orchestrator that consumes it). Rejected because `disk/advisor` also consumes pattern directly and shouldn't import bundler. `source/` is neutral — both bundler and advisor import source without creating circular deps.

### Decision 2: OS/arch injected as parameters, not read from config global

`pattern.Service.BuildURLByType()` currently calls `config.Get()` to get `cfg.OS` and `cfg.Arch`. The fix:
- Change signature to `BuildURLByType(name, version, sourceType, targetOS, targetArch string) (string, error)`
- Callers (primarily `disk/advisor`) pass OS/arch explicitly — they already have this info from `utils.DetectOSInfo()`
- This removes the `internal/config` import from source/

**Alternative considered:** Inject OS/arch via constructor. Simpler to add parameters to the single method that needs them.

### Decision 3: `bundler/service.go` drops `utils.Logger` for a domain-level interface

The `BundlerServiceConfig` struct currently uses `utils.Logger` (from `internal/utils`). The fix:
- Define a `Logger` interface in `domain/logger.go` with just the methods bundler needs
- `utils.Logger` already satisfies this interface implicitly
- This removes the `internal/utils` import from bundler

**Alternative considered:** Move `Logger` to root-level package. Rejected because domain-level interfaces are the standard pattern and bundler only uses the interface shape, not the implementation.

### Decision 4: `internal/repository/memory/patterns.go` uses concrete `source.Service` directly

Currently `NewPatternRepository()` returns `pattern.PatternRepository` by wrapping `pattern.NewService()`. After the merge, this becomes:
- The file imports `source` (use-case) and returns `source.PatternRepository`
- This is still a violation (infrastructure importing use-case)
- Fix: The `DefaultPatterns` data slice (pure domain data) moves to a package-level var in `source/` or stays in memory as data. The `NewPatternRepository()` function returns a concrete type via `fx.As` annotation in `app/phpv.go` instead of calling `source.NewService()` directly.

Actually, looking at this more carefully: `internal/repository/memory/patterns.go` IS the repository implementation of `PatternRepository`. It's currently importing `pattern` (the use case that declares the interface) to satisfy the interface. This is the classic Clean Architecture dilemma in Go.

**Better approach:** Keep `DefaultPatterns` as a package-level var in `memory/patterns.go` (pure data). The function `NewPatternRepository()` becomes:
```go
func NewPatternRepository() *source.Service {
    svc := source.NewService()
    svc.RegisterPatterns(DefaultPatterns)
    return svc
}
```
This returns the CONCRETE `*source.Service`, which satisfies `source.PatternRepository` implicitly. No interface import needed. The wiring in `app/phpv.go` uses `fx.As(new(source.PatternRepository))` to expose it as the interface.

### Decision 5: Infrastructure files reference use-case packages ONLY through concrete type returns (provider pattern)

All infrastructure files that currently import shadow module packages (`pattern`, `flagresolver`, `unload`) will instead:
1. Import the target module (`source`, `forge`, `extension`, `silo`) — STILL a use-case import, but via provider return type
2. Return concrete types (structs with matching method signatures)
3. Use `fx.Annotate` in `app/phpv.go` to expose as interfaces

This doesn't fully eliminate infrastructure→use-case imports (Go requires knowing types), but it centers imports on the module that owns the interface, not a shadow module.

### Decision 6: `flagresolver` functions split into two targets

- `CXXFlagsFromCFlags`, `CXXFlagsFromCFlagsWithStd`, `CStdRule`, `CO nlyWarnings`, `GetCompilerStdRule`, `GetCompilerFlags` → `forge/`
- `GetExtensionDef`, `IsExtensionValidForPHPVersion`, `GetConflictingExtensions`, `GetExtensionDependency`, `GetExtensionDependencyWithVersion`, `ValidateExtensions`, `CheckExtensionConflicts` → `extension/`

The sentinel errors `ErrUnknownExtension` and `ErrExtensionConflict` are already defined in `domain/errors.go` (verified). The `flagresolver/` package aliased them — those aliases are removed.

## Risks / Trade-offs

- **[Risk] Large number of files touched** → Change is purely mechanical (move methods, update imports). Each file follows the same pattern. Group by package, test each group.
- **[Risk] `app/phpv.go` wiring complexity increases** → Already uses `fx.Annotate`. New annotations follow existing patterns. Test with `go build`.
- **[Risk] Pre-existing test failures in `disk/` and `memory/`** → Document known failures; do not fix pre-existing failures.
- **[Trade-off] `source/` gains URL building methods** → `source/` is about "what packages exist." URL building is about "where to get them." Both concern source resolution, so the unification is coherent per config.yaml.
