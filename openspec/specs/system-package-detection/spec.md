# system-package-detection Specification

## Purpose

Defines which system packages `phpv` checks for, reports, and offers to install
before a build, and what it guarantees about using them afterwards.

## Requirements

### Requirement: Prompted packages reflect the build plan

The set of system packages `phpv` reports before a build SHALL be derived from
the build plan for the requested package version and extensions, so that it
names only libraries the build can actually consume.

#### Scenario: Extension-backed dev library is reported
- **WHEN** a requested extension declares a system package requirement and that
  package is absent
- **THEN** the missing list contains that package

#### Scenario: Library with no build relationship is not reported
- **WHEN** a candidate library has no buildable package definition and no
  extension in the request declares it
- **THEN** the missing list does not contain that library

#### Scenario: Prompt is skipped entirely with --no-system
- **WHEN** the user passes `--no-system`
- **THEN** no system check or install prompt occurs and every dependency is built
  from source

### Requirement: Installed packages carry a resolved version

After the user consents to installing missing system packages and the install
succeeds, every package returned to the build SHALL report `Installed` together
with a non-empty normalized version.

#### Scenario: Accepted install yields a usable version
- **WHEN** the user accepts the install prompt and the package manager succeeds
- **THEN** each newly installed package is reported as installed with a
  non-empty version

#### Scenario: Version is absent only when the package is genuinely unusable
- **WHEN** a package is reported as installed but the package manager yields no
  version
- **THEN** the build treats it as unavailable rather than assuming it is usable

#### Scenario: Declined or failed install leaves the build consistent
- **WHEN** the user declines the prompt, or the install command fails
- **THEN** the reported package state matches what is actually on the system and
  the build proceeds by building those dependencies from source

### Requirement: Prompt states how packages will be used

Before installing, `phpv` SHALL state that reported packages are used only when
the installed version satisfies the build constraint, and that incompatible
versions are built from source instead.

#### Scenario: User is told about the version gate
- **WHEN** the install prompt is shown
- **THEN** the text indicates that version-incompatible system packages will be
  built from source rather than silently ignored

#### Scenario: Non-interactive session is skipped safely
- **WHEN** stdin is not a terminal and `--auto-deps` is not set
- **THEN** no install is attempted and the build continues from source

### Requirement: System package use is observable

When a dependency is satisfied by a system package, the build SHALL report that
choice, and when a system package is rejected for version incompatibility it
SHALL report the rejection with both versions.

#### Scenario: System package selected
- **WHEN** a dependency constraint is satisfied by the installed system version
- **THEN** the build reports that the system package is being used

#### Scenario: Version incompatibility is explained
- **WHEN** the installed system version does not satisfy the dependency
  constraint
- **THEN** the build reports the rejection, naming the found and required
  versions, and builds the dependency from source

### Requirement: Doctor reports the same package set as install

The `doctor` command SHALL evaluate the same system package set that a build
would check, and SHALL distinguish missing build dependencies from missing
advisory development libraries in its report.

#### Scenario: Missing build dependency is reported as such
- **WHEN** a package required by the build plan for the installed extensions is
  absent
- **THEN** `doctor` reports it as a missing build dependency with a remediation
  command

#### Scenario: Missing advisory library is not presented as required
- **WHEN** a listed package is an optional development library rather than a
  build-plan dependency, and it is absent
- **THEN** `doctor` reports it separately from required build dependencies

#### Scenario: Doctor agrees with install
- **WHEN** `doctor` and `phpv install` are run on the same system with the same
  extensions
- **THEN** neither reports a package as required that the other omits on the
  grounds of having no build relationship
