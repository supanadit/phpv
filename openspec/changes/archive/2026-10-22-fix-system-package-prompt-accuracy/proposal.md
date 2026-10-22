# Proposal

## Why

`phpv install` prompts to install system packages, and users do not trust it —
correctly. Two defects make the prompt unreliable:

1. **Accepted packages are never used.** `system.Check` sets `Package.Version`
   only for packages that were *already* installed (`system/service.go:39-41`).
   After the user accepts the prompt, `checkSystemDeps` marks those packages
   `Installed = true` but leaves `Version` empty
   (`internal/terminal/php.go:1083-1085`). The assembler's hybrid-mode gate
   requires a non-empty version
   (`assembler/service.go:246`), so every just-installed package is silently
   rebuilt from source anyway. The user installs `libcurl-devel`, then watches
   `curl` compile from source.
2. **The prompt claims more than it can know.** `checkSystemDeps` receives only
   extension names, not dependency constraints, so it prompts for every missing
   package regardless of whether the system version will satisfy the build
   constraint. The version check happens later, inside `Assemble`, where an
   incompatible version produces `system-incompatible` and a source build. The
   prompt text ("Install via dnf install -y …?") implies the packages *will* be
   used; often they will not be.

A third, related accuracy gap: the prompt's dependency list is a hardcoded
`phpDeps` array plus `RequiresPackages` from extension definitions. Several
entries in that array are not build dependencies at all (`sqlite3`, `readline`,
`pcre2`, `argon2`, `sodium`, `libjpeg` have no `builtInPackages` entry and no
`RequiresPackage` link), so the prompt asks users to install libraries the build
never uses.

## What Changes

- Fix `checkSystemDeps` so packages installed via the prompt carry a real
  version: re-run the system check after a successful install, so every package
  in the returned map reflects what is actually on disk now.
- Make the prompt state the truth about what happens next: distinguish packages
  that are known build dependencies (will be used if the version fits) from
  advisory ones, and say that version-incompatible packages build from source.
- Derive the prompted dependency set from build knowledge instead of a
  hardcoded array, so `phpv install` only asks for libraries the current build
  plan can actually use. Keep advisory/prompt-only entries only where an
  extension genuinely needs a dev library that is not modelled as a buildable
  package.
- Apply the same derived set and the required/advisory distinction to `doctor`,
  which today carries a second copy of the same hardcoded array and reports
  every absent entry as a required missing library.
- No change to `--no-system`, `--auto-deps`, `--dry-run` semantics, or to the
  assembler's version gate itself.

## Capabilities

### New Capabilities

- `system-package-detection`: how `phpv` decides which system packages a build
  needs, how it reports them, and what it promises about their use after the
  user consents to installing them.

### Modified Capabilities

None. No specs exist yet (`openspec list --specs` is empty).

## Impact

- `system/service.go`: `Check` must be re-runnable to refresh versions after an
  install (already is; no signature change needed).
- `internal/terminal/php.go`: `checkSystemDeps` refresh-after-install, and the
  prompt wording in `promptAndInstall`.
- `internal/repository/memory/graph.go` and/or `domain/extension.go`: the
  dependency list that feeds the prompt.
- `internal/terminal/extension.go`: `extension add` shares `checkSystemDeps`,
  so it inherits both fixes.
- `system/types.go`: possibly a flag or field on `Package` to distinguish "build
  dependency" from "advisory dev library"; decided in design.
- `doctor/service.go`: `checkSystemPackages` adopts the shared derived set and
  splits its report into missing build dependencies vs. missing advisory dev
  libraries. `doctor/service_test.go` gains coverage for the split.
- Behavioural: builds that previously rebuilt `curl`/`libxml2`/`oniguruma` from
  source after installing the system package will now link the system package.
  This changes build output and is the intended fix.
- `doctor` may report fewer missing packages once the list is accurate.
