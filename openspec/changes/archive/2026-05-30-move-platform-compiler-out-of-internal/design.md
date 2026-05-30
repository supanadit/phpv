## Context

`internal/platform/` and `internal/compiler/` contain logic that, per the module map in `openspec/config.yaml`, belongs to the **Advisor** module:

- **Platform detection + package-name mapping**: Advisor "Detects the Linux distribution and package manager... and maps tool names to distro-specific package names for install suggestions."
- **Compiler selection**: "Determines which C compiler to use... this decision is **made by Advisor** during readiness checks."

Currently these concerns are scattered across three locations:
1. `internal/platform/service.go` — `PlatformService` with package-name mapping (only used by `internal/repository/disk/advisor.go`)
2. `internal/compiler/service.go` — `CompilerService` with gcc/zig selection (only used by `internal/terminal/`)
3. `internal/terminal/handler_system.go` — `doctorPkgNames` map (duplicates platform's mapping)

They also violate Clean Architecture: `internal/` is for I/O only, not business logic.

## Goals / Non-Goals

**Goals:**
- Delete `internal/platform/` — absorb package-name mapping into Advisor's infrastructure layer
- Delete `internal/compiler/` — absorb compiler types, selection logic, and availability checks into Advisor
- Remove the `Compiler` field from `TerminalHandler` — compiler info flows through Advisor
- Consolidate the duplicate `doctorPkgNames` map in `handler_system.go` with the merged mapping
- Zero behavioral change — `phpv doctor` output must be identical

**Non-Goals:**
- Moving `internal/config/` to `config/` (global config singleton)
- Splitting `internal/utils/` into domain/pkg packages
- Introducing new interfaces or changing the AdvisorRepository contract more than necessary

## Decisions

### Decision 1: Package-name mapping lives in advisor infrastructure, not a new use-case package

`PlatformService.GetPackageName()` is only called from `internal/repository/disk/advisor.go` (lines using `r.platform.GetInstallSuggestion(name)`). Rather than creating a new root-level `platform/` package that still only gets used by Advisor, we inline the mapping as a private function inside the advisor disk repository. This keeps the dependency graph simpler.

### Decision 2: Compiler types and selection logic move to the Advisor domain/use-case layer

The `CompilerType` enum (`gcc`, `zig`), `CompilerInfo` struct, and selection rules (`GetRequiredCompilerForPHP`, `GetEffectiveCompilerForPHP`, etc.) are moved into the Advisor module. Specifically:

- `CompilerType` and `CompilerInfo` → `domain/advisor.go` (domain types — pure data, zero imports)
- Selection logic → `advisor/service.go` (use-case layer, consumed via AdvisorRepository)

Terminal gets compiler info through the Advisor service instead of a direct `CompilerService` reference.

### Decision 3: Add compiler info to Advisor's readiness check

Advisor's `Check()` already determines build tools availability. We extend this to include compiler selection results for each PHP major version. The Terminal's `DoctorV2()` already calls `DoctorV2()` internally to build its `compilerByMajor` list — after the merge, this data comes from Advisor.

The minimal approach: add a `GetCompilerReadiness(phpVersion string)` method to `AdvisorRepository` that returns compiler availability and effective compiler per major version. The existing `TerminalHandler.Compiler` field is replaced by calling this through the existing advisor service.

### Decision 4: Terminal's `doctorPkgNames` map uses the merged Advisor mapping

`handler_system.go` has its own `doctorPkgNames` map (20 entries mapping tool names to distro-specific packages). After merging platform's mapping into Advisor, we either:
- Export the mapping from Advisor's infrastructure (cleaner)
- Keep `doctorPkgNames` as-is but note it's a delivery-layer concern (simpler, less risk)

We choose the simpler approach: `doctorPkgNames` stays in `handler_system.go` as a delivery-layer format concern. The merged platform mapping in Advisor's infrastructure handles the `GetInstallSuggestion()` case. The duplication is acceptable because Terminal formats suggestions differently ("phpv will build X" vs "apt install X") and the two maps serve different contexts.

## Layer Map

```
Before:
  internal/platform/     ← business logic in wrong layer
  internal/compiler/     ← business logic in wrong layer
  advisor/ (thin wrapper) → internal/repository/disk/advisor.go → imports internal/platform/
  internal/terminal/     → imports internal/compiler/

After:
  domain/advisor.go      ← CompilerType, CompilerInfo types (pure data)
  advisor/service.go     ← compiler selection logic + CompilerReadiness method on interface
  internal/repository/   ← absorbs platform package-name mapping (private helper)
    disk/advisor.go
  internal/terminal/     ← gets compiler info through advisor.Repository, no direct compiler import
```

## Risks / Trade-offs

- **Risk**: `advisor/service.go` grows significantly with compiler selection logic. Currently it's a 3-line passthrough.
  - **Mitigation**: Compiler logic is ~120 lines. Moving it there makes the use-case layer more meaningful. Follow-up can extract a `compiler.go` helper within the advisor package if needed.

- **Risk**: Changing `AdvisorRepository` interface is a breaking change for all implementations.
  - **Mitigation**: Only one implementation exists (`internal/repository/disk/advisor.go`). No external consumers.

- **Trade-off**: `doctorPkgNames` map stays in terminal — not perfectly deduplicated, but avoids coupling terminal formatting to advisor infrastructure. Tracked for future clean-up when `internal/config` and `internal/utils` are also addressed.
