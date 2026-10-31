# Design — file-based-patch-system

## Context

Today every patch phpv applies lives as Go code in `internal/repository/memory/patcher.go` (~540 lines): a `switch` over package names with inline `VersionRange` strings and `Apply func(sourceDir string) error` closures doing `grep`-style `strings.Replace`/regex edits on extracted sources. Silently skipping a patch that finds no matching pattern is currently relied upon (e.g., versions whose source drifted). The assembler consumes patches through `patcher.Service.PatchesFor` / `Prepare` at three call sites (`assembler/service.go:261, 433, 806`), and `app/phpv.go` wires the repository via fx with `memory.NewPatcherRepository`. The CLI is cobra commands registered through fx; `update/` already contains the GitHub release/download/checksum plumbing to reuse for sync.

Related contracts that stay as-is: `patcher.Patch`, `PatcherRepository`, `PreparedPatches`, and `domain.MatchVersionRange` (`>=x`,`^`,`~`,`=`, comma/space-separated, no wildcard).

## Goals / Non-Goals

Goals:
- Patches become auditable files: unified diffs + optional flags manifests, versionable independently of the binary.
- Old binaries can receive new patches via runtime sync into `$PHPV_ROOT`.
- Loud failure semantics replace silent skips.
- Keep the assembler/patcher service boundary untouched.

Non-Goals:
- Editing patches with phpv (no `phpv patch edit`), building diffs from inside phpv, or shipping patches outside the main repo.
- Automatic pre-install sync; installs stay offline-deterministic.
- Supporting packages other than those already in the current patch set (php, oniguruma, libzip, curl, icu).

## Decisions

### D1: Directory-per-range layout, encoded for filesystem safety
`patches/<package>/<range-directory>/NNN-<slug>.patch`; range directories use a filesystem-safe encoding of the constraint syntax (valid on Windows, no reserved characters): `8.0.0-8.1.0` means `>=8.0.0,<8.1.0`, `6.0.0+` means `>=6.0.0`, a bare `6.9.7` means an exact version, and `any` matches everything. One helper (`patches.DirRangeExpr`) translates a directory name into the existing `domain.MatchVersionRange` expression, so matching reuses the tested constraint logic. Exact-version directories like pyenv's would force ~30 duplicates per PHP minor series; wildcard dirs (`7.4.x`) would need new constraint syntax. Ranges make one patch file cover a whole minor series. The `patches/` directory in this repository is the **publication source** — contributors author `.patch` files there and CI validates them — but the binary never embeds it (D4); the running CLI reads only from `$PHPV_ROOT` trees, fetched by the sync mechanism (D5).

### D2: Flags in `flags.yaml` (one new dependency: gopkg.in/yaml.v3)
Each range directory MAY contain a `flags.yaml` declaring `extraCFlags: []` and `configureFlags: []`. Strings pass through unchanged so assembler-side placeholder resolution (`{{prefix}}`, `{{source}}`, `{{dep:NAME}}`) is preserved. Config-only entries (curl, libzip) have no `.patch` files, only a manifest. Loader output is the existing `patcher.Patch` struct (`Apply` = patch-tool invocation), so `Prepare` needs no changes. Alternative considered: keeping flags in Go — rejected because it retains the code-to-add-a-patch loop this change exists to eliminate.

### D3: Application via system `patch -p1 --forward -d <root>`; loud failure
`patch` is present on effectively every distro phpv builds on (pyenv has the same requirement). Source root resolution: the extracted tree nests under `silo.SourcePath` (e.g. `sources/php/7.4.33/php-7.4.33/`); the loader reads the first target path from each diff and probes candidate roots (shallow walk, depth ≤ 2) until the path exists — replacing the current `findFile` content walking. On context mismatch the application fails and the assembler aborts with patch file name, package@version, and root (spec: loud failure). `--forward` retains the one legitimate silent case, "already applied". Packages without any `patch` binary get an actionable error mentioning `apt install patch` / `pacman -S patch`.

### D4: Loader = `internal/repository/patches`, two on-disk sources, per-directory precedence
Constructor `NewRepository(syncedRoot, localRoot string, opts...) patcher.PatcherRepository` implementing the existing interface; the binary ships no patch data at all, so no `go:embed` is used anywhere. Merge algorithm: build a `map[dirPath]{source, files...}` walking two roots — user-local (`$PHPV_ROOT/patches-local`) and synced (`$PHPV_ROOT/patches`) — in precedence order; first source to claim a directory wins it wholly (its manifest + files); the later source only adds directories not yet claimed. `PatchesFor` then filters by `MatchVersionRange` exactly like `patcher.Service` does today. Pure Go diff application (e.g. `go-gitdiff`) was considered as a `patch`-binary fallback but adds parser risk for a tool that's already universal.

### D5: Delta-based sync service over GitHub `main`, wired into both the CLI and install
New `patchsync/` package (repository + service in the project's hexagonal style), against the publication source = this repository's `main` branch (patches are pushed there occasionally; consumers don't need binary releases to receive them):

- **Listing:** `GET /repos/supanadit/phpv/git/trees/HEAD?recursive=1` filtered to `patches/` → the remote file list with git blob SHAs at the pinned HEAD SHA (1 API call; the existing GitHub plumbing in `internal/repository/disk/update.go` is the pattern).
- **Delta:** phpv keeps `$PHPV_ROOT/patches/.manifest.json` (path → blob SHA) plus `.revision` and `.lastcheck` metadata. Delta = added/changed/removed vs the manifest. Per-file content downloads run against `raw.githubusercontent.com` at the pinned SHA (not the rate-limited API), staged per-file then renamed into place. Matching manifest entries are never re-downloaded.
- **Scoped delta:** positionals `[package] [version]` (bare value ⇒ version ⇒ `php` package); scope filtering applies `MatchVersionRange` to the remote listing paths' range directories, so `update 7.4` computes the full delta but downloads/applies only matching files.
- **Atomicity:** manifest + `.revision` + `.lastcheck` are written together only after the delta completes; per-file stage-then-rename means a partial run leaves a consistent old manifest that the next run resumes from.
- **Stale-lease (install path):** the assembler's pre-build hook calls `EnsureFresh`: if `.lastcheck` is within the lease (6h default, override via `PHPV_PATCH_LEASE`), the revision check is skipped entirely; otherwise it runs the listing check and only fetches on revision change. Explicit `phpv patch update` ignores the lease (users asking for it want current state).
- **Failure policy split:** CLI `patch update` failure exits non-zero with the cause; the install hook's failure emits a warning event and continues (with the previous tree, or unpatched on a cold machine) — install already requires network for tarball downloads but must not turn a patch-refresh hiccup into a hard failure of an otherwise feasible build.
- **Delivery:** cobra group `phpv patch` with `update` (alias `fetch`), `status`, `list [package] [version]` in `app/phpv.go`, delegating to the sync service; `list` calls the loader in dry-query mode and reports each result's source tree.

### D6: Conversion of existing patches
Each Go patch function becomes `.patch` file(s) produced by: fetching pristine upstream source for representative versions, running the current Go function, and `git diff`ing against pristine. Ranges are split on real drift discovered via `patch --dry-run` across edge versions. Header comments (root-cause notes now in Go source) move into the patch's `Subject:` header, pyenv-style.

## Risks / Trade-offs

- [Unified diffs demand accurate context; silently-tolerant string replace becomes a hard failure on any version in a range whose source drifted] → split ranges on `--dry-run` evidence during conversion (D6); `--fuzz=2` remains the patch tool's default tolerance; failures name the patch file, so range refinement is a file-system edit, not a code change.
- [No embedded patches means a tree-less machine builds unpatched by design (cold install offline, cleared `$PHPV_ROOT`)] → explicit user decision: builds report a clear unpatched outcome instead of failing; installs need the network for source tarballs anyway, so the window where patches would help offline is already gated on connectivity.
- [GitHub API rate limits could stall the auto-sync check] → each install performs at most one listing call and only outside the stale-lease (D5); content fetches always go through `raw.githubusercontent.com`, which is not API-rate-limited; lease failures degrade to the previous tree.
- [`patch` binary absent on exotic minimal images] → explicit preflight error with per-distro install hint; considered but rejected: pure-Go apply fallback for the rare case.
- [Sync trust] → downloads happen only over HTTPS against a content pinned by the remote listing's blob SHAs; the manifest diff keeps partial failures idempotent; user-local tree is never overwritten; managed tree is wholly replaceable, so a bad sync is fixed with a re-run.
- [Last-write-wins quirk in the current `Prepare` flag merging] → preserved behavior is not carried over; loader emits flags in ordinal order and `Prepare` already concatenates, so no new merge rule is introduced.

## Migration Plan

1. Convert the current patches into the `patches/` tree with CI `--dry-run` validation (D6) and land the two-source loader — the binary builds with zero embedded patch data from day one.
2. Land the `patchsync` service (delta listing, manifest, scoped updates, stale-lease) and the `phpv patch` command group; the assembler pre-build hook starts failing-safe → warning.
3. Rewire fx to the loader, delete `memory/patcher.go`, update docs; plugin behavior is code-free from here on (a patch fix is a PR to `patches/`, picked up by any running phpv on its next check).
Rollback: any step reverts to the previous wiring; users restore patches by re-running `phpv patch update` on a fixed publication tree.

## Open Questions

- None material. Range splitting during D6 conversion is empirical work driven by `--dry-run` results, and cannot change the specs or task breakdown.
