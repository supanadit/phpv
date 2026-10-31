# Tasks

## 1. Contracts (domain) & publication tree

- [x] 1.1 Add `gopkg.in/yaml.v3` and define the `flags.yaml` manifest struct (`extraCFlags`, `configureFlags`) with a parse helper; verify a unit test parses a sample manifest including `{{dep:zlib}}` placeholders unchanged
- [x] 1.2 Create the `patches/` publication tree skeleton: one directory per current patch entry (php/oniguruma/libzip/curl/icu with their existing ranges) containing placeholders for patch files and flags manifests, plus a `patches/README.md` declaring it as the sync publication source; verify `go build ./...` succeeds
- [x] 1.3 Verify the binary contains no baked-in patch data: unit test asserts no `go:embed` directive exists and the package exposes patch content only through injected roots; verify `go vet ./...` is clean

## 2. Loader (repository adapter)

- [x] 2.1 Implement `internal/repository/patches` loader walking two roots (synced, user-local) in precedence order with per-directory ownership; verify a table test: user-local dir shadows a synced dir with the same path, unclaimed synced dirs apply, manifest-only dir selects flags
- [x] 2.2 Implement range matching (directory name via `domain.MatchVersionRange`) and ordinal ordering returning existing `patcher.Patch` structs; verify tests cover match/no-match/flags-only/ordering scenarios from the source-patching spec
- [x] 2.3 Implement source-root resolution (shallow probe for first target path from each diff, replacing `findFile`) and the loud `patch -p1 --forward -d <root> -i <file>` apply with already-applied tolerance; verify via test with a temp tree: clean apply, context-mismatch failure message includes the file name and root, pre-patched source reports already-applied
- [x] 2.4 Add missing-`patch`-binary preflight with per-distro install hint, and the unpatched-build outcome path (no matching entries in any source → build proceeds with report); verify the error and report messages name the cause

## 3. Patch conversion

- [x] 3.1 Write the conversion helper script (fetch pristine upstream source, run current Go patch function, `git diff` → `.patch` with pyenv-style Subject header preserving today's root-cause comments); verify each output applies with `patch --dry-run` on pristine source
- [x] 3.2 Generate all `.patch` files and `flags.yaml` manifests for every current entry, splitting ranges listed in design D6 where `--dry-run` shows context drift; verify every generated file survives `patch --dry-run` against representative edge versions per range (matrix recorded in `patches/README.md`)
- [x] 3.3 Verify converted diffs cover each Go function's replacements: for each version in the validation matrix, applying the tree produces byte-identical files to running the old Go patch functions

## 4. Rewiring & removal

- [x] 4.1 Swap fx wiring `memory.NewPatcherRepository` → `patches.NewRepository` (roots injected from config: `$PHPV_ROOT/patches`, `$PHPV_ROOT/patches-local`) in `app/phpv.go`; verify `make test` passes and `patcher/service_test.go` still passes unchanged
- [x] 4.2 Delete `internal/repository/memory/patcher.go` and unused helpers; verify `golangci-lint`/`go vet` report no dead code and `make build` succeeds
- [x] 4.3 Verify an end-to-end `phpv install 8.0.30` (and one 7.4 build) applies the converted patches against a locally-seeded `$PHPV_ROOT/patches/`, emitting the applied-patch count/names event from the assembler

## 5. Patch sync (delta service + delivery)

- [x] 5.1 Implement `patchsync/` repository adapter: Git Trees listing of `patches/` at the pinned `main` HEAD SHA, local `.manifest.json` (path → blob SHA) diff computing the delta, per-file `raw.githubusercontent.com` staged downloads, deletion application, atomic manifest/revision/lastcheck commit; verify with a fake remote: first sync downloads everything, second delta downloads only changed files, interrupted run resumes cheaply
- [x] 5.2 Implement the stale-lease check (`.lastcheck` within 6h default / `PHPV_PATCH_LEASE` override skips the listing) and the scoped sync positionals (`[package] [version]`, bare version ⇒ `php`) filtering delta paths via `MatchVersionRange`; verify: lease expiry triggers a check, scoped `update 7.4` touches only matching files (spec scenarios as tests)
- [x] 5.3 Implement cobra `phpv patch` group: `update` (alias `fetch`) with scope, `status` (revision, check time, counts), `list [package] [version]` (ordered files with source tree labels); verify `phpv patch status` reports recorded vs remote revision and `phpv patch list php 8.0.30` lists ordered patch files with each source tree
- [x] 5.4 Implement the assembler pre-build `EnsureFresh` hook (lease-aware check, delta on change, warn-and-continue on failure); verify: install emits the warning event when the remote is unreachable and starts the build with the previous tree on a synced machine, and builds unpatched with a report on a cold machine (offline behavior split per spec)
- [x] 5.5 Verify sync never touches `$PHPV_ROOT/patches-local/` and replaces renamed range directories in the managed tree (spec scenarios as tests); also verify explicit `phpv patch update` fails non-zero with a clear message when offline, unlike the install path

## 6. Docs & integration

- [x] 6.1 README: "How patching works" (synced-from-`main` tree, stale-lease, unpatched-with-warning semantics) + "Adding a patch" (PR to `patches/`, or instant override via `patches-local/`) + `phpv patch` command docs; verify documented commands run as written
- [x] 6.2 `doctor` gains the patch-tree line (managed tree present? revision? lease status?); verify `phpv doctor` output shows the state for both synced and never-synced machines
- [x] 6.3 Integration pass: `make test`, then a full install run where an outdated-binary-like machine receives a just-published patch through install-time delta sync; verify builds consume the fetched patches per precedence, and an offline cold machine emits the unpatched report without failing

## Workflow follow-up

- Archive the change after the project's review requirements are satisfied.
- Verify the archived result.
