## ADDED Requirements

### Requirement: Source package resolves URL patterns for download

The Source module SHALL provide a `PatternRepository` interface that resolves package names and versions to download URLs with OS/arch-aware matching. The interface SHALL be consumed by the bundler orchestrator and the advisor infrastructure.

#### Scenario: Match URL pattern by package name and version
- **WHEN** `MatchPattern(name, version)` is called with a registered package name
- **THEN** the system returns the best-matching `URLPattern` that satisfies the version constraint

#### Scenario: Match URL pattern by type, OS, and architecture
- **WHEN** `MatchPatternByType(name, sourceType, targetOS, targetArch, version)` is called
- **THEN** the system returns the pattern with exact OS+arch match, falling back to OS-only match, then platform-independent match

#### Scenario: Build URL from pattern template
- **WHEN** `BuildURL(pattern, version)` is called
- **THEN** the system replaces `{version}`, `{major}.{minor}`, `{major}`, `{minor}`, and `{ext}` placeholders with actual version values

#### Scenario: Build URL with explicit OS and arch parameters
- **WHEN** `BuildURLByType(name, version, sourceType, targetOS, targetArch)` is called
- **THEN** the system matches a pattern for the given OS/arch and builds the URL
- **AND** the system does NOT read OS/arch from any global config or environment variable

#### Scenario: No matching pattern found
- **WHEN** `MatchPattern(name, version)` is called for an unregistered package
- **THEN** the system returns an error with message containing the package name

### Requirement: Source service registers and indexes URL patterns

The Source module SHALL provide a `Service` struct that maintains an in-memory index of `URLPattern` values keyed by package name. The service SHALL prevent duplicate patterns.

#### Scenario: Register URL patterns
- **WHEN** `RegisterPatterns(patterns)` is called with a slice of patterns
- **THEN** the system indexes each pattern by its name
- **AND** duplicate patterns (same type, template, checksum) are not added

#### Scenario: Build all URLs including fallbacks
- **WHEN** `BuildURLs(pattern, version)` is called
- **THEN** the system returns the primary URL followed by all fallback URLs with template variables expanded
