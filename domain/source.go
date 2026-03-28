package domain

import (
	"fmt"
	"regexp"
	"strings"
)

type Source struct {
	Name    string
	Version string
	URL     string
}

type Version struct {
	Major  int
	Minor  int
	Patch  int
	Suffix string
	Raw    string
}

type URLPattern struct {
	Name          string
	Constraint    func(v *Version) bool
	Template      string
	ExtensionFunc func(v *Version) string
}

var patternIndex = make(map[string][]URLPattern)

func RegisterPatterns(patterns []URLPattern) {
	for _, p := range patterns {
		patternIndex[p.Name] = append(patternIndex[p.Name], p)
	}
}

func MatchPattern(name string, v *Version) (URLPattern, error) {
	patterns, ok := patternIndex[name]
	if !ok {
		return URLPattern{}, fmt.Errorf("no URL pattern found for %s", name)
	}
	for _, p := range patterns {
		if p.Constraint(v) {
			return p, nil
		}
	}
	return URLPattern{}, fmt.Errorf("no matching URL pattern for %s@%s", name, v.Raw)
}

var versionRegex = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)([a-z]*)$`)

func ParseVersion(version string) *Version {
	matches := versionRegex.FindStringSubmatch(version)
	if matches == nil {
		return &Version{Raw: version}
	}
	var suffix string
	if len(matches) > 4 {
		suffix = matches[4]
	}
	return &Version{
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

func BuildURL(pattern URLPattern, v *Version) (string, error) {
	url := pattern.Template
	url = strings.ReplaceAll(url, "{version}", v.Raw)

	majorMinor := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	url = strings.ReplaceAll(url, "{major}.{minor}", majorMinor)

	if pattern.ExtensionFunc != nil {
		ext := pattern.ExtensionFunc(v)
		url = strings.ReplaceAll(url, "{ext}", ext)
	}

	return url, nil
}
