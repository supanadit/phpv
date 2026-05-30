## MODIFIED Requirements

### Requirement: Advisor provides compiler readiness for PHP versions

The Advisor module SHALL determine which C compiler (gcc or zig) is appropriate and available for each PHP major version, and expose this information through its repository interface alongside existing readiness checks.

#### Scenario: PHP 5.x-7.x prefers gcc when available

- **WHEN** Advisor checks compiler readiness for PHP 7.4.33 and gcc is available
- **THEN** the effective compiler is `gcc`

#### Scenario: PHP 8.x prefers zig when gcc unavailable

- **WHEN** Advisor checks compiler readiness for PHP 8.2.0 and gcc is not available but zig is
- **THEN** the effective compiler is `zig`

#### Scenario: Zig auto-download eligibility

- **WHEN** Advisor checks compiler readiness for PHP 8.2.0, zig is not installed, but make is available
- **THEN** the system reports zig as available with auto-download flag set to true

#### Scenario: No viable compiler blocks build

- **WHEN** Advisor checks compiler readiness and neither gcc nor zig is available
- **THEN** the system reports no viable compiler and the buildability verdict is blocked

#### Scenario: Forced compiler override

- **WHEN** Advisor checks required compiler for PHP 8.2.0 with a forced compiler of `gcc`
- **THEN** the required compiler is `gcc` regardless of PHP version preference

### Requirement: Advisor maps tool names to distro-specific package names

The Advisor module SHALL map generic tool/library names to distro-specific package names for the detected package manager, and include install suggestions in readiness check results.

#### Scenario: Map library name to apt package

- **WHEN** Advisor generates an install suggestion for `libxml2` on a Debian/Ubuntu system
- **THEN** the suggestion uses the apt package name `libxml2-dev`

#### Scenario: Map library name to brew package

- **WHEN** Advisor generates an install suggestion for `icu` on macOS
- **THEN** the suggestion uses the brew package name `icu4c`

#### Scenario: Fallback when no mapping exists

- **WHEN** Advisor generates an install suggestion for an unmapped tool name
- **THEN** the suggestion uses the tool name as-is with the detected install command

## REMOVED Requirements

### Requirement: Standalone CompilerService package

**Reason**: Compiler selection is an Advisor concern per the module map. The standalone `internal/compiler/` package duplicated this responsibility outside of Advisor.

**Migration**: Terminal consumers call Advisor for compiler readiness instead of a separate `CompilerService`. The `CompilerType`, `CompilerInfo`, and selection logic move into the Advisor module (domain types → `domain/advisor.go`, selection logic → `advisor/service.go`).

### Requirement: Standalone PlatformService package

**Reason**: Platform detection and package-name mapping are Advisor concerns per the module map. The standalone `internal/platform/` package duplicated this responsibility and also violated Clean Architecture by placing business logic in `internal/`.

**Migration**: Package-name mapping is absorbed into Advisor's infrastructure (`internal/repository/disk/advisor.go`). Install suggestion generation uses Advisor's readiness checks.
