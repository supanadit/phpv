package utils

import (
	"regexp"

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
