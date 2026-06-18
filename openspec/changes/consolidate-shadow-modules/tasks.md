## 1. Merge unload/ into silo/

- [ ] 1.1 Move `UnloadRepository` interface and `Service` struct from `unload/service.go` into `silo/service.go`. The `Service` struct wraps the interface and delegates `Unpack`.
- [ ] 1.2 Delete `unload/service.go` and the `unload/` directory.

## 2. Merge flagresolver compiler logic into forge/

- [ ] 2.1 Move `CStdRule` struct, `CXXFlagsFromCFlags`, `CXXFlagsFromCFlagsWithStd`, `COnlyWarnings` from `flagresolver/service.go` into `forge/service.go`.
- [ ] 2.2 Add `GetCompilerStdRule(phpVersion string) CStdRule` method to `forge.ForgeRepository` interface (or keep on `flagresolver.Repository` renamed as forge). Actually: add to forge's existing `Service` struct — the service already holds a `flagResolver *flagresolver.Service` field. Replace that field with direct method access after the merge.
- [ ] 2.3 Add `GetCompilerFlags(compiler string, phpVersion string) []string` method to `forge.ForgeRepository` interface.

## 3. Merge flagresolver extension validation into extension/

- [ ] 3.1 Move extension validation methods from `flagresolver/service.go` into `extension/service.go`: `GetExtensionDef`, `IsExtensionValidForPHPVersion`, `GetConflictingExtensions`, `GetExtensionDependency`, `GetExtensionDependencyWithVersion`, `ValidateExtensions`, `CheckExtensionConflicts`.
- [ ] 3.2 Add these methods to the `extension.Repository` interface. The `Service` struct wraps and delegates to the repository.
- [ ] 3.3 Delete `flagresolver/service.go` and the `flagresolver/` directory.

## 4. Merge pattern/ into source/

- [ ] 4.1 Move `PatternRepository` interface and `Service` struct from `pattern/service.go` into `source/service.go`. Include `RegisterPatterns`, `MatchPattern`, `MatchPatterns`, `MatchPatternByType`, `BuildURL`, `BuildURLs`, `BuildURLByType`.
- [ ] 4.2 Fix `BuildURLByType` signature: add `targetOS, targetArch string` parameters instead of calling `config.Get()`. Remove `internal/config` and `internal/utils` imports.
- [ ] 4.3 Delete `pattern/service.go` and the `pattern/` directory.

## 5. Fix bundler violations

- [ ] 5.1 Define a `Logger` interface in `domain/logger.go` with methods used by `BundlerServiceConfig` (e.g., `Info`, `Warn`, `Error`).
- [ ] 5.2 Replace `utils.Logger` with `domain.Logger` in `BundlerServiceConfig`. Remove `internal/utils` import from `bundler/service.go`.

## 6. Update infrastructure consumers

- [ ] 6.1 Update `internal/repository/memory/patterns.go`: change import from `pattern` to `source`. `NewPatternRepository()` returns `*source.Service` (concrete type).
- [ ] 6.2 Update `internal/repository/memory/flags.go` and `flags_test.go`: change imports from `flagresolver` to `forge` and `extension` as appropriate.
- [ ] 6.3 Update `internal/repository/disk/forge.go`: change `unload` import to `silo`.
- [ ] 6.4 Update `internal/repository/disk/bundler.go`: change `pattern`, `flagresolver`, `unload` imports to `source`, `forge`, `extension`, `silo`.
- [ ] 6.5 Update `internal/repository/disk/bundler_packager.go`: change `flagresolver` import to `forge`/`extension`.
- [ ] 6.6 Update `internal/repository/disk/bundler_php.go`: change `flagresolver` import to `forge`/`extension`.
- [ ] 6.7 Update `internal/repository/disk/advisor.go`: change `pattern` import to `source`. Update `BuildURLByType` call to pass OS/arch parameters.
- [ ] 6.8 Update `forge/service.go`: remove `flagresolver` import. The `flagResolver` field becomes unnecessary — methods are now directly on forge's own service or extension's service.

## 7. Update composition root

- [ ] 7.1 Update `app/phpv.go`: replace `pattern.NewService()` with `source.NewService()`. Replace `pattern.PatternRepository` with `source.PatternRepository`. Add `fx.Annotate` where needed to expose concrete types as interfaces.
- [ ] 7.2 Update `app/phpv.go`: replace `unload.NewService()` with `silo.NewUnloadService()`. Replace `unload.UnloadRepository` with `silo.UnloadRepository`.
- [ ] 7.3 Update `app/phpv.go`: replace `flagresolver.NewService()` references with `forge` and `extension` service constructors.

## 8. Update remaining consumers

- [ ] 8.1 Update `bundler/service.go`: change imports from `pattern`, `flagresolver`, `unload` to `source`, `forge`, `extension`, `silo`. Update struct field types in `BundlerServiceConfig`.

## 9. Verify

- [ ] 9.1 Run `go build ./...` — confirm zero compile errors.
- [ ] 9.2 Run `go test ./...` — confirm zero regressions (pre-existing failures in disk/ and memory/ are acceptable).
- [ ] 9.3 Verify no remaining imports of `pattern`, `flagresolver`, or `unload` anywhere in the codebase.
