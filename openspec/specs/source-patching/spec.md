# source-patching Specification

## Purpose

Defines how phpv discovers, selects, applies, and reports source-code patches against package source trees before building, including how multiple patch sources are merged and how application failures surface.

## Requirements

### Requirement: File-based patch selection by package and version range
phpv SHALL discover patches as files in a versioned directory tree of the form `patches/<package>/<range-directory>/<ordinal>-<slug>.patch`, where the range directory encodes a version range in a filesystem-safe grammar: `<min>-<max>` (min inclusive, max exclusive), `<version>+` (open-ended), a bare version (exact match), or `any` (every version). Range matching SHALL use the same version-comparison rules as version constraints elsewhere in phpv, and phpv SHALL apply exactly those directories whose range matches the built package version, in ascending ordinal order. A directory containing only a flags manifest SHALL still be a valid selection.

#### Scenario: Version inside a range directory matches
- **WHEN** phpv builds `php@8.0.30` and the patch tree contains `patches/php/8.0.0-8.1.0/0001-intl-cxx17.patch`
- **THEN** that patch is selected for package `php`, reported under its filename-derived name, and applied to the extracted source in ordinal order

#### Scenario: Version outside the range does not match
- **WHEN** phpv builds `php@7.4.33` and the only range directories for `php` are `8.0.0-8.1.0` and `7.0.0-8.0.0`
- **THEN** only patches under `7.0.0-8.0.0` are selected — the `8.0.0-8.1.0` directory is ignored

#### Scenario: Flags-only entry with no patch files
- **WHEN** phpv builds `curl@8.x` and the selected range directory contains only a flags manifest with configure options
- **THEN** the manifest flags are still contributed to the build and no patch application is attempted

### Requirement: Patch content is a unified diff applied with a standard patch tool
Each patch file SHALL be a valid unified diff with paths relative to the package's extracted source root, applied as whole-file content (not interpreted or rewritten) using a standard patch invocation at strip level 1 against the located source root. Because extracted sources may be nested inside the per-package source path, phpv SHALL resolve the deepest directory that contains the patch's target files before applying.

#### Scenario: Nested source root is resolved
- **WHEN** the extracted source for `php@7.4.33` lives at `<sourcePath>/php-7.4.33/` and a patch touches `ext/standard/scanf.c`
- **THEN** the patch is applied against `<sourcePath>/php-7.4.33/` as the root, and the reported application succeeds when the diff applies cleanly

#### Scenario: Patch applies in ordinal order
- **WHEN** two selected patch files for the same package modify the same file
- **THEN** the second patch observes the first patch's changes and applies cleanly on top of them

### Requirement: Patch application failures are loud
When a selected patch fails to apply, phpv SHALL abort the build and report a message that names the failing patch file, the package and version, and the resolved source root. phpv MUST NOT silently skip a selected patch whose context does not match (replacing the previous substring-replacement behavior). An already-applied patch leading to ambiguous matchfulness SHALL be reported as that patch's outcome.

#### Scenario: Context mismatch aborts the build
- **WHEN** a selected patch `0001-intl-cxx17.patch` cannot match its context in `php@8.0.30` source
- **THEN** the build fails and the emitted message references the patch file name, `php@8.0.30`, and the source root, rather than continuing

#### Scenario: Already-applied patch is reported
- **WHEN** a selected patch's changes are already present in the source and the patch tool reports it as already applied
- **THEN** phpv reports the patch as already applied and continues the build without failing

### Requirement: Per-directory flags manifest contributes build flags
A range directory MAY include a flags manifest declaring additional C compiler flags (ExtraCFlags) and configure options (ConfigureFlags). phpv SHALL combine the flags of all selected entries for the package into the package's build environment, resolved in the same ordinal order as patches, with placeholders (`{{prefix}}`, `{{source}}`, `{{dep:NAME}}`) resolved by the assembler exactly as inline patch declarations resolve today.

#### Scenario: Manifest placeholders resolve
- **WHEN** the selected manifest for `libzip` declares `ConfigureFlags: ["-DZLIB_ROOT={{dep:zlib}}"]` and the build plan contains a locally-built zlib
- **THEN** the placeholder resolves to zlib's install prefix and is passed to libzip's configure invocation

#### Scenario: Manifest plus patch flags combine
- **WHEN** a selected entry's manifest contributes ExtraCFlags and another selected entry's manifest contributes ConfigureFlags for the same package
- **THEN** both flag sets reach the package build (C flags for compilation, configure options for configuration), in selection order

### Requirement: Merged patch sources with precedence
phpv SHALL merge two patch sources — user-local patches and synced patches under `PHPV_ROOT` — with user-local first, and SHALL NOT ship patches inside the binary. Precedence SHALL be decided per patch-tree directory: a directory present in a higher-precedence source shadows the same directory in the lower one, while directories present only in the lower source still apply.

#### Scenario: User-local directory shadows the synced equivalent
- **WHEN** both the synced tree and the user-local tree contain `patches/php/7.0.0-8.0.0/` and the user restructures that directory's patch files
- **THEN** only the user-local tree's selection for that directory is applied — no synced file from that directory is re-applied

#### Scenario: New directory in a lower-priority source still applies
- **WHEN** the user-local tree contains `patches/oniguruma/6.9.7/0001-fetched-hotfix.patch` but the synced tree has no `patches/oniguruma/` directory at all
- **THEN** the user-local patch is selected and applied together with any synced patches from other directories

### Requirement: Missing patch tree is a reported outcome, not an error
When no synced patch tree exists on the machine (cold install, first run offline) and no user-local patches cover the package, phpv SHALL proceed with the build without applying any patches for it, and SHALL report that the package builds unpatched. This outcome MUST NOT abort the build, because patch fetches may be unavailable while package sources still download successfully.

#### Scenario: Cold offline machine builds unpatched with a warning
- **WHEN** `phpv install` runs after its patch fetch failed and `$PHPV_ROOT/patches/` exists but does not cover the built package's version, and `$PHPV_ROOT/patches-local/` is empty for that package
- **THEN** the build continues without patches and emits a report naming the package that built unpatched and the reason (no matching patch entries present)

#### Scenario: User-local patches alone still apply when the synced tree is absent
- **WHEN** `$PHPV_ROOT/patches/` was removed entirely but `$PHPV_ROOT/patches-local/php/<range>/0001-hotfix.patch` exists and matches the built version
- **THEN** that user-local patch is still selected and applied

### Requirement: Patch application is observable
When patches are selected and applied for a package, the build SHALL report the count and the names of applied patch files, so users can correlate a build result with the patch sources involved. When no patch is selected, phpv SHALL report that outcome with the package Zero-verbosity it was checked.

#### Scenario: Applied notification lists patch names
- **WHEN** two patches apply to `php@8.0.30`
- **THEN** a build event reports `Applied 2 patch(es)` to php@8.0.30 and names both patch files

#### Scenario: No patches selected is reported
- **WHEN** phpv builds a package whose library version has no matching range directory in any source
- **THEN** phpv continues the build without patch details, without simulating their absence as an error
