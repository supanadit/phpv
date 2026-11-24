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
