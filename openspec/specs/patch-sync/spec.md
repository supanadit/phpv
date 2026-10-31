# patch-sync Specification

## Purpose

Defines how phpv keeps the patch tree on `PHPV_ROOT` current at runtime — patches are runtime data published to the repository's `main` branch and never shipped inside the binary, so users with an outdated phpv still receive newly published patches.

## Requirements

### Requirement: Update command syncs the managed patch tree as a delta
The `phpv patch` CLI group SHALL provide an `update` command that refreshes the managed patch tree at `PHPV_ROOT/patches/` from the published tree using delta sync: only files that are new or changed relative to the local manifest are downloaded, and remote deletions are applied. A machine with no prior sync SHALL receive the full tree.

#### Scenario: Delta downloads only changed files
- **WHEN** the remote tree at the latest revision differs from the local manifest by one changed and one new patch file, and it previously contained ten other files
- **THEN** exactly the two files are downloaded and the other ten remain untouched on disk

#### Scenario: Remote deletion is applied locally
- **WHEN** a range directory was removed upstream between two revisions
- **THEN** the stale files disappear from the managed tree and the manifest after a successful update

### Requirement: Delta sync records state atomically and resumes
Downloaded files SHALL be staged and moved into place individually; the local manifest and revision marker SHALL be updated together only after the delta completes successfully. An interrupted delta SHALL leave the previous manifest intact so the next run transparently completes the remainder.

#### Scenario: Interrupted delta resumes cheaply
- **WHEN** downloads for the delta failed partway through
- **THEN** the manifest still reflects the last successful state and a subsequent update downloads only the missing remainder

### Requirement: Revision tracking skips redundant syncs
phpv SHALL record the revision of the last successfully applied sync (and the per-file manifest) inside the managed tree. Before downloading anything, the update path SHALL first determine the remote revision and, when it equals the recorded revision, complete without any file downloads.

#### Scenario: Already up to date performs no download
- **WHEN** the user runs `phpv patch update` and the remote revision matches the recorded revision
- **THEN** phpv reports that the patch tree is already up to date and downloads nothing

#### Scenario: Outdated marker triggers a delta
- **WHEN** the remote revision differs from the recorded revision because patches were republished
- **THEN** phpv downloads only the delta between the recorded and remote revision and records the new revision

### Requirement: Scoped update limits the delta to a version
`phpv patch update` SHALL accept an optional scope positional in the same `[package] [version]` form as `phpv patch list`; a single bare argument SHALL be treated as a version name and scoped to the `php` package (e.g. `phpv patch update 7.4`). When a scope is given, the delta SHALL be computed for the full tree but only files under directories whose encoded range matches the scope SHALL be downloaded or deleted — files outside the scope remain governed by their previous syncs.

#### Scenario: Scoped sync touches only matching files
- **WHEN** the remote tree contains changed files under `php/7.4.0-8.0.0/` and `icu/58.0-60.0/`, and the user runs `phpv patch update 7.4`
- **THEN** only the files under `php/7.4.0-8.0.0/` are synchronized; the `icu` files, unchanged or not, remain as previously synced

#### Scenario: Scoped sync of a version with no matching files reports emptiness
- **WHEN** the user runs `phpv patch update 8.9` and no remote directory range matches PHP 8.9
- **THEN** phpv completes without downloading anything and reports that no patches match the requested scope

### Requirement: Fetch is an alias of update
The `phpv patch` CLI group SHALL accept `fetch` as an alias for `update` with identical behavior, including the scope argument.

#### Scenario: Alias behaves identically
- **WHEN** the user runs `phpv patch fetch 7.4` on the same state that `phpv patch update 7.4` would see
- **THEN** the result is indistinguishable from `patch update 7.4` (same delta outcome, same manifest and revision state, same messaging subject to the differing command name in output)

### Requirement: Sync never touches user-local patches
The managed tree `$PHPV_ROOT/patches/` SHALL be entirely owned by the sync process — update may add, rewrite, or remove any file inside it. A separate user-local tree `$PHPV_ROOT/patches-local/` SHALL never be read, written, or removed by the sync mechanism, and sync SHALL preserve any content there across updates.

#### Scenario: Republished range directories are replaced, not appended
- **WHEN** the remote patch tree renames a range directory that also exists in the managed tree
- **THEN** the stale managed range directory is removed, not left alongside the new one

#### Scenario: User-local additions survive an update
- **WHEN** the user previously placed `patches/oniguruma/6.9.7/0001-hotfix.patch` under `patches-local/` and runs `phpv patch update`
- **THEN** that file remains untouched and continues to be honored at build time alongside the synced tree

### Requirement: Install self-syncs patches before building
Every `phpv install` SHALL ensure the managed patch tree is fresh before building, so that patches reach users regardless of binary age. The check and delta SHALL match `phpv patch update` behavior. A failed fetch SHALL be emitted as a warning and the install SHALL continue: with the previous synced tree when one exists, or unpatched when none does.

#### Scenario: New patches reach an outdated binary at install time
- **WHEN** the user's binary predates newly published patches and they run `phpv install`
- **THEN** the managed tree is delta-synced before the build starts and the build applies the newly published patches

#### Scenario: Install proceeds with warning when the fetch fails
- **WHEN** `phpv install` cannot reach the patch publication source on a machine with an existing synced tree
- **THEN** the build starts with the existing synced tree after emitting a warning that patch freshness could not be verified

#### Scenario: Cold first install offline builds unpatched with a warning
- **WHEN** `phpv install` runs on a machine with no synced tree and the fetch fails
- **THEN** the build proceeds without downloaded patches after emitting a warning, per the source-patching rule that a missing tree is an outcome rather than an error

### Requirement: Install freshness checks respect a stale-lease window
Install-time freshness checks SHALL consult the recorded check time and skip the remote revision check while within a bounded, documented stale-lease window; outside the window they SHALL perform the revision check and delta sync. Explicit `phpv patch update` SHALL ignore the lease.

#### Scenario: Stale-lease skips the remote check
- **WHEN** an install occurs within the lease window after a successful check and delta
- **THEN** no revision check call is made and the build uses the already-synced tree, reporting that the check was skipped as fresh

### Requirement: Status and list make patch state observable
The `phpv patch` CLI group SHALL provide `status`, reporting the recorded managed-tree revision, check time (whether inside the stale-lease), whether remote content differs, and patch counts per source; and `list [package] [version]`, reporting which patch files would be selected for that package version and from which source tree each came.

#### Scenario: Status identifies available update
- **WHEN** the user runs `phpv patch status` and the managed tree's revision is behind the remote revision
- **THEN** phpv reports the recorded revision, the remote revision, and that an update is available

#### Scenario: List shows what would apply for a version
- **WHEN** the user runs `phpv patch list php 8.0.30`
- **THEN** phpv lists the selected patch files and flags manifests for `php@8.0.30` in application order and indicates each file's source tree (synced or user-local)
