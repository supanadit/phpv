package memory

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/extension"
	"github.com/supanadit/phpv/flagresolver"
	"github.com/supanadit/phpv/internal/utils"
)

// NewFlagRepository creates a new flag repository with the given extension repository.
func NewFlagRepository(extRepo extension.Repository) flagresolver.Repository {
	return &flagRepo{extRepo: extRepo}
}

type flagRepo struct {
	extRepo extension.Repository
}

// packageFlagRule defines configure flags for a specific package version range.
// This allows different flags to be used based on the package version being built.
//
// Example usage:
//
//	// Simple flags with no version constraint
//	{"m4", "", "", []string{"--disable-maintainer-mode"}, nil}
//
//	// Flags only for a specific version range
//	{"openssl", "1.1.0", "2.9.9", []string{"shared"}, nil}
//
//	// Flags with version-specific additions
//	{"icu", "77.0", "", []string{"--disable-extras"}, map[string][]string{
//		"78.0": {"--disable-samples"},
//	}}
type packageFlagRule struct {
	// Name is the package name (e.g., "openssl", "curl", "icu").
	Name string

	// MinVer is the minimum version (inclusive) for this rule to apply.
	// Empty string means no minimum (matches all versions >= 0.0.0).
	MinVer string

	// MaxVer is the maximum version (inclusive) for this rule to apply.
	// Empty string means no maximum (matches all versions <= infinity).
	MaxVer string

	// Flags are the base configure flags applied when version matches.
	Flags []string

	// FlagsMin contains additional flags keyed by minimum version.
	// Any flag with a min version <= the target version will be included.
	// Use this for incremental flag additions across versions.
	FlagsMin map[string][]string
}

// packageFlags defines default configure flags for each supported package.
// Rules are evaluated in order; first matching rule wins.
// Version comparison uses major.minor (patch is ignored).
//
// Supported packages:
//   - m4: GNU m4 macro processor
//   - openssl: OpenSSL cryptography library (adds "no-legacy" for >= 3.0)
//   - curl: cURL HTTP client library
//   - libxml2: XML parsing library
//   - zlib: compression library
//   - oniguruma: regex library (used by PHP's onig extension)
//   - icu: International Components for Unicode (used by PHP's intl extension)
//   - re2c: lexer generator
//   - autoconf, automake, libtool: build tools
//   - flex, bison: parser generators
//   - perl: required for building OpenSSL
//   - cmake: build system
//
// Note: PHP itself is NOT listed here; use GetPHPConfigureFlags() for PHP.
var packageFlags = []packageFlagRule{
	{"m4", "", "", []string{"--disable-maintainer-mode"}, nil},
	{"openssl", "", "", []string{"shared", "no-ssl3", "no-tests"}, nil},
	{"curl", "", "", []string{"--with-ssl", "--without-brotli", "--disable-ldap", "--without-libpsl", "--without-libidn2", "--without-zstd", "--without-nghttp2", "--without-zlib"}, nil},
	{"libxml2", "", "", []string{"--disable-shared", "--enable-static", "--without-lzma", "--without-python", "--disable-dependency-tracking", "--with-zlib"}, nil},
	{"zlib", "", "", nil, nil},
	{"oniguruma", "", "", nil, nil},
	{"icu", "", "", []string{"--disable-extras", "--disable-samples"}, nil},
	{"re2c", "", "", nil, nil},
	{"autoconf", "", "", nil, nil},
	{"automake", "", "", nil, nil},
	{"libtool", "", "", nil, nil},
	{"flex", "", "", nil, nil},
	{"bison", "", "", nil, nil},
	{"perl", "", "", nil, nil},
	{"cmake", "", "", nil, nil},
}

// cstdRules defines C/C++ standard flags for different PHP version ranges.
// The order matters - first matching rule wins.
//
// These rules handle compiler compatibility issues, particularly with GCC 15+
// which requires C++17 for ICU library headers (std::enable_if_t, std::u16string_view, etc.).
//
// Rule structure:
//   - MinPHP: Minimum PHP version (inclusive), empty means no minimum
//   - MaxPHP: Maximum PHP version (inclusive), empty means no maximum
//   - CStd: C standard flag (e.g., "-std=gnu11")
//   - CXXStd: C++ standard flag (e.g., "-std=gnu++17")
//
// Version comparison uses major.minor (patch is ignored).
//
// Usage:
//   rule := s.GetCompilerStdRule("8.0.30")
//   // Returns CStd: "-std=gnu11", CXXStd: "-std=gnu++17" for PHP 8.0
//
// Extending for new PHP versions:
//   Add a new rule before the catch-all, e.g:
//   {MinPHP: "8.4", MaxPHP: "", CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
// CompilerFlagRule defines C compiler flags (CFLAGS) for a specific compiler and PHP version range.
// This allows different compiler flags to be used based on the compiler type and PHP version.
//
// Rules are evaluated in order; first matching rule wins.
//
// Example:
//
//	CompilerFlagRule{Compiler: "gcc", MinPHP: "5.0", MaxPHP: "7.99", CFLAGS: []string{"-std=gnu11", "-fPIC", ...}}
//	CompilerFlagRule{Compiler: "gcc", MinPHP: "8.0", MaxPHP: "", CFLAGS: []string{"-Wno-error", "-fPIC"}}
//	CompilerFlagRule{Compiler: "zig", MinPHP: "", MaxPHP: "", CFLAGS: []string{"-std=gnu11", "-fPIC", ...}}
type CompilerFlagRule struct {
	// Compiler is the compiler type ("gcc" or "zig").
	Compiler string

	// MinPHP is the minimum PHP version (inclusive) for this rule.
	// Empty string means no minimum.
	MinPHP string

	// MaxPHP is the maximum PHP version (inclusive) for this rule.
	// Empty string means no maximum.
	MaxPHP string

	// CFLAGS are the C compiler flags for this compiler/version combination.
	CFLAGS []string
}

// compilerFlagRules defines C compiler flags for different compiler and PHP version ranges.
// Rules are evaluated in order; first matching rule wins.
//
// GCC rules:
//   - PHP 5.x-7.x: Includes -std=gnu11, -fPIC, and GCC 15+ compatibility flags.
//     -fno-strict-function-pointer-casts: old PHP code casts function pointers
//       (e.g., scanf.c casts strtoul to a no-args function pointer).
//     -fpermissive: GCC 15+ hard-errors when calling a function through a cast
//       pointer with wrong arg count; -fpermissive downgrades it to a warning.
//   - PHP 8.0+: GCC 15+ still hits scanf.c function pointer cast issues so
//     -fpermissive and -Wno-cast-function-type are needed here too.
//
// Zig rules:
//   - All PHP versions: Includes -std=gnu11, -fPIC, and suppression flags for
//     warnings that Zig's C compiler emits that are not relevant for PHP builds.
var compilerFlagRules = []CompilerFlagRule{
	{
		Compiler: "gcc",
		MinPHP:   "5.0",
		MaxPHP:   "7.99",
		CFLAGS:   []string{"-std=gnu11", "-fPIC", "-fno-strict-function-pointer-casts", "-fpermissive", "-Wno-cast-function-type", "-Wno-error", "-Wno-array-parameter", "-Wno-deprecated-non-prototype", "-Wno-implicit-function-declaration", "-Wno-incompatible-pointer-types"},
	},
	{
		Compiler: "gcc",
		MinPHP:   "8.0",
		MaxPHP:   "",
		CFLAGS:   []string{"-fpermissive", "-Wno-cast-function-type", "-Wno-error", "-fPIC"},
	},
	{
		Compiler: "zig",
		MinPHP:   "",
		MaxPHP:   "",
		CFLAGS:   []string{"-std=gnu11", "-fPIC", "-Wno-error", "-fno-sanitize=undefined", "-Wno-cast-align", "-Wno-unused-but-set-variable", "-Wno-deprecated-non-prototype", "-Wno-array-parameter", "-Wno-implicit-function-declaration"},
	},
}

var cstdRules = []flagresolver.CStdRule{
	// PHP 5.x - 7.x with GCC 15+ needs C++17 for ICU headers
	{MinPHP: "5.0", MaxPHP: "7.4", CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
	// PHP 8.0 - 8.2 with GCC 15+ needs C++17 for ICU headers
	{MinPHP: "8.0", MaxPHP: "8.2", CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
	// PHP 8.3+ likely already has proper C++ support
	{MinPHP: "8.3", MaxPHP: "", CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
}

// GetCompilerStdRule returns the C/C++ standard flag rule for the given PHP version.
// It matches against cstdRules in order, returning the first rule where the PHP version
// falls within the rule's MinPHP/MaxPHP range.
//
// Usage:
//
//	stdRule := r.GetCompilerStdRule("8.0.30")
//	fmt.Println(stdRule.CStd)    // "-std=gnu11"
//	fmt.Println(stdRule.CXXStd) // "-std=gnu++17"
//
// Returns a default rule with CStd="-std=gnu11", CXXStd="-std=gnu++17" if no rule matches.
func (r *flagRepo) GetCompilerStdRule(phpVersion string) flagresolver.CStdRule {
	v := utils.ParseVersion(phpVersion)
	for _, rule := range cstdRules {
		minOK := rule.MinPHP == "" || versionGE(v, rule.MinPHP)
		maxOK := rule.MaxPHP == "" || versionLE(v, rule.MaxPHP)
		if minOK && maxOK {
			return rule
		}
	}
	// Default fallback
	return flagresolver.CStdRule{CStd: "-std=gnu11", CXXStd: "-std=gnu++17"}
}

// GetCompilerFlags returns C compiler flags (CFLAGS) for a specific compiler and PHP version.
// It matches against compilerFlagRules in order, returning the first rule where the
// compiler type and PHP version fall within the rule's ranges.
//
// The compiler parameter should be "gcc" or "zig".
//
// Usage:
//
//	cflags := r.GetCompilerFlags("gcc", "7.4.33")
//	// Returns: ["-std=gnu11", "-fPIC", "-fno-strict-function-pointer-casts", ...]
//
//	cflags := r.GetCompilerFlags("gcc", "8.2.0")
//	// Returns: ["-Wno-error", "-fPIC"]
//
//	cflags := r.GetCompilerFlags("zig", "8.0.30")
//	// Returns: ["-std=gnu11", "-fPIC", "-Wno-error", ...]
//
// Returns an empty slice if no matching rule is found.
func (r *flagRepo) GetCompilerFlags(compiler string, phpVersion string) []string {
	v := utils.ParseVersion(phpVersion)

	for _, rule := range compilerFlagRules {
		if rule.Compiler != compiler {
			continue
		}

		minOK := rule.MinPHP == "" || versionGE(v, rule.MinPHP)
		maxOK := rule.MaxPHP == "" || versionLE(v, rule.MaxPHP)
		if minOK && maxOK {
			result := make([]string, len(rule.CFLAGS))
			copy(result, rule.CFLAGS)
			return result
		}
	}

	return []string{}
}

// versionGE checks if version v >= minVersion using major.minor comparison.
// Patch versions are ignored for comparison.
//
// Examples:
//
//	v := ParseVersion("8.0.30")
//	v.GE("8.0")   // true
//	v.GE("8.1")   // false
//	v.GE("7.4")   // true
//	v.GE("8.2")   // false
func versionGE(v *domain.Version, minVersion string) bool {
	mv := utils.ParseVersion(minVersion)
	if v.Major > mv.Major {
		return true
	}
	if v.Major < mv.Major {
		return false
	}
	return v.Minor >= mv.Minor
}

// versionLE checks if version v <= maxVersion using major.minor comparison.
// Patch versions are ignored for comparison.
//
// Examples:
//
//	v := ParseVersion("8.0.30")
//	v.LE("8.2")   // true
//	v.LE("8.0")   // true
//	v.LE("7.9")   // false
//	v.LE("8.3")   // true
func versionLE(v *domain.Version, maxVersion string) bool {
	mv := utils.ParseVersion(maxVersion)
	if v.Major < mv.Major {
		return true
	}
	if v.Major > mv.Major {
		return false
	}
	return v.Minor <= mv.Minor
}

// GetConfigureFlags returns configure flags for a specific package version.
// It matches against packageFlags in order, returning the first rule where the
// package name and version fall within the rule's MinVer/MaxVer range.
//
// The function also handles:
//   - Version-specific flags via FlagsMin map
//   - Special case for OpenSSL >= 3.0 (adds "no-legacy")
//
// Usage:
//
//	flags := r.GetConfigureFlags("openssl", "3.2.0")
//	// Returns: ["shared", "no-ssl3", "no-tests", "no-legacy"]
//
//	flags := r.GetConfigureFlags("icu", "77.0")
//	// Returns: ["--disable-extras", "--disable-samples"]
//
//	flags := r.GetConfigureFlags("m4", "1.4.19")
//	// Returns: ["--disable-maintainer-mode"]
//
// Returns an empty slice if no matching rule is found.
func (r *flagRepo) GetConfigureFlags(name string, version string) []string {
	v := utils.ParseVersion(version)

	for _, rule := range packageFlags {
		if rule.Name != name {
			continue
		}

		// Check version range
		minOK := rule.MinVer == "" || versionGE(v, rule.MinVer)
		maxOK := rule.MaxVer == "" || versionLE(v, rule.MaxVer)
		if !minOK || !maxOK {
			continue
		}

		// Collect flags
		var result []string
		result = append(result, rule.Flags...)

		// Check version-specific flags
		if rule.FlagsMin != nil {
			for minVer, flags := range rule.FlagsMin {
				if versionGE(v, minVer) {
					result = append(result, flags...)
				}
			}
		}

		// Special case: openssl >= 3.0 adds "no-legacy"
		if name == "openssl" && v.Major >= 3 {
			result = append(result, "no-legacy")
		}

		return result
	}

	return []string{}
}

func (r *flagRepo) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	flags := []string{
		"--disable-all",
		"--enable-cli",
	}

	if len(extensions) == 0 {
		return flags
	}

	extensions = r.expandImplied(extensions)
	v := utils.ParseVersion(phpVersion)

	for _, ext := range extensions {
		if extDef, ok := r.extRepo.GetExtensionDef(ext); ok {
			if r.extRepo.IsExtensionValidForPHPVersion(ext, phpVersion) {
				flags = append(flags, extDef.Flag)
			}
		}
	}

	if contains(extensions, "opcache") && v.Major >= 7 {
		flags = append(flags, "--enable-opcache")
	}

	return flags
}

func (r *flagRepo) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return r.extRepo.GetExtensionDef(name)
}

func (r *flagRepo) IsExtensionValidForPHPVersion(name string, phpVersion string) bool {
	return r.extRepo.IsExtensionValidForPHPVersion(name, phpVersion)
}

func (r *flagRepo) GetConflictingExtensions(name string) []string {
	return r.extRepo.GetConflictingExtensions(name)
}

func (r *flagRepo) GetExtensionDependency(name string) (string, bool) {
	return r.extRepo.GetExtensionDependency(name)
}

func (r *flagRepo) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	return r.extRepo.GetExtensionDependencyWithVersion(extName, phpVersion)
}

func (r *flagRepo) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	return r.extRepo.ValidateExtensions(extensions, phpVersion)
}

func (r *flagRepo) expandImplied(extensions []string) []string {
	visited := make(map[string]bool)
	var result []string

	var add func(name string)
	add = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		result = append(result, name)
		if extDef, ok := r.extRepo.GetExtensionDef(name); ok {
			for _, implied := range extDef.Implied {
				add(implied)
			}
		}
	}

	for _, ext := range extensions {
		add(ext)
	}
	return result
}

func (r *flagRepo) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	return r.extRepo.CheckExtensionConflicts(extensions)
}
