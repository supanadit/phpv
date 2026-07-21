package memory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/supanadit/phpv/patcher"
)

// PatcherRepository returns patches to apply to extracted source trees
// before building. Most patches exist to make old C code compile on modern
// toolchains (GCC 14+ defaults to -std=gnu23, which is stricter about
// function pointer types).
// inMemoryPatcher is the default in-memory patcher for build compatibility
// on modern toolchains.
type inMemoryPatcher struct{}

func NewPatcherRepository() patcher.PatcherRepository {
	return &inMemoryPatcher{}
}

func (p *inMemoryPatcher) PatchesFor(name string, version string) []patcher.Patch {
	switch name {
	case "oniguruma":
		// Oniguruma 5.9.x (used by PHP 5.x and 7.0) has K&R-style function
		// pointer declarations in st.h that modern GCC rejects as errors.
		// Oniguruma 6.9.x has a different st_foreach issue fixed by the
		// other patch below.
		return []patcher.Patch{
			{
				Name:         "oniguruma-gcc15-st_prototypes",
				Package:      "oniguruma",
				VersionRange: ">=5.0.0, <6.0.0",
				Apply:        patchOnigurumaStPrototypes,
				ExtraCFlags:  []string{"-Wno-error=incompatible-pointer-types", "-Wno-incompatible-pointer-types"},
			},
			{
				Name:         "oniguruma-gcc15-st_foreach",
				Package:      "oniguruma",
				VersionRange: ">=6.0.0",
				Apply:        patchOnigurumaStForeach,
				ExtraCFlags:  []string{"-Wno-error=incompatible-pointer-types", "-Wno-incompatible-pointer-types"},
			},
		}
	case "php":
		// PHP 7.4's scanf.c uses K&R-style function pointer casts that GCC 15
		// rejects outright. Patch the fn declaration to match the actual
		// strtoll(str, endptr, base) signature. The call site passes 3 args.
		patches := []patcher.Patch{{
			Name:         "php-gcc15-scanf-fn",
			Package:      "php",
			VersionRange: ">=7.0.0, <8.0.0",
			Apply:        patchPhpScanfFn,
		}}
		// PHP 5.x and 7.0's generated configure script clobbers CFLAGS during
		// the "checking for openssl support in libcurl" test by replacing them
		// with the output of `curl-config --cflags` (which is just an -I path).
		// The conftest uses strncasecmp() without including <strings.h>, which
		// is a hard error on GCC 14+ (implicit function declarations default
		// to error). Append warning-suppression flags to the curl-config
		// substitution so the test compiles clean.
		patches = append(patches, patcher.Patch{
			Name:         "php-configure-curl-cflags-warnings",
			Package:      "php",
			VersionRange: ">=5.0.0, <7.1.0",
			Apply:        patchPhpConfigureCurlCflags,
		})
		// PHP 5.x's pre-generated bison parser uses yystrlen/yystpcpy which
		// are not defined in bison 3.x. Add defines to the parser source.
		patches = append(patches, patcher.Patch{
			Name:         "php-bison3-yystrlen",
			Package:      "php",
			VersionRange: ">=5.0.0, <6.0.0",
			Apply:        patchPhpBison3Compat,
		})
		// PHP 8.0's ext/intl/config.m4 hardcodes C++11 for the intl extension,
		// but ICU 74+ (including Arch's system ICU 78.3) requires C++17 headers.
		// The generated configure script sets PHP_INTL_STDCXX from a feature test;
		// override it to -std=gnu++17 so the extension compiles against modern ICU.
		patches = append(patches, patcher.Patch{
			Name:         "php-intl-cxx17",
			Package:      "php",
			VersionRange: ">=8.0.0, <8.1.0",
			Apply:        patchPhpIntlCxx17,
		})
		return patches
	case "curl":
		// Curl needs explicit TLS backend + disabled optional features.
		// The {{dep:openssl}} placeholder is resolved by the assembler to
		// the openssl install prefix.
		return []patcher.Patch{{
			Name:    "curl-openssl-and-disable-extras",
			Package: "curl",
			ConfigureFlags: []string{
				"--with-openssl={{dep:openssl}}",
				"--without-brotli",
				"--disable-ldap",
				"--without-libpsl",
				"--without-libidn2",
				"--without-zstd",
				"--without-nghttp2",
				"--without-zlib",
			},
		}}
	case "icu":
		// ICU 58.x has two issues on modern toolchains:
		// 1. xlocale.h was removed in glibc 2.26+ (merged into locale.h).
		// 2. -Wimplicit-fallthrough warnings are treated as errors on GCC 15+.
		return []patcher.Patch{{
			Name:         "icu-gcc15-compat",
			Package:      "icu",
			VersionRange: ">=58.0, <60.0",
			Apply:        patchIcu58Compat,
		}}
	}
	return nil
}

// patchPhpScanfFn rewrites the bad fn typedef in PHP 7.4's scanf.c so the
// ZEND_STRTOL_PTR function pointer can be called with 3 arguments.
func patchPhpScanfFn(sourceDir string) error {
	scanfs, err := findFile(sourceDir, "ext/standard/scanf.c")
	if err != nil {
		return err
	}
	data, err := os.ReadFile(scanfs)
	if err != nil {
		return err
	}
	old := "zend_long	(*fn)();"
	newStr := "zend_long	(*fn)(char *, char **, int);"
	if !strings.Contains(string(data), old) {
		// Already patched or different signature; skip.
		return nil
	}
	if err := os.WriteFile(scanfs, []byte(strings.Replace(string(data), old, newStr, 1)), 0o644); err != nil {
		return err
	}
	return nil
}

// patchPhpIntlCxx17 forces PHP 8.0's intl extension to compile its C++ sources
// with -std=gnu++17. PHP 8.0's ext/intl/config.m4 hardcodes a C++11 standard
// check, but ICU 74+ headers require C++17 features. The generated configure
// script assigns PHP_INTL_STDCXX from the feature test; we override the
// assignment so the extension always uses C++17 on modern toolchains.
func patchPhpIntlCxx17(sourceDir string) error {
	configurePath, err := findFile(sourceDir, "configure")
	if err != nil {
		return err
	}
	data, err := os.ReadFile(configurePath)
	if err != nil {
		return err
	}
	content := string(data)
	// The assignment is indented inside the expanded macro; match any leading
	// whitespace so the patch is robust to changes in the generated script.
	oldPattern := regexp.MustCompile(`(?m)^\s+eval PHP_INTL_STDCXX="\$switch"`)
	newStr := "        eval PHP_INTL_STDCXX=\"-std=gnu++17\""
	if !oldPattern.MatchString(content) {
		// Already patched or the generated configure differs; skip silently.
		return nil
	}
	content = oldPattern.ReplaceAllString(content, newStr)
	return os.WriteFile(configurePath, []byte(content), 0o644)
}

func findFile(root, rel string) (string, error) {
	// Walk source dir looking for the relative path. The extracted source is
	// nested under e.g. sources/php/7.4.33/php-7.4.33/.
	var found string
	want := filepath.ToSlash(rel)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		p := filepath.ToSlash(path)
		// Match either the exact file name (e.g. "configure") or a path that
		// ends with the requested relative path (e.g. "ext/standard/scanf.c").
		// The previous substring check false-matched directory names containing
		// the target name, such as .github/actions/configure-macos/action.yml
		// when looking for "configure".
		if filepath.Base(p) == want || strings.HasSuffix(p, "/"+want) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("not found: %s under %s", rel, root)
	}
	return found, nil
}

// patchOnigurumaStPrototypes fixes the K&R-style function pointer declarations
// in oniguruma 5.9.x's st.h, st.c, and oniguruma.h. Modern GCC interprets
// `int (*hash)()` as "takes no arguments" and rejects calls with arguments.
// The fix adds proper prototypes matching the actual call sites.
func patchOnigurumaStPrototypes(sourceDir string) error {
	var stPath, stcPath, onigHPath string
	entries, _ := os.ReadDir(sourceDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(sourceDir, e.Name())
		candidates := []string{
			filepath.Join(sub, "st.h"),
			filepath.Join(sub, "src", "st.h"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				stPath = c
				break
			}
		}
		stcCandidates := []string{
			filepath.Join(sub, "st.c"),
			filepath.Join(sub, "src", "st.c"),
		}
		for _, c := range stcCandidates {
			if _, err := os.Stat(c); err == nil {
				stcPath = c
				break
			}
		}
		onigHCandidates := []string{
			filepath.Join(sub, "oniguruma.h"),
			filepath.Join(sub, "src", "oniguruma.h"),
		}
		for _, c := range onigHCandidates {
			if _, err := os.Stat(c); err == nil {
				onigHPath = c
				break
			}
		}
		if stPath != "" && stcPath != "" && onigHPath != "" {
			break
		}
	}
	if stPath == "" {
		return fmt.Errorf("patchOnigurumaStPrototypes: st.h not found in %s", sourceDir)
	}

	// Patch st.h: fix struct st_hash_type function pointer prototypes.
	data, err := os.ReadFile(stPath)
	if err != nil {
		return err
	}
	content := string(data)
	content = strings.Replace(content,
		"    int (*compare)();\n    int (*hash)();",
		"    int (*compare)(st_data_t, st_data_t);\n    int (*hash)(st_data_t);", 1)
	content = strings.Replace(content,
		"#define ST_NUMCMP\t((int (*)()) 0)",
		"#define ST_NUMCMP\t((int (*)(st_data_t, st_data_t)) 0)", 1)
	content = strings.Replace(content,
		"#define ST_NUMHASH\t((int (*)()) -2)",
		"#define ST_NUMHASH\t((int (*)(st_data_t)) -2)", 1)
	// Fix st_foreach declaration: replace the _()/ANYARGS macro-based
	// K&R-style declaration with a proper prototype.
	content = strings.Replace(content,
		"int st_foreach _((st_table *, int (*)(ANYARGS), st_data_t));",
		"int st_foreach(st_table *, int (*)(st_data_t, st_data_t, st_data_t), st_data_t);", 1)
	if err := os.WriteFile(stPath, []byte(content), 0o644); err != nil {
		return err
	}

	// Patch st.c: fix the local function pointer declaration in st_foreach.
	if stcPath != "" {
		data, err := os.ReadFile(stcPath)
		if err != nil {
			return err
		}
		content := string(data)
		content = strings.Replace(content,
			"    int (*func)();",
			"    int (*func)(st_data_t, st_data_t, st_data_t);", 1)
		if err := os.WriteFile(stcPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	// Patch oniguruma.h: force PV_ macro to use proper prototypes.
	if onigHPath != "" {
		data, err := os.ReadFile(onigHPath)
		if err != nil {
			return err
		}
		content := string(data)
		content = strings.Replace(content,
			"#ifndef PV_\n#ifdef HAVE_STDARG_PROTOTYPES\n# define PV_(args) args\n#else\n# define PV_(args) ()\n#endif\n#endif",
			"#ifndef PV_\n# define PV_(args) args\n#endif", 1)
		if err := os.WriteFile(onigHPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

// patchOnigurumaStForeach fixes K&R-style function pointer declarations in
// oniguruma 6.x's st.h/st.c that modern GCC (14+/C23) rejects. Upstream
// fixed these in 6.9.10+; we backport all the necessary changes to 6.0–6.9.x.
//
// This patch covers: st_hash_type members (compare/hash), ST_NUMCMP/ST_NUMHASH
// macros, st_foreach ANYARGS prototype, st.c local func pointer, and
// oniguruma.h PV_ macro. Each replacement is conditional so it's safe on
// versions where some fixes already exist upstream (e.g. 6.9.9 has proper
// st_hash_type but still uses _()/ANYARGS macros).
func patchOnigurumaStForeach(sourceDir string) error {
	var stPath, stcPath, onigHPath string
	entries, _ := os.ReadDir(sourceDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(sourceDir, e.Name())
		for _, c := range []string{filepath.Join(sub, "st.h"), filepath.Join(sub, "src", "st.h")} {
			if _, err := os.Stat(c); err == nil {
				stPath = c
				break
			}
		}
		for _, c := range []string{filepath.Join(sub, "st.c"), filepath.Join(sub, "src", "st.c")} {
			if _, err := os.Stat(c); err == nil {
				stcPath = c
				break
			}
		}
		for _, c := range []string{filepath.Join(sub, "oniguruma.h"), filepath.Join(sub, "src", "oniguruma.h")} {
			if _, err := os.Stat(c); err == nil {
				onigHPath = c
				break
			}
		}
		if stPath != "" {
			break
		}
	}
	for _, c := range []string{filepath.Join(sourceDir, "src", "st.h"), filepath.Join(sourceDir, "st.h")} {
		if stPath == "" {
			if _, err := os.Stat(c); err == nil {
				stPath = c
			}
		}
	}
	if stPath == "" {
		return fmt.Errorf("patchOnigurumaStForeach: st.h not found in %s", sourceDir)
	}

	data, err := os.ReadFile(stPath)
	if err != nil {
		return err
	}
	content := string(data)

	// Fix struct st_hash_type: K&R () → proper prototypes.
	// Oniguruma 6.9.8 still has these; 6.9.9 already fixed them upstream.
	content = strings.Replace(content,
		"    int (*compare)();\n    int (*hash)();",
		"    int (*compare)(st_data_t, st_data_t);\n    int (*hash)(st_data_t);", 1)

	// Fix ST_NUMCMP/ST_NUMHASH casts.
	content = strings.Replace(content,
		"#define ST_NUMCMP\t((int (*)()) 0)",
		"#define ST_NUMCMP\t((int (*)(st_data_t, st_data_t)) 0)", 1)
	content = strings.Replace(content,
		"#define ST_NUMHASH\t((int (*)()) -2)",
		"#define ST_NUMHASH\t((int (*)(st_data_t)) -2)", 1)

	// Fix st_foreach: replace the _()/ANYARGS macro-based K&R declaration.
	// Old: int st_foreach _((st_table *, int (*)(ANYARGS), st_data_t));
	// New: int st_foreach(st_table *, int (*)(st_data_t, st_data_t, st_data_t), st_data_t);
	oldPattern := regexp.MustCompile(`int\s+st_foreach\s+_\(\(st_table\s*\*\s*,\s*int\s*\(\*\)\(ANYARGS\)\s*,\s*st_data_t\)\)\s*;`)
	content = oldPattern.ReplaceAllString(content,
		"int st_foreach(st_table *, int (*)(st_data_t, st_data_t, st_data_t), st_data_t);")

	if err := os.WriteFile(stPath, []byte(content), 0o644); err != nil {
		return err
	}

	// Fix st.c: K&R local function pointer declarations.
	if stcPath != "" {
		data, err := os.ReadFile(stcPath)
		if err != nil {
			return err
		}
		content := string(data)
		content = strings.Replace(content,
			"    int (*func)();",
			"    int (*func)(st_data_t, st_data_t, st_data_t);", 1)
		if err := os.WriteFile(stcPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	// Fix oniguruma.h: force PV_ macro to use proper prototypes instead of ().
	if onigHPath != "" {
		data, err := os.ReadFile(onigHPath)
		if err != nil {
			return err
		}
		content := string(data)
		content = strings.Replace(content,
			"#ifndef PV_\n#ifdef HAVE_STDARG_PROTOTYPES\n# define PV_(args) args\n#else\n# define PV_(args) ()\n#endif\n#endif",
			"#ifndef PV_\n# define PV_(args) args\n#endif", 1)
		if err := os.WriteFile(onigHPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

// patchPhpConfigureCurlCflags adds -Wno-implicit-function-declaration to the
// CFLAGS set during the "checking for openssl support in libcurl" and
// "checking for gnutls support in libcurl" tests in PHP 5.x/7.0's generated
// configure script.
//
// Root cause: PHP 5.x/7.0's configure does:
//
//	save_CFLAGS="$CFLAGS"
//	CFLAGS="`$CURL_CONFIG --cflags`"
//	...openssl/gnutls tests use $CFLAGS in ac_link...
//	CFLAGS="$save_CFLAGS"
//
// The $CURL_CONFIG --cflags for modern curl returns only an -I include path,
// stripping all our -Wno-... flags. The conftest programs use strncasecmp()
// without including <strings.h>, which is a hard error on GCC 14+ where
// implicit function declarations default to error.
func patchPhpConfigureCurlCflags(sourceDir string) error {
	configurePath, err := findFile(sourceDir, "configure")
	if err != nil {
		return err
	}
	data, err := os.ReadFile(configurePath)
	if err != nil {
		return err
	}
	content := string(data)
	old := "CFLAGS=\"`$CURL_CONFIG --cflags`\""
	newStr := "CFLAGS=\"`$CURL_CONFIG --cflags` -Wno-implicit-function-declaration -Wno-error=implicit-function-declaration\""
	if !strings.Contains(content, old) {
		// Already patched or different version; skip silently.
		return nil
	}
	content = strings.Replace(content, old, newStr, 1)
	return os.WriteFile(configurePath, []byte(content), 0o644)
}

// patchIcu58Compat fixes ICU 58.x for modern toolchains:
//   - xlocale.h was removed in glibc 2.26+ (merged into locale.h).
//   - -Wimplicit-fallthrough warnings are errors on GCC 15+.
func patchIcu58Compat(sourceDir string) error {
	// Fix xlocale.h → locale.h in digitlst.cpp
	digitlstPath, err := findFile(sourceDir, "i18n/digitlst.cpp")
	if err == nil {
		data, err := os.ReadFile(digitlstPath)
		if err == nil {
			content := string(data)
			old := "#   include <xlocale.h>"
			newStr := "#   include <locale.h>"
			if strings.Contains(content, old) {
				content = strings.Replace(content, old, newStr, 1)
				os.WriteFile(digitlstPath, []byte(content), 0o644)
			}
		}
	}
	// Add -Wno-implicit-fallthrough to CXXFLAGS in the configure script
	// so ICU's C++ files compile on GCC 15+.
	configurePath, err := findFile(sourceDir, "configure")
	if err == nil {
		data, err := os.ReadFile(configurePath)
		if err == nil {
			content := string(data)
			old := `CXXFLAGS="$CXXFLAGS"`
			newStr := `CXXFLAGS="$CXXFLAGS -Wno-implicit-fallthrough"`
			if strings.Contains(content, old) {
				content = strings.Replace(content, old, newStr, 1)
				os.WriteFile(configurePath, []byte(content), 0o644)
			}
		}
	}
	return nil
}

// patchPhpBison3Compat adds yystrlen/yystpcpy defines to PHP 5.x's
// pre-generated bison parser. Bison 3.x removed these internal symbols
// from the generated code, but PHP 5.x's parser.c was generated with
// bison 2.x and references them.
func patchPhpBison3Compat(sourceDir string) error {
	parserPath, err := findFile(sourceDir, "Zend/zend_language_parser.c")
	if err != nil {
		return err
	}
	data, err := os.ReadFile(parserPath)
	if err != nil {
		return err
	}
	content := string(data)
	old := "#include \"zend.h\""
	newStr := "#include \"zend.h\"\n\n#ifndef yystrlen\n#if defined __STDC_VERSION__ && __STDC_VERSION__ >= 201112L\n#define yystrlen strlen\n#else\nstatic size_t yystrlen(const char *s) { const char *p = s; while (*p) p++; return (size_t)(p - s); }\n#endif\n#endif\n#ifndef yystpcpy\n#if defined __STDC_VERSION__ && __STDC_VERSION__ >= 201112L\n#define yystpcpy stpcpy\n#else\nstatic char *yystpcpy(char *d, const char *s) { while ((*d++ = *s++)); return d - 1; }\n#endif\n#endif"
	if strings.Contains(content, old) {
		content = strings.Replace(content, old, newStr, 1)
		return os.WriteFile(parserPath, []byte(content), 0o644)
	}
	return nil
}
