## ADDED Requirements

### Requirement: Silo unpacks source archives to destination directories

The Silo module SHALL provide an `UnloadRepository` interface with an `Unpack(source, destination)` method that extracts archive files (`.tar.gz`, `.tar.xz`, `.zip`) to a destination directory. The interface SHALL be consumed by the bundler orchestrator and forge infrastructure.

#### Scenario: Unpack a tar.gz archive
- **WHEN** `Unpack("/tmp/source.tar.gz", "/tmp/dest")` is called
- **THEN** the system extracts the archive contents into the destination directory
- **AND** returns an `Unload` struct containing the extracted path

#### Scenario: Unpack a tar.xz archive
- **WHEN** `Unpack("/tmp/source.tar.xz", "/tmp/dest")` is called
- **THEN** the system extracts the archive contents into the destination directory

#### Scenario: Unpack a zip archive
- **WHEN** `Unpack("/tmp/source.zip", "/tmp/dest")` is called
- **THEN** the system extracts the archive contents into the destination directory

#### Scenario: Unpack fails with invalid source
- **WHEN** `Unpack("/nonexistent/archive.tar.gz", "/tmp/dest")` is called
- **THEN** the system returns an error indicating the source could not be read

### Requirement: Silo service delegates to repository

The Silo module SHALL provide a `Service` struct that wraps `UnloadRepository` and delegates `Unpack` calls to the repository implementation. The service SHALL be injectable via uber-go/fx.

#### Scenario: Service delegates unpack to repository
- **WHEN** `Service.Unpack(source, dest)` is called
- **THEN** the system calls `UnloadRepository.Unpack(source, dest)` and returns the result
