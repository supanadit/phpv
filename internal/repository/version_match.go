package repository

import (
	"regexp"
	"strconv"
	"strings"
)

// MatchVersionRange checks if a version string satisfies a range expression.
// Supports operators: >=, <=, >, <, =, ~, ^ and comma/space-separated AND conditions.
// Examples: ">=8.1.0 <8.2.0", ">=1.0.2,<4.0.0", "~6.9.0", "^8.0"
func MatchVersionRange(rangeStr, version string) bool {
	parts := strings.Split(rangeStr, ",")
	if len(parts) == 1 {
		parts = strings.Split(rangeStr, " ")
	}

	versionToMatch := parseConstraint(version)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		c := parseConstraint(part)
		if !satisfyConstraint(versionToMatch, c) {
			return false
		}
	}

	return true
}

type constraint struct {
	operator string
	major     int
	minor     int
	patch     int
	suffix    string
}

var versionRe = regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?([a-z]*)$`)

func parseConstraint(part string) constraint {
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

	matches := versionRe.FindStringSubmatch(remaining)
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

func compareConstraintParts(v1, v2 constraint) int {
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
		return compareConstraintParts(version, c) == 0
	case ">":
		return compareConstraintParts(version, c) > 0
	case ">=":
		return compareConstraintParts(version, c) >= 0
	case "<":
		return compareConstraintParts(version, c) < 0
	case "<=":
		return compareConstraintParts(version, c) <= 0
	case "~":
		if version.major != c.major {
			return false
		}
		minVer := c
		minVer.patch = 0
		maxVer := c
		maxVer.minor++
		maxVer.patch = 0
		return compareConstraintParts(version, minVer) >= 0 && compareConstraintParts(version, maxVer) < 0
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
		return compareConstraintParts(version, minVer) >= 0 && compareConstraintParts(version, maxVer) < 0
	}

	return false
}

// LatestMatching finds the highest version that starts with the given prefix.
func LatestMatching(versions []string, prefix string) string {
	var best string
	for _, v := range versions {
		if strings.HasPrefix(v, prefix) {
			if best == "" || CompareVersions(v, best) > 0 {
				best = v
			}
		}
	}
	return best
}