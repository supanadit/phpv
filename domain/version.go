package domain

import (
	"fmt"
	"regexp"
	"strconv"
)

type Version struct {
	Major int
	Minor int
	Patch int
	Extra string
}

func (v Version) String() string {
	if v.Extra != "" {
		return fmt.Sprintf("%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Extra)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func ParseVersion(s string) (Version, error) {
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9]+))?$`)
	matches := re.FindStringSubmatch(s)

	if len(matches) == 0 {
		return Version{}, fmt.Errorf("invalid version format: %s", s)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	extra := matches[4]

	return Version{
		Major: major,
		Minor: minor,
		Patch: patch,
		Extra: extra,
	}, nil
}
