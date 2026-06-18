## Why

Three root-level packages — `pattern/`, `flagresolver/`, and `unload/` — exist outside the `openspec/config.yaml` module map. Each duplicates responsibilities already assigned to documented modules: URL pattern matching (Source/Silo), compiler flag resolution (Forge), extension validation (Extension), and archive extraction (Silo). They also violate Clean Architecture: `pattern/` imports `internal/config` and `internal/utils` (use-case importing internal/), and infrastructure packages like `internal/repository/memory/patterns.go` import `pattern/` (infrastructure importing use-case). Consolidating these removes 8 architecture violations and restores the module map as the single source of truth.

## What Changes

- **Merge `pattern/` into `source/`**: Move `PatternRepository` interface, `Service`, and `BuildURL`/`MatchPattern` methods to `source/service.go`. Remove `config.Get()` call — inject OS/arch as parameters instead. **BREAKING** for consumers: `pattern.PatternRepository` → `source.PatternRepository`, `pattern.Service` → `source.Service`.
- **Merge `flagresolver/` into `forge/` and `extension/`**: Move compiler flag logic (`CStdRule`, `CXXFlagsFromCFlags`, `GetCompilerStdRule`, `GetCompilerFlags`, `COnlyWarnings`) to `forge/service.go`. Move extension validation methods (`GetExtensionDef`, `IsExtensionValidForPHPVersion`, `GetConflictingExtensions`, `GetExtensionDependency`, `GetExtensionDependencyWithVersion`, `ValidateExtensions`, `CheckExtensionConflicts`) to `extension/service.go`. **BREAKING** for consumers.
- **Merge `unload/` into `silo/`**: Move `UnloadRepository` interface and `Service` to `silo/service.go`. **BREAKING** for consumers.
- **Fix `bundler/service.go`**: Remove `internal/utils` import — replace `utils.Logger` with a domain-level logger interface.
- **Delete `pattern/`, `flagresolver/`, `unload/`**: Remove all three shadow module directories.

## Capabilities

### New Capabilities
<!-- None — this change consolidates existing functionality, not introducing new capabilities. All four target modules already exist in config.yaml. -->

### Modified Capabilities

Delta specs document requirement changes to existing modules:

- `source`: Gains `PatternRepository` interface with URL pattern matching and `BuildURL`/`BuildURLs`/`BuildURLByType` methods. OS/arch passed as parameters (no longer read from config global).
- `forge`: Gains `CStdRule` struct, `CXXFlagsFromCFlags`/`CXXFlagsFromCFlagsWithStd` functions, `COnlyWarnings` constant, `GetCompilerStdRule` and `GetCompilerFlags` methods on the repository interface.
- `extension`: Gains `GetExtensionDef`, `IsExtensionValidForPHPVersion`, `GetConflictingExtensions`, `GetExtensionDependency`, `GetExtensionDependencyWithVersion`, `ValidateExtensions`, `CheckExtensionConflicts` methods on the repository interface. Gains sentinel errors `ErrUnknownExtension` and `ErrExtensionConflict`.
- `silo`: Gains `UnloadRepository` interface with `Unpack(source, destination) (*domain.Unload, error)` method.

## Impact

- **Deleted**: `pattern/`, `flagresolver/`, `unload/` directories (3 packages)
- **Modified**: `source/service.go`, `forge/service.go`, `extension/service.go`, `silo/service.go`, `bundler/service.go`
- **Infrastructure consumers updated**: `internal/repository/memory/patterns.go`, `internal/repository/memory/flags.go`, `internal/repository/disk/bundler.go`, `internal/repository/disk/bundler_packager.go`, `internal/repository/disk/bundler_php.go`, `internal/repository/disk/forge.go`, `internal/repository/disk/advisor.go`
- **Composition root updated**: `app/phpv.go`
- **Tests updated**: memory test files, affected test files
- **Non-goals**: Fixing `internal/config/` or `internal/utils/` — those are separate violations. Not modifying domain types (no new structs needed). Not changing any runtime behavior.
