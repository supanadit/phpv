package repository

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionRange defines a range of versions from 'from' to 'to' (inclusive).
// Supports "x" or "X" as a wildcard in any segment (e.g., "8.x.x" means all 8.x versions).
type VersionRange struct {
	From string
	To   string
}

// MinorRange defines patch bounds for a single minor version.
// PatchStart defaults to 0.
type MinorRange struct {
	Minor      int
	PatchStart int
	PatchEnd   int
}

// BuildRanges concatenates multiple VersionRange slices into a single slice.
// Useful for combining several BuildMinorRanges calls without repeated append.
func BuildRanges(ranges ...[]VersionRange) []VersionRange {
	var result []VersionRange
	for _, r := range ranges {
		result = append(result, r...)
	}
	return result
}

// BuildMinorRanges builds VersionRange slices for a given major version
// and its per-minor patch bounds.
//
// Example:
//
//	BuildMinorRanges(4, []MinorRange{
//	    {Minor: 0, PatchEnd: 6},
//	    {Minor: 1, PatchEnd: 2},
//	})
//	// → [{From: "4.0.0", To: "4.0.6"}, {From: "4.1.0", To: "4.1.2"}]
func BuildMinorRanges(major int, minors []MinorRange) []VersionRange {
	ranges := make([]VersionRange, 0, len(minors))
	for _, m := range minors {
		ranges = append(ranges, VersionRange{
			From: fmt.Sprintf("%d.%d.%d", major, m.Minor, m.PatchStart),
			To:   fmt.Sprintf("%d.%d.%d", major, m.Minor, m.PatchEnd),
		})
	}
	return ranges
}

// RenderTemplate replaces placeholders in a URL template with the given
// version string. Supported placeholders:
//   - {version} — full version string (e.g., "8.2.1")
//   - {major}   — major version number (e.g., "8")
//   - {minor}   — minor version number (e.g., "2")
//   - {patch}   — patch version number (e.g., "1")
func RenderTemplate(tmpl, version string) string {
	s := ParseVersion(version)
	r := strings.NewReplacer(
		"{version}", version,
		"{major}", strconv.Itoa(s.Major),
		"{minor}", strconv.Itoa(s.Minor),
		"{patch}", strconv.Itoa(s.Patch),
	)
	return r.Replace(tmpl)
}

// CompareVersions compares two semver strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareVersions(a, b string) int {
	sa := ParseVersion(a)
	sb := ParseVersion(b)

	if sa.Major != sb.Major {
		if sa.Major < sb.Major {
			return -1
		}
		return 1
	}
	if sa.Minor != sb.Minor {
		if sa.Minor < sb.Minor {
			return -1
		}
		return 1
	}
	if sa.Patch != sb.Patch {
		if sa.Patch < sb.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// GenerateVersions generates all version strings within the given ranges,
// skipping any versions listed in the skip list. Versions not covered by any
// range are considered gaps and are not generated.
func GenerateVersions(ranges []VersionRange, skip []string) []string {
	skipSet := make(map[string]bool, len(skip))
	for _, v := range skip {
		skipSet[v] = true
	}

	var versions []string
	for _, r := range ranges {
		from := ParseVersion(r.From)
		to := ParseVersion(r.To)
		versions = append(versions, expandRange(from, to, skipSet)...)
	}
	return versions
}

// Semver represents a parsed semantic version with major.minor.patch.
type Semver struct {
	Major, Minor, Patch int
}

// ParseVersion parses a version string like "8.2.1", "58.2", or "8.x.x".
// Wildcard "x"/"X" is represented as -1. Two-part versions (e.g. "58.2")
// are treated as major.minor with patch=0.
func ParseVersion(v string) Semver {
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return Semver{}
	}
	var s Semver
	for i, p := range parts {
		if p == "x" || p == "X" {
			switch i {
			case 0:
				s.Major = -1
			case 1:
				s.Minor = -1
			case 2:
				s.Patch = -1
			}
		} else {
			n, _ := strconv.Atoi(p)
			switch i {
			case 0:
				s.Major = n
			case 1:
				s.Minor = n
			case 2:
				s.Patch = n
			}
		}
	}
	return s
}

// expandRange generates all version strings from 'from' to 'to' (inclusive),
// skipping versions present in skipSet. Wildcards (-1) are resolved to
// reasonable bounds: major 0-99, minor 0-99, patch 0-999.
func expandRange(from, to Semver, skipSet map[string]bool) []string {
	// Resolve wildcards to reasonable defaults
	if from.Major == -1 {
		from.Major = 0
	}
	if from.Minor == -1 {
		from.Minor = 0
	}
	if from.Patch == -1 {
		from.Patch = 0
	}
	if to.Major == -1 {
		to.Major = 99
	}
	if to.Minor == -1 {
		to.Minor = 99
	}
	if to.Patch == -1 {
		to.Patch = 999
	}

	var versions []string
	for major := from.Major; major <= to.Major; major++ {
		minMinor := from.Minor
		maxMinor := to.Minor
		if major > from.Major {
			minMinor = 0
		}
		if major < to.Major {
			maxMinor = 99
		}

		for minor := minMinor; minor <= maxMinor; minor++ {
			minPatch := from.Patch
			maxPatch := to.Patch
			if major > from.Major || minor > from.Minor {
				minPatch = 0
			}
			if major < to.Major || minor < to.Minor {
				maxPatch = 999
			}

			for patch := minPatch; patch <= maxPatch; patch++ {
				ver := fmt.Sprintf("%d.%d.%d", major, minor, patch)
				if !skipSet[ver] {
					versions = append(versions, ver)
				}
			}
		}
	}
	return versions
}
