## ADDED Requirements

### Requirement: Extension module validates extensions for PHP version compatibility

The Extension module SHALL provide methods on its `Repository` interface to validate whether extensions are valid for a given PHP version, check for conflicting extensions, and resolve extension-to-package dependencies.

#### Scenario: Validate a single extension for PHP version
- **WHEN** `IsExtensionValidForPHPVersion("gd", "8.4.0")` is called
- **THEN** the system returns `true` if gd is supported in PHP 8.4, `false` otherwise

#### Scenario: Validate a list of extensions
- **WHEN** `ValidateExtensions(["gd", "imaginary-ext"], "8.4.0")` is called
- **THEN** the system returns a slice containing `"imaginary-ext"` as unknown
- **AND** returns an error if any unknown extensions are found

#### Scenario: Check for conflicting extensions
- **WHEN** `CheckExtensionConflicts(["mysql", "mysqli", "pdo_mysql"])` is called
- **THEN** the system returns any conflict pairs (e.g., extensions that cannot coexist)

#### Scenario: Resolve extension dependency
- **WHEN** `GetExtensionDependency("gd")` is called
- **THEN** the system returns the package name the extension depends on (e.g., `"libgd"`) and `true`

#### Scenario: Resolve extension dependency with version
- **WHEN** `GetExtensionDependencyWithVersion("gd", "8.4.0")` is called
- **THEN** the system returns the package name and version constraint for the PHP version

#### Scenario: Get extension definition
- **WHEN** `GetExtensionDef("gd")` is called
- **THEN** the system returns the `ExtensionDef` struct with configure flags and dependencies

#### Scenario: Get conflicting extensions list
- **WHEN** `GetConflictingExtensions("mysql")` is called
- **THEN** the system returns a list of extension names that conflict with mysql

### Requirement: Extension module exposes sentinel errors

The Extension module SHALL reference sentinel errors `ErrUnknownExtension` and `ErrExtensionConflict` from `domain/errors.go`. The Service SHALL return these errors when validation or conflict checks fail.

#### Scenario: Unknown extension validation fails
- **WHEN** `Service.ValidateExtensions(["imaginary-ext"], "8.4.0")` is called
- **THEN** the system returns `domain.ErrUnknownExtension`

#### Scenario: Extension conflict detected
- **WHEN** `Service.CheckExtensionConflicts(["mysql", "mysqli"])` is called with conflicting extensions
- **THEN** the system returns `domain.ErrExtensionConflict`
