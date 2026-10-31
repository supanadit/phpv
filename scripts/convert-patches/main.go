// Command convert-patches converts phpv's historical in-memory Go patch
// functions into the file-based patches/ tree (change: file-based-patch-system).
//
// For every entry below it:
//  1. downloads the pristine upstream source for a representative version,
//  2. copies the extracted tree into a/ (pristine) and b/ (patched),
//  3. runs the current Go patch function on b/ via the public patcher API,
//  4. records `git diff --no-index a b` as a unified diff, post-processed
//     to -p1 form and wrapped in a pyenv-style mail header (Subject:
//     carries the root-cause comment from the Go source),
//  5. validates the result with `patch --dry-run` against every version
//     in the entry's validation matrix (the generation version included).
//
// Usage: go run ./scripts/convert-patches [-validate-only] [-out patches]
//
// Downloads go into their own fresh temporary directories.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type entry struct {
	patchName   string   // patcher.Patch.Name to select
	outFile     string   // output path inside the patches/ tree (slash form)
	description string   // root-cause text for the Subject header
	genVersion  string   // version the diff is generated from
	validate    []string // dry-run validation matrix (includes genVersion)
	url         func(v string) string
}

func phpURL(v string) string { return "https://www.php.net/distributions/php-" + v + ".tar.gz" }
func onigURL(v string) string {
	return "https://github.com/kkos/oniguruma/releases/download/v" + v + "/onig-" + v + ".tar.gz"
}
func icuURL(v string) string {
	return "https://github.com/unicode-org/icu/releases/download/release-" + strings.ReplaceAll(v, ".", "-") + "/icu4c-" + strings.ReplaceAll(v, ".", "_") + "-src.tgz"
}

var pointerFlags = "-Wno-error=incompatible-pointer-types\n  -Wno-incompatible-pointer-types"

var entries = []entry{
	{
		patchName: "php-configure-curl-cflags-warnings",
		outFile:   "php/7.0.0-7.1.0/0001-configure-curl-cflags.patch",
		description: "PHP 7.0's generated configure clobbers CFLAGS during the 'checking " +
			"for openssl support in libcurl' test with `curl-config --cflags` output " +
			"(just an -I path). The conftest uses strncasecmp() without <strings.h>, " +
			"a hard error on GCC 14+; append warning-suppression flags to the " +
			"substitution so the test compiles clean.",
		genVersion: "7.0.33",
		validate:   []string{"7.0.33"},
		url:        phpURL,
	},
	{
		patchName: "php-configure-curl-cflags-warnings",
		outFile:   "php/5.0.0-7.0.0/0001-configure-curl-cflags.patch",
		description: "PHP 5.x's generated configure clobbers CFLAGS during the 'checking " +
			"for openssl support in libcurl' test with `curl-config --cflags` output " +
			"(just an -I path). The conftest uses strncasecmp() without <strings.h>, " +
			"a hard error on GCC 14+; append warning-suppression flags to the " +
			"substitution so the test compiles clean.",
		genVersion: "5.6.40",
		validate:   []string{"5.6.40"},
		url:        phpURL,
	},
	{
		patchName: "php-bison3-yystrlen",
		outFile:   "php/5.0.0-6.0.0/0001-bison3-yystrlen.patch",
		description: "PHP 5.x's pre-generated bison parser references yystrlen/yystpcpy, " +
			"which bison 3.x removed from generated code. Add fallback defines " +
			"to the parser source.",
		genVersion: "5.6.40",
		validate:   []string{"5.6.40"},
		url:        phpURL,
	},
	{
		patchName: "php-intl-cxx17",
		outFile:   "php/8.0.0-8.1.0/0001-intl-cxx17.patch",
		description: "PHP 8.0's ext/intl/config.m4 hardcodes a C++11 standard check, but " +
			"ICU 74+ headers require C++17. The generated configure assigns " +
			"PHP_INTL_STDCXX from a feature test; override it to -std=gnu++17.",
		genVersion: "8.0.30",
		validate:   []string{"8.0.30", "8.0.0"},
		url:        phpURL,
	},
	{
		patchName: "oniguruma-gcc15-st_prototypes",
		outFile:   "oniguruma/5.0.0-6.0.0/0001-st-prototypes.patch",
		description: "Oniguruma 5.9.x has K&R-style function pointer declarations in st.h, " +
			"st.c, and oniguruma.h that modern GCC rejects as errors. Add proper " +
			"prototypes matching the actual call sites.",
		genVersion: "5.9.6",
		validate:   []string{"5.9.6"},
		url:        onigURL,
	},
	{
		patchName: "oniguruma-gcc15-st_foreach",
		outFile:   "oniguruma/6.0.0-6.9.9/0001-st-foreach.patch",
		description: "Oniguruma 6.0-6.9.8 has K&R-style function pointer declarations in " +
			"st.h/st.c (st_hash_type members, ST_NUMCMP/ST_NUMHASH macros, " +
			"st_foreach ANYARGS prototype, local func pointer) that modern GCC " +
			"(14+/C23) rejects. Upstream fixed these in 6.9.9+.",
		genVersion: "6.9.7",
		validate:   []string{"6.9.7", "6.9.8"},
		url:        onigURL,
	},
	{
		patchName: "oniguruma-gcc15-st_foreach",
		outFile:   "oniguruma/6.9.9-6.9.10/0001-st-foreach.patch",
		description: "Oniguruma 6.9.9 fixed the st_hash_type members upstream but still " +
			"uses _()/ANYARGS macro declarations for st_foreach and the K&R local " +
			"func pointer in st.c; fix the remaining ones. Fixed fully in 6.9.10.",
		genVersion: "6.9.9",
		validate:   []string{"6.9.9"},
		url:        onigURL,
	},
	{
		patchName: "icu-gcc15-compat",
		outFile:   "icu/58.0-59.0/0001-gcc15-compat.patch",
		description: "ICU 58.x on modern toolchains: xlocale.h was removed in glibc 2.26+ " +
			"(merged into locale.h), and -Wimplicit-fallthrough is an error on " +
			"GCC 15+.",
		genVersion: "58.2",
		validate:   []string{"58.2"},
		url:        icuURL,
	},
}

func main() {
	validateOnly := flag.Bool("validate-only", false, "only run the dry-run validation matrix")
	equivalence := flag.Bool("equivalence", false, "verify tree application matches legacy Go function output byte-for-byte")
	out := flag.String("out", "patches", "output patches/ directory")
	flag.Parse()

	ctx := context.Background()
	repo := newPatcherRepo()
	var failures int
	for _, e := range entries {
		fmt.Printf("== %s (%s)\n", e.outFile, e.patchName)
		if *equivalence {
			if err := checkEquivalence(ctx, repo, e); err != nil {
				fmt.Printf("   EQUIVALENCE FAILED: %v\n", err)
				failures++
			} else {
				fmt.Printf("   equivalent on %s\n", e.genVersion)
			}
			continue
		}
		var patchFile string
		if !*validateOnly {
			var err error
			patchFile, err = generate(ctx, repo, e, *out)
			if err != nil {
				fmt.Printf("   GENERATE FAILED: %v\n", err)
				failures++
				continue
			}
			fmt.Printf("   generated %s\n", patchFile)
		} else {
			patchFile = filepath.Join(*out, filepath.FromSlash(e.outFile))
		}
		for _, v := range e.validate {
			if err := validate(ctx, e, v, patchFile); err != nil {
				fmt.Printf("   VALIDATION FAILED for %s: %v\n", v, err)
				failures++
				continue
			}
			fmt.Printf("   dry-run OK on %s\n", v)
		}
	}
	if failures > 0 {
		fmt.Printf("\n%d failure(s)\n", failures)
		os.Exit(1)
	}
	fmt.Println("\nall entries generated and validated")
}
