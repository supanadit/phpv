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
		from := parseVersion(r.From)
		to := parseVersion(r.To)
		versions = append(versions, expandRange(from, to, skipSet)...)
	}
	return versions
}

// semver represents a parsed semantic version with major.minor.patch.
type semver struct {
	major, minor, patch int
}

// parseVersion parses a version string like "8.2.1" or "8.x.x".
// Wildcard "x"/"X" is represented as -1.
func parseVersion(v string) semver {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return semver{}
	}
	var s semver
	for i, p := range parts {
		if p == "x" || p == "X" {
			switch i {
			case 0:
				s.major = -1
			case 1:
				s.minor = -1
			case 2:
				s.patch = -1
			}
		} else {
			n, _ := strconv.Atoi(p)
			switch i {
			case 0:
				s.major = n
			case 1:
				s.minor = n
			case 2:
				s.patch = n
			}
		}
	}
	return s
}

// expandRange generates all version strings from 'from' to 'to' (inclusive),
// skipping versions present in skipSet. Wildcards (-1) are resolved to
// reasonable bounds: major 0-99, minor 0-99, patch 0-999.
func expandRange(from, to semver, skipSet map[string]bool) []string {
	// Resolve wildcards to reasonable defaults
	if from.major == -1 {
		from.major = 0
	}
	if from.minor == -1 {
		from.minor = 0
	}
	if from.patch == -1 {
		from.patch = 0
	}
	if to.major == -1 {
		to.major = 99
	}
	if to.minor == -1 {
		to.minor = 99
	}
	if to.patch == -1 {
		to.patch = 999
	}

	var versions []string
	for major := from.major; major <= to.major; major++ {
		minMinor := from.minor
		maxMinor := to.minor
		if major > from.major {
			minMinor = 0
		}
		if major < to.major {
			maxMinor = 99
		}

		for minor := minMinor; minor <= maxMinor; minor++ {
			minPatch := from.patch
			maxPatch := to.patch
			if major > from.major || minor > from.minor {
				minPatch = 0
			}
			if major < to.major || minor < to.minor {
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
