# Design

## Context

See proposal.md — Why. State of the code that constrains the fix:

- `system.Check(phpDeps []string)` takes bare dependency names and returns
  `Available`/`Missing`. It sets `Package.Version` only inside the
  `if installed` branch (`system/service.go:39-41`), so `Missing` entries always
  carry `Version == ""`. `Check` is otherwise pure (no caching), so calling it
  twice is cheap: each package costs one `rpm`/`dpkg`/`apk`/`pacman` invocation.
- `checkSystemDeps` (`internal/terminal/php.go:1034`) builds `phpDeps` from a
  hardcoded 11-entry array plus `RequiresPackages` from extension definitions,
  then on consent does:
  ```go
  for _, p := range result.Missing {
      p.Installed = true          // Version stays ""
      systemAvail[p.Name] = p
  }
  ```
- The assembler gate that consumes this map (`assembler/service.go:244-252`)
  requires `sysPkg.Installed && sysPkg.Version != ""`, then applies
  `domain.MatchVersionRange(constraint, sysPkg.Version)`. So an empty version
  fails the gate and the dependency is rebuilt from source — silently.
- `promptAndInstall` (`internal/terminal/php.go:1097`) is shared by
  `checkSystemDeps` (`php install`, `extension add`) and `checkBuildTools`, and
  returns a single bool "were packages installed".
- Build knowledge: `builtInPackages()` lists the 16 buildable packages
  (`autoconf automake bison bzip2 curl flex icu libpq libtool libxml2 libzip m4
  oniguruma openssl perl php zig zlib`); `RequiresPackage` (singular) is what
  puts a package into a build plan for an extension. `RequiresPackages`
  (plural) is prompt-only metadata. Of the 11 hardcoded `phpDeps`, only
  `openssl libxml2 zlib oniguruma curl libzip` are buildable-or-linked;
  `sqlite3 readline pcre2 argon2 sodium libjpeg` have neither a
  `builtInPackages` entry nor a `RequiresPackage` link, so the build cannot use
  them as planned dependencies.
- `system/packages.go` maps dependency names to distro package names per distro
  (fedora/ubuntu/alpine/arch); names absent from a map are silently skipped by
  `Check`, which is a second, quieter source of "prompted but unusable".
- `doctor/service.go:152` holds a **second copy** of the same hardcoded
  `phpDeps` array (also including `argon2`), and reports its `Missing` entries as
  one issue titled "Missing system libraries". `doctor` is therefore in scope:
  it currently disagrees with what `phpv install` will ask for.

## Goals / Non-Goals

**Goals:**
- A prompted package that installs successfully is *used* by the build when its
  version satisfies the constraint.
- The prompt does not list packages the build cannot use.
- The prompt tells the truth about the version gate.
- `doctor` reports the same package set as `phpv install`, separating required
  build dependencies from advisory dev libraries.

**Non-Goals:**
- Changing `MatchVersionRange`, the assembler's dependency resolution, or the
  decision to prefer system over source when versions fit.
- Making `normalizeVersion` understand every distro version scheme. Note the
  known gap: `libargon2-devel` reports `20190702` (a date, not semver), which
  cannot satisfy `>=1.0`-style constraints. This change removes `argon2` from
  the prompted set anyway (no build link), so the gap stops mattering here; a
  general fix is separate work.
- Adding package-manager install support for new distros.

## Decisions

**1. Refresh by re-running `Check` after install, instead of calling
`getVersion` per package.**
`checkSystemDeps` becomes: check → prompt/install → if installed, `Check` again
and rebuild `systemAvail` from the fresh `Available`. Rationale: reuses the one
code path that already knows how to resolve versions per PM, and it is
self-correcting — a partially failed install, or a package the PM installed
under a different name, yields the true state rather than our assumption.
Alternative rejected: adding `Version: getVersion(...)` inside the
`for _, p := range result.Missing` loop — duplicative, and it asserts success
without verifying that the package actually became visible to
`isInstalled`/`getVersion`.

**2. Derive the prompted set from build knowledge, and keep an explicit
advisory list for the remainder.**
The hardcoded `phpDeps` array is replaced by: (a) the dependency names reachable
from the build plan for the requested version/extensions — which is exactly the
set `RequiresPackage` already populates — plus (b) a small, explicitly named
list for dev libraries that genuinely have no buildable package definition but
are still needed (`sqlite3`, `readline`, `pcre2`, `sodium`, `libjpeg`,
`libpng`, `freetype`, `libpq` are the real ones; each is a system-only header
dependency of an extension). `argon2` is dropped: no extension in the graph
links it via `RequiresPackage`, and its version cannot be compared anyway.
Alternative rejected: filtering the current array at runtime by probing
`builtInPackages` — it would still hardcode the same list, just later, and
would hide the reason each entry exists.

**3. Carry "why is this package listed" in the data, not in the prompt code.**
Add a field to `system.Package` — `Required bool` (a known build dependency with
a version constraint that will be checked) — set by the caller that builds the
list. The prompt prints required packages plainly and marks advisory ones, so
the distinction is expressed once in data and can be reused by `doctor`.
Alternative rejected: two separate `Check` calls (required/advisory) — doubles
the package-manager subprocess count for no behavioural gain.

**4. Prompt wording states the version gate rather than predicting outcomes.**
`promptAndInstall` is shared with `checkBuildTools` (where the version gate does
not apply), so the added sentence is parameterized: dependency prompts get the
"used only when the version satisfies the requirement, otherwise built from
source" line; build-tool prompts do not. Alternative rejected: changing
`promptAndInstall`'s wording globally — it would be wrong for build tools.

**5. Verification is a unit test on the returned map, not an end-to-end build.**
The regression that matters — `Installed == true` with `Version == ""` — is
directly assertable with a fake `SystemService`. An end-to-end `phpv install`
is a multi-minute source build and cannot run in CI. `system.Check` itself gets
a test using a stubbed package manager where feasible; otherwise the
refresh-after-install behavior is covered at the `checkSystemDeps` seam.

**6. `doctor` adopts the same dependency set and the `Required` flag.**
`doctor/service.go:152` carries a second copy of the hardcoded `phpDeps` array,
including `argon2`, so leaving it alone would keep a stale, inaccurate list in
the command users run *precisely to diagnose* missing packages. `doctor`'s
system-library check is reworked to consume the same derived set and to split its
report into two issues: missing **build dependencies** (severity warning,
names the affected constraint) and missing **optional dev libraries** (severity
info, advisory wording). The two lists share one `Check` call. To avoid a third
copy of the derivation, the dependency-set construction moves to one exported
function used by both `internal/terminal` and `doctor`; the extension metadata
source stays the graph repository in both cases.
Alternative rejected: leaving `doctor` on its own array — it would report
`argon2` as a required missing library while `phpv install` no longer asks for
it, which is exactly the inconsistency this change exists to remove.

## Risks / Trade-offs

- **Re-running `Check` doubles package-manager invocations on the install path**
  → acceptable: it only happens after a user-initiated install, and it is one
  subprocess per package. The non-install path is unchanged.
- **Builds that previously rebuilt from source will now link system libraries**,
  which is the point, but it changes build output and could surface
  ABI mismatches that were previously masked → the assembler's existing
  `mustBuild` ABI-compatibility logic (`assembler/service.go:225-235`) still
  forces a rebuild when a dep must be built because a local library is in play;
  this change does not touch it. Worth a manual `phpv install 8 --dry-run`
  check before release.
- **Removing `argon2`/`pcre2`-style entries could under-prompt** if a future
  extension needs them → they remain in `system/packages.go` name maps and can
  be re-added to the advisory list in one line; the accurate path is to add a
  `RequiresPackages` entry to the extension definition.
- **`Package.Required` is a new field on a struct returned across the
  `SystemService` interface** → internal-only package, no external consumers;
  `doctor` and existing tests construct `Package` with named fields, so the
  addition is source-compatible.
- **A package present on the system but absent from a distro's name map is
  still skipped** → pre-existing behaviour, unchanged by this design; the
  proposal scopes it out, but the prompt can no longer claim to cover it
  because advisory entries are labelled.

## Migration Plan

Single change, no data migration, no CLI flags added or removed. Rollback is
`git revert`. Users who relied on `--auto-deps` get the corrected behaviour
without any action; `--no-system` and `--dry-run` are untouched.

## Open Questions

None. The `doctor` question was resolved in favour of adopting `Required`
(Decision 6).
