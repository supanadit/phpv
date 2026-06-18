## ADDED Requirements

### Requirement: Forge resolves C/C++ compiler standard flags per PHP version

The Forge module SHALL provide a `CStdRule` struct defining C and C++ standard flags (`CStd`, `CXXStd`) with optional minimum/maximum PHP version ranges. The `Repository` interface SHALL expose `GetCompilerStdRule(phpVersion)` returning the appropriate rule.

#### Scenario: Get compiler standard rule for PHP 8.0+
- **WHEN** `GetCompilerStdRule("8.0.30")` is called
- **THEN** the system returns a `CStdRule` with `CStd: "-std=gnu11"` and `CXXStd: "-std=gnu++17"`

#### Scenario: Get compiler standard rule for pre-8.0 PHP
- **WHEN** `GetCompilerStdRule("7.4.33")` is called
- **THEN** the system returns a `CStdRule` with appropriate C and C++ standards

### Requirement: Forge converts CFLAGS to CXXFLAGS

The Forge module SHALL provide `CXXFlagsFromCFlags` and `CXXFlagsFromCFlagsWithStd` functions that convert C compiler flags to C++ compiler flags by replacing C standard flags with C++ equivalents and stripping C-only warning flags.

#### Scenario: Convert C11 flags to C++17 for PHP build
- **WHEN** `CXXFlagsFromCFlags(["-std=gnu11", "-O2"], true)` is called
- **THEN** the system returns `["-std=gnu++17", "-O2"]`

#### Scenario: Strip C-only warning flags
- **WHEN** `CXXFlagsFromCFlags(["-std=gnu11", "-Wstrict-prototypes"], false)` is called
- **THEN** the system returns `["-std=gnu++17"]` (stripping the C-only warning)

#### Scenario: Convert using version-specific C++ standard rule
- **WHEN** `CXXFlagsFromCFlagsWithStd(["-std=gnu11", "-O2"], true, stdRule)` is called with `stdRule.CXXStd = "-std=gnu++14"`
- **THEN** the system returns `["-std=gnu++14", "-O2"]`

#### Scenario: Ensure C++ standard for PHP build when no C standard flag present
- **WHEN** `CXXFlagsFromCFlags(["-O2"], true)` is called (no `-std=` flag in CFLAGS)
- **THEN** the system appends `-std=gnu++17` to the result

### Requirement: Forge provides C-only warning flag exclusion list

The Forge module SHALL expose a `COnlyWarnings` map of C compiler warning flags that have no C++ equivalent and must be stripped during CFLAGS→CXXFLAGS conversion.

#### Scenario: Reference C-only warning flags
- **WHEN** the `COnlyWarnings` map is imported by bundler infrastructure
- **THEN** it contains entries for `-Wstrict-prototypes`, `-Wno-implicit-function-declaration`, and similar C-only flags

### Requirement: Forge provides compiler flags per compiler type

The `Repository` interface SHALL expose `GetCompilerFlags(compiler, phpVersion)` returning C compiler flags (`CFLAGS`) specific to a compiler type ("gcc" or "zig") and PHP version.

#### Scenario: Get GCC flags for PHP 8.4
- **WHEN** `GetCompilerFlags("gcc", "8.4.0")` is called
- **THEN** the system returns a slice of CFLAGS appropriate for GCC building PHP 8.4

#### Scenario: Get compiler flags for unsupported PHP version
- **WHEN** `GetCompilerFlags("gcc", "4.0.0")` is called
- **THEN** the system returns an empty slice or system-default flags
