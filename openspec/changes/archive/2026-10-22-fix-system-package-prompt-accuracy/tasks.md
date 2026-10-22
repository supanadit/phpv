# Tasks

## 1. Refresh package state after install

- [x] 1.1 In `checkSystemDeps` (`internal/terminal/php.go:1034`), replace the
  loop that sets `p.Installed = true` on `result.Missing` with a re-run of
  `h.systemSvc.Check(phpDeps)` after a successful install, rebuilding
  `systemAvail` from the fresh `Available` list; verify with a fake
  `SystemService` test in `internal/terminal` that a package which becomes
  installed after the prompt is reported with `Installed == true` **and** a
  non-empty `Version`.
- [x] 1.2 Add the companion test that pins the failure mode: fake `Check`
  returning an empty `Version` after refresh must leave the package out of the
  returned map (or marked unusable), so the assembler gate cannot treat it as a
  usable system package. Verify the test fails against the current
  `p.Installed = true` loop and passes after 1.1.
- [x] 1.3 Cover the declined/failed paths: fake install returning an error and
  the prompt being declined must both leave the returned map reflecting only
  what `Check` actually reports. Verify with two table-driven cases in the same
  test file.

## 2. Derive the prompted dependency set from build knowledge

- [x] 2.1 Add `Required bool` to `system.Package` (`system/types.go`) with a doc
  comment stating it means "known build dependency whose version will be
  checked before use"; verify `go build ./...` and that
  `doctor/service.go:236` and the `assembler/service_test.go` fixtures still
  compile unchanged (named-field construction).
- [x] 2.2 Replace the hardcoded `phpDeps` array in `checkSystemDeps` with the
  build-plan-derived set (dependency names from `RequiresPackage` for the
  requested extensions) plus an explicitly named advisory list for dev
  libraries with no buildable package definition; drop `argon2`; set
  `Required` accordingly. Extract that derivation into one exported function so
  `doctor` can share it. Verify with a unit test asserting that a library with
  no `builtInPackages` entry and no `RequiresPackages` link is absent from the
  `Check` input, and that a `RequiresPackages`-declared library is present.
- [x] 2.3 Verify `extension add` inherits both fixes by exercising
  `checkSystemDeps` through the `extensionAdd` path with the fake system
  service; verify `go test ./internal/terminal` passes.

## 3. Align doctor with the build's package set

- [x] 3.1 Rework `doctor.checkSystemPackages` (`doctor/service.go:152`) to use
  the shared derived dependency set from 2.2 instead of its own hardcoded array
  (which still contains `argon2`); verify with a `doctor` test that `argon2` is
  absent from the dependency input on a system without it.
- [x] 3.2 Split doctor's system-library issue into missing build dependencies
  (warning severity, names the constraint) and missing advisory dev libraries
  (informational, advisory wording), from a single `Check` call; verify with a
  `doctor/service_test.go` table test using the fake system service that a
  required package and an advisory package produce two distinct issues.
- [x] 3.3 Confirm doctor and install agree: verify with a test that both derive
  their dependency set from the same function, so a package added to one cannot
  silently diverge from the other.

## 4. Make the prompt honest

- [x] 4.1 Parameterize `promptAndInstall` (`internal/terminal/php.go:1097`) so
  dependency prompts print the version-gate sentence ("used only when the
  installed version satisfies the build requirement, otherwise built from
  source") and build-tool prompts do not; verify by capturing stdout in a test
  and asserting the sentence appears for a dependency prompt and is absent for a
  build-tool prompt.
- [x] 4.2 Mark advisory packages distinctly from required ones in the missing
  list output; verify with a test asserting a `Required: false` entry renders
  differently from a `Required: true` entry.
- [x] 4.3 Confirm the non-interactive path is unchanged: with stdin not a
  terminal and `--auto-deps` unset, no install is attempted and the message is
  emitted; verify with an existing or added test using a non-char-device stdin.

## 5. Integration verification

- [x] 5.1 Run `go build ./... && go vet ./... && go test ./...` and confirm the
  only failure is the pre-existing `compiler` package test needing a `clang`
  binary on the host; record the exact output.
- [x] 5.2 Manually run `phpv install 8 --dry-run` on this machine and confirm the
  reported missing list no longer includes `argon2`, and that any listed
  package is one the build plan can consume; paste the observed list into the
  change summary.
