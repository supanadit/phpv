# patches/

This directory is the **publication source** for phpv's runtime patch tree.
It is never embedded in the phpv binary; users receive it via
`phpv patch update` (delta sync from this repository's `main` branch) into
`$PHPV_ROOT/patches/`.

## Layout

```
patches/<package>/<range-directory>/<ordinal>-<slug>.patch
patches/<package>/<range-directory>/flags.yaml       (optional)
```

- `<package>`: php, oniguruma, libzip, curl, icu.
- `<range-directory>`: a filesystem-safe encoding of a version range
  (valid on Windows, no reserved characters):
  - `8.0.0-8.1.0` — from 8.0.0 (inclusive) to 8.1.0 (exclusive)
  - `6.0.0+` — 6.0.0 and everything above
  - `6.9.7` — exactly 6.9.7
  - `any` — every version of the package
- Patch files are **unified diffs** applied with `patch -p1` against the
  package's extracted source root. Ordinals (NNN-) define application order.
- `flags.yaml` optionally declares `extraCFlags` and `configureFlags`;
  placeholders `{{prefix}}`, `{{source}}`, `{{dep:NAME}}` are resolved by
  the assembler at build time. A directory with only `flags.yaml` is valid.

## Adding a patch

1. Commit a unified diff (generated with `git diff` or `diff -u`) named
   `NNN-slug.patch` under the right package and version range.
2. CI validates every `.patch` with `patch --dry-run` against
   representative versions of its range.
3. Users pick it up on their next `phpv install` or `phpv patch update` —
   no phpv release needed. For instant local overrides, users can place
   patches under `$PHPV_ROOT/patches-local/` (same layout; takes
   precedence per directory).

## Validation matrix

Every `.patch` was generated from pristine upstream source (converted from
phpv's historical in-memory patch functions, byte-identical output verified)
and validated with `patch --dry-run` against these versions:

| Patch | Generated from | Dry-run validated on |
|---|---|---|
| `php/5.0.0-7.0.0/0001-configure-curl-cflags.patch` | 5.6.40 | 5.6.40 |
| `php/7.0.0-7.1.0/0001-configure-curl-cflags.patch` | 7.0.33 | 7.0.33 |
| `php/5.0.0-6.0.0/0001-bison3-yystrlen.patch` | 5.6.40 | 5.6.40 |
| `php/8.0.0-8.1.0/0001-intl-cxx17.patch` | 8.0.30 | 8.0.30, 8.0.0 |
| `oniguruma/5.0.0-6.0.0/0001-st-prototypes.patch` | 5.9.6 | 5.9.6 |
| `oniguruma/6.0.0-6.9.9/0001-st-foreach.patch` | 6.9.7 | 6.9.7, 6.9.8 |
| `oniguruma/6.9.9-6.9.10/0001-st-foreach.patch` | 6.9.9 | 6.9.9 |
| `icu/58.0-59.0/0001-gcc15-compat.patch` | 58.2 | 58.2 |

Notes from conversion:

- The historical `php-gcc15-scanf-fn` function matched no PHP 7.x source
  (its tab-indented pattern never existed in released 7.0–7.4 trees), so it
  was a silent no-op and has no file equivalent.
- The 7.0.33 curl-cflags diff does not apply to 5.6.40 (context drift in the
  generated configure), hence the split into `5.0.0-7.0.0` and
  `7.0.0-7.1.0`.
- Oniguruma 6.9.9 fixed some st.h issues upstream (but not all), hence the
  split at 6.9.9; 6.9.10+ needs no patch at all.
- ICU 59.x was not covered by the historical patch range in practice (the
  old function skipped silently when patterns were absent); the converted
  patch is scoped to `58.0-59.0`.

To regenerate: `go run ./scripts/convert-patches` (add `-validate-only` for
just the dry-run matrix, `-equivalence` for the byte-identity check).
