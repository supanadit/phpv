package graph

import (
	"strings"

	"github.com/supanadit/phpv/domain"
)

// CXXFlagsFromCFlagsWithStd converts a set of C compiler flags into their C++
// equivalents, substituting the C standard flag with the configured C++ standard
// and dropping C-only warnings. Used when PHP's build needs both CFLAGS and
// CXXFLAGS derived from a single source of truth.
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
