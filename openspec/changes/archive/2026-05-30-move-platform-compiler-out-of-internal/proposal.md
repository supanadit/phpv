## Why

`internal/platform/` and `internal/compiler/` are standalone packages that duplicate responsibilities already assigned to the **Advisor** module per `openspec/config.yaml`:

- **Platform detection + package-name mapping** is explicitly an Advisor concern: "Detects the Linux distribution and package manager... maps tool names to distro-specific package names for install suggestions."
- **Compiler selection** is explicitly an Advisor concern: "Determines which C compiler to use for a given PHP version — this decision is **made by Advisor** during readiness checks."

These packages shouldn't exist as separate modules. `PlatformService` is only called from Advisor's infrastructure (`internal/repository/disk/advisor.go`), and `CompilerService` is only called from Terminal's doctor command but should flow through Advisor. Additionally, `internal/terminal/handler_system.go` duplicates the package-name mapping in its own `doctorPkgNames` map.

**What's actually wrong:**
1. `internal/platform/` — violates Clean Architecture (business logic in `internal/`), and its package-name mapping duplicates what `handler_system.go` already does in `doctorPkgNames`
2. `internal/compiler/` — violates Clean Architecture, and exists outside Advisor despite the config saying compiler selection "is made by Advisor"

## What Changes

- **Delete `internal/platform/`** — merge its `GetPackageName()` mapping and install-suggestion logic into Advisor's infrastructure (`internal/repository/disk/advisor.go`)
- **Delete `internal/compiler/`** — merge its compiler type definitions, version-based selection rules, and availability checks into Advisor. Terminal's doctor command will call Advisor for compiler info instead of holding a direct `CompilerService` reference. The CompilerRepository interface declared in compiler (never satisfied) is removed.
- **Consolidate duplicate package-name maps** — the `doctorPkgNames` map in `internal/terminal/handler_system.go` will use the same mapping that lives in Advisor, eliminating the duplication.
- **Remove Compiler field from TerminalHandler** — compiler info flows through Advisor's readiness check results.
- **No user-facing behavior change** — the doctor command output is preserved identically.

## Capabilities

### Modified Capabilities

- `advisor`: Advisor's readiness check now includes compiler selection results (gcc vs zig per PHP major version, availability, auto-download status). Terminal retrieves compiler info through Advisor instead of a separate CompilerService.

### New Capabilities

None — platform detection and compiler selection already exist in the module map as Advisor responsibilities; this change just consolidates the scattered implementations into the correct module.

## Impact

- **Deleted**: `internal/platform/service.go` (merged into advisor)
- **Deleted**: `internal/compiler/service.go` (merged into advisor)
- **Modified**: `internal/repository/disk/advisor.go` — absorbs platform package-name mapping
- **Modified**: `internal/terminal/handler.go` — removes Compiler field, adds compiler info to advisor result
- **Modified**: `internal/terminal/handler_system.go` — gets compiler info through advisor, removes duplicate `doctorPkgNames`
- **Modified**: `internal/terminal/cobra_tools.go` — uses advisor-provided compiler types
- **Modified**: `domain/advisor.go` — may add compiler info fields to AdvisorCheck or a new result type
- **Non-goals**: Moving `internal/config/` or refactoring `internal/utils/`. These are separate follow-ups.
