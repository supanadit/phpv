package version

import (
	"regexp"
	"strconv"
	"strings"
)

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

	versionRegex := regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?([a-z]*)$`)
	matches := versionRegex.FindStringSubmatch(remaining)

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

func satisfyConstraint(version, constraint constraint) bool {
	versionParts := constraint

	switch constraint.operator {
	case "=":
		return compareVersionParts(version, versionParts) == 0
	case ">":
		return compareVersionParts(version, versionParts) > 0
	case ">=":
		return compareVersionParts(version, versionParts) >= 0
	case "<":
		return compareVersionParts(version, versionParts) < 0
	case "<=":
		return compareVersionParts(version, versionParts) <= 0
	case "~":
		if version.major != versionParts.major {
			return false
		}
		minVer := constraint
		minVer.patch = 0
		maxVer := constraint
		maxVer.minor++
		maxVer.patch = 0
		return compareVersionParts(version, minVer) >= 0 && compareVersionParts(version, maxVer) < 0
	case "^":
		if version.major != versionParts.major {
			return false
		}
		minVer := constraint
		minVer.minor = 0
		minVer.patch = 0
		maxVer := constraint
		maxVer.major++
		maxVer.minor = 0
		maxVer.patch = 0
		return compareVersionParts(version, minVer) >= 0 && compareVersionParts(version, maxVer) < 0
	}

	return false
}

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

type ParsedVersion struct {
	Major  int
	Minor  int
	Patch  int
	Suffix string
}

func Parse(s string) ParsedVersion {
	v := ParsedVersion{}
	parts := strings.Split(s, ".")

	if len(parts) > 0 {
		v.Major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		v.Minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		patchAndSuffix := strings.Split(parts[2], "-")
		v.Patch, _ = strconv.Atoi(patchAndSuffix[0])
		if len(patchAndSuffix) > 1 {
			v.Suffix = patchAndSuffix[1]
		}
	}

	return v
}

func FilterVersionsByConstraint(versions []string, constraint string) []string {
	var filtered []string
	for _, v := range versions {
		if MatchVersionRange(constraint, v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
