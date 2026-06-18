package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/supanadit/phpv/domain"
)

var versionRegex = regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?([a-z]*)$`)

// ParseVersion takes a version string (e.g., "8.4.3") and returns a *domain.Version.
func ParseVersion(version string) *domain.Version {
	matches := versionRegex.FindStringSubmatch(version)
	if matches == nil {
		return &domain.Version{Raw: version}
	}
	var suffix string
	if len(matches) > 4 {
		suffix = matches[4]
	}
	return &domain.Version{
		Major:  parseInt(matches[1]),
		Minor:  parseInt(matches[2]),
		Patch:  parseInt(matches[3]),
		Suffix: suffix,
		Raw:    version,
	}
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// CompareVersions compares two domain.Version values.
func CompareVersions(a, b *domain.Version) int {
	if a.Major != b.Major {
		if a.Major > b.Major {
			return 1
		}
		return -1
	}
	if a.Minor != b.Minor {
		if a.Minor > b.Minor {
			return 1
		}
		return -1
	}
	if a.Patch != b.Patch {
		if a.Patch > b.Patch {
			return 1
		}
		return -1
	}
	if a.Suffix != b.Suffix {
		if a.Suffix > b.Suffix {
			return 1
		}
		return -1
	}
	return 0
}

// SortVersions sorts version strings in descending order (newest first).
func SortVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		vi := ParseVersion(versions[i])
		vj := ParseVersion(versions[j])
		return CompareVersions(vi, vj) > 0
	})
}

// FilterVersionsByConstraint filters version strings by major.minor.patch prefix constraint.
func FilterVersionsByConstraint(versions []string, constraint string) []string {
	parts := splitConstraint(constraint)
	major := 0
	minor := -1
	patch := -1

	if len(parts) >= 1 {
		fmt.Sscanf(parts[0], "%d", &major)
	}
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &minor)
	}
	if len(parts) >= 3 {
		fmt.Sscanf(parts[2], "%d", &patch)
	}

	var matched []string
	for _, v := range versions {
		pv := ParseVersion(v)
		if pv.Major != major {
			continue
		}
		if minor >= 0 && pv.Minor != minor {
			continue
		}
		if patch >= 0 && pv.Patch != patch {
			continue
		}
		matched = append(matched, v)
	}

	SortVersions(matched)
	return matched
}

func splitConstraint(constraint string) []string {
	var parts []string
	var current strings.Builder

	for _, c := range constraint {
		if c == '.' {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		if c < '0' || c > '9' {
			continue
		}
		current.WriteRune(c)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// ResolveVersionConstraint returns the best matching version for a constraint.
func ResolveVersionConstraint(availableVersions []string, constraint string) (string, error) {
	matched := FilterVersionsByConstraint(availableVersions, constraint)
	if len(matched) == 0 {
		return "", fmt.Errorf("no version found matching %q", constraint)
	}
	return matched[0], nil
}

// ResolveInstalledVersion returns the best matching installed version.
func ResolveInstalledVersion(installedVersions []string, constraint string) (string, error) {
	matched := FilterVersionsByConstraint(installedVersions, constraint)
	if len(matched) == 0 {
		return "", fmt.Errorf("no installed version matching %q", constraint)
	}
	return matched[0], nil
}

// --- Range-based version matching (merged from constraint.go) ---

type constraint struct {
	operator string
	major    int
	minor    int
	patch    int
	suffix   string
}

func parseConstraintPart(part string) constraint {
	part = strings.TrimSpace(part)

	operators := []string{">=", "<=", ">", "<", "=", "~", "^"}

	var operator string
	var remaining string

	for _, op := range operators {
		if strings.HasPrefix(part, op) {
			operator = op
			remaining = strings.TrimPrefix(part, op)
			break
		}
	}

	if operator == "" {
		operator = "="
		remaining = part
	}

	c := constraint{operator: operator}

	verRe := regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?([a-z]*)$`)
	matches := verRe.FindStringSubmatch(remaining)

	if matches != nil {
		c.major, _ = strconv.Atoi(matches[1])
		if matches[2] != "" {
			c.minor, _ = strconv.Atoi(matches[2])
		}
		if matches[3] != "" {
			c.patch, _ = strconv.Atoi(matches[3])
		}
		c.suffix = matches[4]
	}

	return c
}

func compareVersionParts(v1, v2 constraint) int {
	if v1.major != v2.major {
		if v1.major > v2.major {
			return 1
		}
		return -1
	}

	if v1.minor != v2.minor {
		if v1.minor > v2.minor {
			return 1
		}
		return -1
	}

	if v1.patch != v2.patch {
		if v1.patch > v2.patch {
			return 1
		}
		return -1
	}

	if v1.suffix != v2.suffix {
		if v1.suffix > v2.suffix {
			return 1
		}
		return -1
	}

	return 0
}

func satisfyConstraint(version, c constraint) bool {
	switch c.operator {
	case "=":
		return compareVersionParts(version, c) == 0
	case ">":
		return compareVersionParts(version, c) > 0
	case ">=":
		return compareVersionParts(version, c) >= 0
	case "<":
		return compareVersionParts(version, c) < 0
	case "<=":
		return compareVersionParts(version, c) <= 0
	case "~":
		if version.major != c.major {
			return false
		}
		minVer := c
		minVer.patch = 0
		maxVer := c
		maxVer.minor++
		maxVer.patch = 0
		return compareVersionParts(version, minVer) >= 0 && compareVersionParts(version, maxVer) < 0
	case "^":
		if version.major != c.major {
			return false
		}
		minVer := c
		minVer.minor = 0
		minVer.patch = 0
		maxVer := c
		maxVer.major++
		maxVer.minor = 0
		maxVer.patch = 0
		return compareVersionParts(version, minVer) >= 0 && compareVersionParts(version, maxVer) < 0
	}

	return false
}

// MatchVersionRange checks if a version string satisfies a range expression.
// Supports operators: >=, <=, >, <, =, ~, ^ and comma/space-separated AND conditions.
func MatchVersionRange(rangeStr string, version string) bool {
	parts := strings.Split(rangeStr, ",")
	if len(parts) == 1 {
		parts = strings.Split(rangeStr, " ")
	}

	versionToMatch := parseConstraintPart(version)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		c := parseConstraintPart(part)

		if !satisfyConstraint(versionToMatch, c) {
			return false
		}
	}

	return true
}
