# Proposal

## Why

phpv's patches for modern-toolchain build failures are hardcoded Go functions doing fragile string surgery inside `internal/repository/memory/patcher.go` (~540 lines of `strings.Replace` calls with inline version ranges plus "skip silently if pattern not found" behavior). A fix requires Go knowledge and recompiling the binary. pyenv demonstrates a better model: patches are ordinary unified diff files in a conventioned directory tree (`patches/<version>/<package>/*.patch`), applied with `patch -p1`, so a patch is a meta-less, versionable file contributors can read, audit, and add without touching code.

## What Changes

- Introduce a file-based patch tree at `patches/<package>/<version-range>/NNN-slug.patch`, using real unified diff files applied with `patch -p1`, with optional `flags.yaml` per directory declaring `ExtraCFlags` / `ConfigureFlags` (placeholder support such as `{{dep:zlib}}` is preserved). The repository tree is the publication source for authoring and CI validation; it is never embedded in the binary — **phpv ships zero patches baked in**, so patch files are runtime data (**BREAKING**).
- Replace the in-memory patcher repository with a loader that matches range directories via the existing `domain.MatchVersionRange` syntax, merges two on-disk patch sources with per-directory precedence (user-local → synced), and fails loudly when a patch does not apply.
- Add runtime sync (`phpv patch update` / `fetch`) that pulls the patch tree from this repository's GitHub `main` branch into `$PHPV_ROOT/patches/`. Sync is delta-based: the local patch manifest is diffed against the remote tree listing at the latest revision and only added/changed files are downloaded (deletions are applied). An optional scope argument limits the delta (`phpv patch update 7.4` syncs only PHP 7.4 patches). Add `phpv patch status` and `phpv patch list`.
- Make `phpv install` self-updating for patches: before building, it checks the remote revision (skip re-check while within a short stale-lease window) and syncs when changed; on fetch failure it warns and continues without downloaded patches. Because patches are published to `main`, users with an outdated binary still receive current patches automatically.
- **BREAKING**: convert all existing Go patch functions into `.patch` files; the silent-skip behavior of unpatched/unmatched source code is replaced by loud patch-application failures. The `patcher.PatcherRepository` service contract (`PatchesFor`, `Prepare`) is unchanged; `internal/repository/memory/patcher.go` is deleted.
- A second managed source root `$PHPV_ROOT/patches-local/` holds user-authored patches that sync never touches.

## Capabilities

### New Capabilities

- `source-patching`: How phpv discovers, selects (by package and version range), applies, and reports patches against package source trees before building — including merged-source precedence and loud failure semantics. Patches are never shipped inside the binary; they are runtime data living under `PHPV_ROOT`.
- `patch-sync`: How phpv keeps the patch tree on `PHPV_ROOT` current at runtime (install-time auto-sync plus update/fetch/status/list commands, delta-based downloads, stale-lease check skipping, atomic manifest commit, lenient offline behavior for installs).

### Modified Capabilities

- (none — no existing spec requirements change; `system-package-detection` covers system package reporting, which is unaffected)

## Impact

- **Code**: delete `internal/repository/memory/patcher.go`; add `internal/repository/patches` loader (two-root filesystem merge) and a patch-sync service; rewire `app/phpv.go` fx providers and cobra command registration; assembler gains a pre-build ensure-fresh-patches hook.
- **Architecture mapping** (per project standards): `domain` — patch entity & repository contract already partially in `patcher/` (Name/Package/VersionRange semantics, `MatchVersionRange` reuse, no new framework deps); the loader and sync service are repository-side adapters; `phpv patch` commands and the installer's auto-sync hook are the delivery layer delegating to the sync service.
- **Data**: the `patches/` tree in this repo becomes the publication source (never embedded); new `patches/.manifest.json` + `.revision` metadata and `$PHPV_ROOT/patches/` + `$PHPV_ROOT/patches-local/` trees at runtime; yaml.v3 dependency for `flags.yaml`; per-file downloads served from `raw.githubusercontent.com` at the pinned revision (tree listing via the GitHub Trees API).
- **Behavior**: patch application failures now abort the build with the patch file and source root in the message (previously silent skips); a machine with no synced tree (cold install, offline first-run) builds unpatched after a warning. Installs already require network for source downloads, so patch freshness rides the same connection instead of the binary.
- **Docs/CI**: README "How patching works" + "Adding a patch" contribution guide; `patch --dry-run` validation of every `.patch` against range edge versions.
