package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/supanadit/phpv/domain"
)

var versionRegex = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)([a-z]*)$`)

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

func SortVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		vi := ParseVersion(versions[i])
		vj := ParseVersion(versions[j])
		return CompareVersions(vi, vj) > 0
	})
}

func FilterVersionsByConstraint(versions []string, constraint string) []string {
	parts := splitConstraint(constraint)
	major := 0
	minor := 0
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
		if pv.Minor != minor {
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
	dotCount := 0

	for _, c := range constraint {
		if c == '.' {
			parts = append(parts, current.String())
			current.Reset()
			dotCount++
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
