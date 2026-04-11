package disk

import (
	"regexp"
	"strings"
)

var buildToolVersionParsers = map[string]func(string) string{
	"m4":       parseM4Version,
	"autoconf": parseAutoconfVersion,
	"automake": parseAutomakeVersion,
	"bison":    parseBisonVersion,
	"flex":     parseFlexVersion,
	"libtool":  parseLibtoolVersion,
	"perl":     parsePerlVersion,
	"re2c":     parseRe2cVersion,
	"zig":      parseZigVersion,
}

func parseM4Version(output string) string {
	re := regexp.MustCompile(`\(GNU M4\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseAutoconfVersion(output string) string {
	re := regexp.MustCompile(`\(GNU Autoconf\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseAutomakeVersion(output string) string {
	re := regexp.MustCompile(`\(GNU Automake\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseBisonVersion(output string) string {
	re := regexp.MustCompile(`\(GNU Bison\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseFlexVersion(output string) string {
	re := regexp.MustCompile(`flex (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parseLibtoolVersion(output string) string {
	re := regexp.MustCompile(`\(GNU libtool\) (\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func parsePerlVersion(output string) string {
	re := regexp.MustCompile(`This is perl 5, version (\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		minor := matches[1]
		return "5." + minor
	}
	re2 := regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)
	matches2 := re2.FindStringSubmatch(output)
	if len(matches2) >= 2 {
		return matches2[1]
	}
	return ""
}

func parseRe2cVersion(output string) string {
	parts := strings.Fields(output)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func parseZigVersion(output string) string {
	parts := strings.Fields(output)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
