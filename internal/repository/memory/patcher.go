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
				VersionRange: ">=6.0.0, <6.9.10",
				Apply:        patchOnigurumaStForeach,
				ExtraCFlags:  []string{"-Wno-error=incompatible-pointer-types", "-Wno-incompatible-pointer-types"},
			},
		}
	case "php":
		// PHP 7.4's scanf.c uses K&R-style function pointer casts that GCC 15
		// rejects outright. Patch the fn declaration to match the actual
		// strtoll(str, endptr, base) signature. The call site passes 3 args.
		return []patcher.Patch{{
			Name:         "php-gcc15-scanf-fn",
			Package:      "php",
			VersionRange: ">=7.0.0, <8.0.0",
			Apply:        patchPhpScanfFn,
		}}
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

func findFile(root, rel string) (string, error) {
	// Walk source dir looking for the relative path. The extracted source is
	// nested under e.g. sources/php/7.4.33/php-7.4.33/.
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(path, rel) || strings.Contains(path, "/"+rel)) {
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

// patchOnigurumaStForeach fixes the st_foreach macro in st.h so the function
// pointer type matches on GCC 15 (C23). Upstream fix is in 6.9.10+; we
// backport the change to 6.9.x by removing the ANYARGS prototype that
// erases the actual signature.
func patchOnigurumaStForeach(sourceDir string) error {
	// st.h may be in src/ or at the top level depending on the tarball.
	// Walk up to 2 levels deep to find it.
	candidates := []string{
		filepath.Join(sourceDir, "src", "st.h"),
		filepath.Join(sourceDir, "st.h"),
	}
	// Also look one level deeper (e.g., onig-6.9.9/src/st.h).
	entries, _ := os.ReadDir(sourceDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(sourceDir, e.Name())
		candidates = append(candidates,
			filepath.Join(sub, "src", "st.h"),
			filepath.Join(sub, "st.h"),
		)
	}
	var stPath string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			stPath = c
			break
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

	// The broken line uses the ANYARGS macro which expands to an empty
	// parameter list, hiding the actual signature. Replace it with the
	// real signature so callers match.
	// Old: int st_foreach _((st_table *, int (*)(ANYARGS), st_data_t));
	// New: int st_foreach(st_table *, int (*)(st_data_t, st_data_t, st_data_t), st_data_t);
	oldPattern := regexp.MustCompile(`int\s+st_foreach\s+_\(\(st_table\s*\*\s*,\s*int\s*\(\*\)\(ANYARGS\)\s*,\s*st_data_t\)\)\s*;`)
	if !oldPattern.MatchString(content) {
		// Already patched or different version — silently no-op.
		return nil
	}
	content = oldPattern.ReplaceAllString(content,
		"int st_foreach(st_table *, int (*)(st_data_t, st_data_t, st_data_t), st_data_t);")

	if err := os.WriteFile(stPath, []byte(content), 0o644); err != nil {
		return err
	}

	// Also update regparse.c to drop the ARG_UNUSED attribute on st_foreach
	// callbacks, and force -std=gnu17 via CFLAGS to avoid GCC 15 C23
	// behavior. The simplest portable fix is to add a configure-time env.
	regparseCandidates := []string{
		filepath.Join(sourceDir, "src", "regparse.c"),
		filepath.Join(sourceDir, "regparse.c"),
	}
	for _, rc := range regparseCandidates {
		data, err := os.ReadFile(rc)
		if err != nil {
			continue
		}
		// No-op for now: the st.h fix is sufficient on oniguruma 6.9.9.
		_ = data
		_ = strings.TrimSpace
		break
	}

	return nil
}
