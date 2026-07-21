package memory

import (
	"strconv"
	"strings"

	"github.com/supanadit/phpv/domain"
)

type compilerFlagRule struct {
	Compiler  string
	MinPHP    string
	MaxPHP    string
	CFLAGDefs []domain.CompilerFlagDef
}

type cstdRule struct {
	MinPHP string
	MaxPHP string
	CStd   string
	CXXStd string
}

var compilerFlagRuleCandidates = []compilerFlagRule{
	{
		Compiler: "gcc",
		MinPHP:   "5.0",
		MaxPHP:   "7.3",
		CFLAGDefs: []domain.CompilerFlagDef{
			{Flag: "-std=gnu11", Purpose: "C11 standard"},
			{Flag: "-fPIC", Purpose: "position-independent code for shared objects"},
			{Flag: "-DTRUE=1", Purpose: "define TRUE for PHP 7.x intl compat on GCC 15+"},
			{Flag: "-DFALSE=0", Purpose: "define FALSE for PHP 7.x intl compat on GCC 15+"},
			{Flag: "-fpermissive", Needs: ">=gcc15", Purpose: "downgrade pointer-cast errors to warnings"},
			{Flag: "-Wno-cast-function-type", Needs: ">=gcc8", Purpose: "suppress function-type cast warnings"},
			{Flag: "-Wno-error", Purpose: "never treat warnings as errors"},
			{Flag: "-Wno-array-parameter", Needs: ">=gcc11", Purpose: "suppress array-parameter mismatch warnings"},
			{Flag: "-Wno-deprecated-non-prototype", Needs: ">=gcc15", Purpose: "suppress C23 prototype warnings"},
			{Flag: "-Wno-implicit-function-declaration", Needs: ">=gcc14", Purpose: "suppress C99 implicit-decl warnings"},
			{Flag: "-Wno-incompatible-pointer-types", Needs: ">=gcc14", Purpose: "suppress incompatible-pointer warnings"},
		},
	},
	{
		Compiler: "gcc",
		MinPHP:   "7.4",
		MaxPHP:   "7.99",
		CFLAGDefs: []domain.CompilerFlagDef{
			{Flag: "-std=gnu11", Purpose: "C11 standard"},
			{Flag: "-fPIC", Purpose: "position-independent code for shared objects"},
			{Flag: "-fpermissive", Needs: ">=gcc15", Purpose: "downgrade pointer-cast errors to warnings"},
			{Flag: "-Wno-cast-function-type", Needs: ">=gcc8", Purpose: "suppress function-type cast warnings"},
			{Flag: "-Wno-error", Purpose: "never treat warnings as errors"},
			{Flag: "-Wno-array-parameter", Needs: ">=gcc11", Purpose: "suppress array-parameter mismatch warnings"},
			{Flag: "-Wno-deprecated-non-prototype", Needs: ">=gcc15", Purpose: "suppress C23 prototype warnings"},
			{Flag: "-Wno-implicit-function-declaration", Needs: ">=gcc14", Purpose: "suppress C99 implicit-decl warnings"},
			{Flag: "-Wno-incompatible-pointer-types", Needs: ">=gcc14", Purpose: "suppress incompatible-pointer warnings"},
		},
	},
	{
		Compiler: "gcc",
		MinPHP:   "8.0",
		MaxPHP:   "",
		CFLAGDefs: []domain.CompilerFlagDef{
			{Flag: "-fpermissive", Needs: ">=gcc15", Purpose: "downgrade pointer-cast errors to warnings"},
			{Flag: "-Wno-cast-function-type", Needs: ">=gcc8", Purpose: "suppress function-type cast warnings"},
			{Flag: "-Wno-error", Purpose: "never treat warnings as errors"},
			{Flag: "-fPIC", Purpose: "position-independent code for shared objects"},
		},
	},
	{
		Compiler: "zig",
		MinPHP:   "",
		MaxPHP:   "",
		CFLAGDefs: []domain.CompilerFlagDef{
			{Flag: "-std=gnu11", Purpose: "C11 standard"},
			{Flag: "-fPIC", Purpose: "position-independent code for shared objects"},
			{Flag: "-Wno-error", Purpose: "never treat warnings as errors"},
			{Flag: "-fno-sanitize=undefined", Purpose: "disable UB sanitizer (zig-specific)"},
			{Flag: "-Wno-cast-align", Purpose: "suppress cast-align warnings (zig-specific)"},
			{Flag: "-Wno-unused-but-set-variable", Purpose: "suppress unused warnings"},
			{Flag: "-Wno-deprecated-non-prototype", Purpose: "suppress C23 prototype warnings"},
			{Flag: "-Wno-array-parameter", Purpose: "suppress array-parameter mismatch warnings"},
			{Flag: "-Wno-implicit-function-declaration", Purpose: "suppress C99 implicit-decl warnings"},
		},
	},
}

var cstdRules = []cstdRule{
	{MinPHP: "5.0", MaxPHP: "7.4", CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
	{MinPHP: "8.0", MaxPHP: "8.2", CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
	{MinPHP: "8.3", MaxPHP: "", CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
}

func parseVersion(v string) (major, minor int) {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) > 0 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return
}

func versionGE(v string, min string) bool {
	vmaj, vmin := parseVersion(v)
	mmaj, mmin := parseVersion(min)
	if vmaj > mmaj {
		return true
	}
	if vmaj < mmaj {
		return false
	}
	return vmin >= mmin
}

func versionLE(v string, max string) bool {
	vmaj, vmin := parseVersion(v)
	mmaj, mmin := parseVersion(max)
	if vmaj < mmaj {
		return true
	}
	if vmaj > mmaj {
		return false
	}
	return vmin <= mmin
}

func getCompilerFlagCandidates(compiler string, phpVersion string) []domain.CompilerFlagDef {
	for _, rule := range compilerFlagRuleCandidates {
		if rule.Compiler != compiler {
			continue
		}
		minOK := rule.MinPHP == "" || versionGE(phpVersion, rule.MinPHP)
		maxOK := rule.MaxPHP == "" || versionLE(phpVersion, rule.MaxPHP)
		if minOK && maxOK {
			result := make([]domain.CompilerFlagDef, len(rule.CFLAGDefs))
			copy(result, rule.CFLAGDefs)
			return result
		}
	}
	return nil
}

func getCompilerFlags(compiler string, phpVersion string) []string {
	candidates := getCompilerFlagCandidates(compiler, phpVersion)
	result := make([]string, len(candidates))
	for i, c := range candidates {
		result[i] = c.Flag
	}
	return result
}

func getCompilerStdRule(phpVersion string) domain.CompilerRule {
	for _, rule := range cstdRules {
		minOK := rule.MinPHP == "" || versionGE(phpVersion, rule.MinPHP)
		maxOK := rule.MaxPHP == "" || versionLE(phpVersion, rule.MaxPHP)
		if minOK && maxOK {
			return domain.CompilerRule{
				CStd:   rule.CStd,
				CXXStd: rule.CXXStd,
			}
		}
	}
	return domain.CompilerRule{CStd: "-std=gnu11", CXXStd: "-std=gnu++17"}
}

func CXXFlagsFromCFlagsWithStd(cflags []string, isPHPBuild bool, stdRule domain.CompilerRule) []string {
	cxxflags := make([]string, 0, len(cflags))
	hasCXXStd := false

	for _, f := range cflags {
		if f == "-std=gnu11" || f == "-std=c11" {
			if stdRule.CXXStd != "" {
				cxxflags = append(cxxflags, stdRule.CXXStd)
			} else {
				cxxflags = append(cxxflags, "-std=gnu++17")
			}
			hasCXXStd = true
		} else if strings.HasPrefix(f, "-std=c++") || strings.HasPrefix(f, "-std=gnu++") {
			cxxflags = append(cxxflags, f)
			hasCXXStd = true
		} else if cOnlyWarnings[f] {
			continue
		} else {
			cxxflags = append(cxxflags, f)
		}
	}

	if isPHPBuild && !hasCXXStd {
		if stdRule.CXXStd != "" {
			cxxflags = append(cxxflags, stdRule.CXXStd)
		} else {
			cxxflags = append(cxxflags, "-std=gnu++17")
		}
	}

	return cxxflags
}

var cOnlyWarnings = map[string]bool{
	"-Wno-implicit-function-declaration": true,
	"-Wno-incompatible-pointer-types":    true,
	"-Wno-array-parameter":               true,
	"-Wno-deprecated-non-prototype":      true,
}
