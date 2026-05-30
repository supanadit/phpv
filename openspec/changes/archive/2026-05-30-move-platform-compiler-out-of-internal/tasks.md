## 1. Move compiler domain types to domain/advisor.go

- [x] 1.1 Add `CompilerType` (`gcc`/`zig`) as a string type and `CompilerInfo` struct (Type, Path, Name, Version, Available, AutoDownload fields) to `domain/advisor.go`. These are pure data types with zero framework imports.

## 2. Merge compiler selection logic into advisor use case

- [x] 2.1 Add compiler selection methods to `advisor/service.go`: `GetRequiredCompilerForPHP(phpVersion, forceCompiler)`, `GetEffectiveCompilerForPHP(phpVersion)`, `IsCompilerAvailable(compilerType)`, `UsesZigForPHP(phpVersion)`, `GetZigTarget()`, `GetZigTargetForGlibc(glibcVersion)`. Move logic from `internal/compiler/service.go` preserving behavior exactly.

- [x] 2.2 Add `GetCompilerReadiness(phpVersion string) (domain.CompilerInfo, error)` to the `AdvisorRepository` interface in `advisor/service.go`. The disk implementation returns compiler availability per PHP version.

- [x] 2.3 Implement `GetCompilerReadiness` in `internal/repository/disk/advisor.go`. The implementation checks gcc and zig availability (using existing build-tool checks), applies PHP version rules, and returns `domain.CompilerInfo`.

- [x] 2.4 Update `advisor.Service` to expose compiler methods as passthroughs to the repository (consistent with existing `Check()` pattern).

## 3. Merge platform package-name mapping into advisor infrastructure

- [x] 3.1 Add a private `getPackageName(tool, pkgMgr string) string` helper to `internal/repository/disk/advisor.go`, absorbing the package-name mapping table from `internal/platform/service.go`. Merge the existing call site (`r.platform.GetInstallSuggestion(name)`) to use this helper directly.

- [x] 3.2 Remove the `platform *platform.PlatformService` field from the `AdvisorRepository` struct in `internal/repository/disk/advisor.go`. Replace its usage in `Check()` with a direct call to `utils.DetectOSInfo()` + the new private helper.

## 4. Update Terminal to use Advisor for compiler info

- [x] 4.1 Remove the `Compiler *compiler.CompilerService` field from `TerminalHandler` struct in `internal/terminal/handler.go`. Remove the `"github.com/supanadit/phpv/internal/compiler"` import.

- [x] 4.2 Update `NewHandler()` in `internal/terminal/handler.go` to not construct a `CompilerService` — remove the `compiler.NewCompilerService(...)` call. The handler now takes an `*advisor.Service` as the 5th parameter.

- [x] 4.3 Update `DoctorV2()` in `internal/terminal/handler_system.go` to get compiler per major version through Advisor's `GetCompilerReadiness()` instead of `h.Compiler.GetEffectiveCompilerForPHP()`.

- [x] 4.4 Update `DoctorV2ResultCompiler()` in `internal/terminal/handler_system.go` to use `h.Advisor.GetEffectiveCompilerForPHP()` instead of `h.Compiler`.

- [x] 4.5 Update `cobra_tools.go` to use `domain.CompilerTypeZig` (from domain package) instead of `compiler.CompilerTypeZig`. Remove `"github.com/supanadit/phpv/internal/compiler"` import.

## 5. Delete old packages

- [x] 5.1 Delete `internal/compiler/service.go` and the `internal/compiler/` directory.

- [x] 5.2 Delete `internal/platform/service.go` and the `internal/platform/` directory.

## 6. Verify

- [x] 6.1 Run `go build ./...` to confirm the project compiles with zero errors.

- [ ] 6.2 Run `go test ./...` — 3 pre-existing test failures confirmed (2 in `internal/repository/disk`, 1 in `internal/terminal`, 2 in `internal/repository/memory`). All pre-existing before this change.
