package domain

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type SystemDepType int

const (
	SystemDepTypeLibrary SystemDepType = iota
	SystemDepTypeBuildTool
)

type SystemDepRequirement struct {
	Name         string
	PkgConfig    string
	MinVersion   string
	VersionRegex string
	VersionCmd   []string
	IsPicky      bool
	IsRequired   bool
}

var systemDepRequirements = map[string]map[string]SystemDepRequirement{
	"8.5": {
		"autoconf": {
			Name: "autoconf", MinVersion: "2.71", VersionRegex: `(\d+\.\d+)`,
			VersionCmd: []string{"autoconf", "--version"}, IsPicky: true, IsRequired: true,
		},
		"automake": {
			Name: "automake", MinVersion: "1.16", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"automake", "--version"}, IsPicky: true, IsRequired: true,
		},
		"libtool": {
			Name: "libtool", MinVersion: "2.4.6", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"libtoolize", "--version"}, IsPicky: true, IsRequired: true,
		},
		"cmake": {
			Name: "cmake", MinVersion: "3.20", VersionRegex: `cmake version (\d+\.\d+\.\d+)`,
			VersionCmd: []string{"cmake", "--version"}, IsPicky: false, IsRequired: false,
		},
		"perl": {
			Name: "perl", MinVersion: "5.0", VersionRegex: `v(\d+\.\d+\.\d+)`,
			VersionCmd: []string{"perl", "-v"}, IsPicky: false, IsRequired: false,
		},
		"m4": {
			Name: "m4", MinVersion: "1.4.18", VersionRegex: `(\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"m4", "--version"}, IsPicky: false, IsRequired: false,
		},
		"re2c": {
			Name: "re2c", MinVersion: "3.0", VersionRegex: `re2c (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"re2c", "--version"}, IsPicky: true, IsRequired: false,
		},
		"flex": {
			Name: "flex", MinVersion: "2.6.4", VersionRegex: `flex (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"flex", "--version"}, IsPicky: false, IsRequired: false,
		},
		"bison": {
			Name: "bison", MinVersion: "3.0", VersionRegex: `bison \(GNU Bison\) (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"bison", "--version"}, IsPicky: false, IsRequired: false,
		},
		"zlib": {
			Name: "zlib", PkgConfig: "zlib", MinVersion: "1.0", IsPicky: false, IsRequired: false,
		},
		"libxml2": {
			Name: "libxml2", PkgConfig: "libxml-2.0", MinVersion: "2.9", IsPicky: false, IsRequired: false,
		},
		"openssl": {
			Name: "openssl", PkgConfig: "openssl", MinVersion: "1.1.1", IsPicky: false, IsRequired: false,
		},
		"curl": {
			Name: "curl", PkgConfig: "libcurl", MinVersion: "7.0", IsPicky: false, IsRequired: false,
		},
		"oniguruma": {
			Name: "oniguruma", PkgConfig: "oniguruma", MinVersion: "6.0", IsPicky: false, IsRequired: false,
		},
	},
	"8.4": {
		"autoconf": {
			Name: "autoconf", MinVersion: "2.71", VersionRegex: `(\d+\.\d+)`,
			VersionCmd: []string{"autoconf", "--version"}, IsPicky: true, IsRequired: true,
		},
		"automake": {
			Name: "automake", MinVersion: "1.16", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"automake", "--version"}, IsPicky: true, IsRequired: true,
		},
		"libtool": {
			Name: "libtool", MinVersion: "2.4.6", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"libtoolize", "--version"}, IsPicky: true, IsRequired: true,
		},
		"cmake": {
			Name: "cmake", MinVersion: "3.20", VersionRegex: `cmake version (\d+\.\d+\.\d+)`,
			VersionCmd: []string{"cmake", "--version"}, IsPicky: false, IsRequired: false,
		},
		"perl": {
			Name: "perl", MinVersion: "5.0", VersionRegex: `v(\d+\.\d+\.\d+)`,
			VersionCmd: []string{"perl", "-v"}, IsPicky: false, IsRequired: false,
		},
		"m4": {
			Name: "m4", MinVersion: "1.4.18", VersionRegex: `(\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"m4", "--version"}, IsPicky: false, IsRequired: false,
		},
		"re2c": {
			Name: "re2c", MinVersion: "3.0", VersionRegex: `re2c (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"re2c", "--version"}, IsPicky: true, IsRequired: false,
		},
		"flex": {
			Name: "flex", MinVersion: "2.6.4", VersionRegex: `flex (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"flex", "--version"}, IsPicky: false, IsRequired: false,
		},
		"bison": {
			Name: "bison", MinVersion: "3.0", VersionRegex: `bison \(GNU Bison\) (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"bison", "--version"}, IsPicky: false, IsRequired: false,
		},
		"zlib": {
			Name: "zlib", PkgConfig: "zlib", MinVersion: "1.0", IsPicky: false, IsRequired: false,
		},
		"libxml2": {
			Name: "libxml2", PkgConfig: "libxml-2.0", MinVersion: "2.9", IsPicky: false, IsRequired: false,
		},
		"openssl": {
			Name: "openssl", PkgConfig: "openssl", MinVersion: "1.1.1", IsPicky: false, IsRequired: false,
		},
		"curl": {
			Name: "curl", PkgConfig: "libcurl", MinVersion: "7.0", IsPicky: false, IsRequired: false,
		},
		"oniguruma": {
			Name: "oniguruma", PkgConfig: "oniguruma", MinVersion: "6.0", IsPicky: false, IsRequired: false,
		},
	},
	"8.3": {
		"autoconf": {
			Name: "autoconf", MinVersion: "2.71", VersionRegex: `(\d+\.\d+)`,
			VersionCmd: []string{"autoconf", "--version"}, IsPicky: true, IsRequired: true,
		},
		"automake": {
			Name: "automake", MinVersion: "1.16", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"automake", "--version"}, IsPicky: true, IsRequired: true,
		},
		"libtool": {
			Name: "libtool", MinVersion: "2.4.6", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"libtoolize", "--version"}, IsPicky: true, IsRequired: true,
		},
		"cmake": {
			Name: "cmake", MinVersion: "3.20", VersionRegex: `cmake version (\d+\.\d+\.\d+)`,
			VersionCmd: []string{"cmake", "--version"}, IsPicky: false, IsRequired: false,
		},
		"perl": {
			Name: "perl", MinVersion: "5.0", VersionRegex: `v(\d+\.\d+\.\d+)`,
			VersionCmd: []string{"perl", "-v"}, IsPicky: false, IsRequired: false,
		},
		"m4": {
			Name: "m4", MinVersion: "1.4.18", VersionRegex: `(\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"m4", "--version"}, IsPicky: false, IsRequired: false,
		},
		"re2c": {
			Name: "re2c", MinVersion: "3.0", VersionRegex: `re2c (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"re2c", "--version"}, IsPicky: true, IsRequired: false,
		},
		"flex": {
			Name: "flex", MinVersion: "2.6.4", VersionRegex: `flex (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"flex", "--version"}, IsPicky: false, IsRequired: false,
		},
		"bison": {
			Name: "bison", MinVersion: "3.0", VersionRegex: `bison \(GNU Bison\) (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"bison", "--version"}, IsPicky: false, IsRequired: false,
		},
		"zlib": {
			Name: "zlib", PkgConfig: "zlib", MinVersion: "1.0", IsPicky: false, IsRequired: false,
		},
		"libxml2": {
			Name: "libxml2", PkgConfig: "libxml-2.0", MinVersion: "2.9", IsPicky: false, IsRequired: false,
		},
		"openssl": {
			Name: "openssl", PkgConfig: "openssl", MinVersion: "1.1.1", IsPicky: false, IsRequired: false,
		},
		"curl": {
			Name: "curl", PkgConfig: "libcurl", MinVersion: "7.0", IsPicky: false, IsRequired: false,
		},
		"oniguruma": {
			Name: "oniguruma", PkgConfig: "oniguruma", MinVersion: "6.0", IsPicky: false, IsRequired: false,
		},
	},
	"7.4": {
		"autoconf": {
			Name: "autoconf", MinVersion: "2.69", VersionRegex: `(\d+\.\d+)`,
			VersionCmd: []string{"autoconf", "--version"}, IsPicky: true, IsRequired: true,
		},
		"automake": {
			Name: "automake", MinVersion: "1.15", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"automake", "--version"}, IsPicky: false, IsRequired: true,
		},
		"libtool": {
			Name: "libtool", MinVersion: "2.4.6", VersionRegex: `(\d+\.\d+(?:\.\d+)?)`,
			VersionCmd: []string{"libtoolize", "--version"}, IsPicky: true, IsRequired: true,
		},
		"cmake": {
			Name: "cmake", MinVersion: "3.10", VersionRegex: `cmake version (\d+\.\d+\.\d+)`,
			VersionCmd: []string{"cmake", "--version"}, IsPicky: false, IsRequired: false,
		},
		"perl": {
			Name: "perl", MinVersion: "5.0", VersionRegex: `v(\d+\.\d+\.\d+)`,
			VersionCmd: []string{"perl", "-v"}, IsPicky: false, IsRequired: false,
		},
		"m4": {
			Name: "m4", MinVersion: "1.4.18", VersionRegex: `(\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"m4", "--version"}, IsPicky: false, IsRequired: false,
		},
		"re2c": {
			Name: "re2c", MinVersion: "1.0", VersionRegex: `re2c (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"re2c", "--version"}, IsPicky: false, IsRequired: false,
		},
		"flex": {
			Name: "flex", MinVersion: "2.5.35", VersionRegex: `flex (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"flex", "--version"}, IsPicky: false, IsRequired: false,
		},
		"bison": {
			Name: "bison", MinVersion: "2.3", VersionRegex: `bison \(GNU Bison\) (\d+\.\d+\.?\d*)`,
			VersionCmd: []string{"bison", "--version"}, IsPicky: false, IsRequired: false,
		},
		"zlib": {
			Name: "zlib", PkgConfig: "zlib", MinVersion: "1.0", IsPicky: false, IsRequired: false,
		},
		"libxml2": {
			Name: "libxml2", PkgConfig: "libxml-2.0", MinVersion: "2.9", IsPicky: false, IsRequired: false,
		},
		"openssl": {
			Name: "openssl", PkgConfig: "openssl", MinVersion: "1.0.2", IsPicky: false, IsRequired: false,
		},
		"curl": {
			Name: "curl", PkgConfig: "libcurl", MinVersion: "7.0", IsPicky: false, IsRequired: false,
		},
		"oniguruma": {
			Name: "oniguruma", PkgConfig: "oniguruma", MinVersion: "6.0", IsPicky: false, IsRequired: false,
		},
	},
}

func GetSystemDepRequirements(phpVersion Version) map[string]SystemDepRequirement {
	versionKey := fmt.Sprintf("%d.%d", phpVersion.Major, phpVersion.Minor)

	if reqs, ok := systemDepRequirements[versionKey]; ok {
		return reqs
	}

	if phpVersion.Major >= 8 && phpVersion.Minor >= 3 {
		return systemDepRequirements["8.3"]
	}

	if phpVersion.Major == 7 {
		return systemDepRequirements["7.4"]
	}

	return systemDepRequirements["7.4"]
}

type SystemDepStatus int

const (
	SystemDepNotFound SystemDepStatus = iota
	SystemDepTooOld
	SystemDepCompatible
)

type SystemDepCheckResult struct {
	Name       string
	Status     SystemDepStatus
	Found      bool
	Version    string
	MinVersion string
	CanUse     bool
}

func CheckSystemDependency(depName string, phpVersion Version) SystemDepCheckResult {
	reqs := GetSystemDepRequirements(phpVersion)
	req, ok := reqs[depName]

	result := SystemDepCheckResult{
		Name:       depName,
		MinVersion: req.MinVersion,
	}

	if !ok {
		result.Status = SystemDepNotFound
		result.CanUse = false
		return result
	}

	if req.PkgConfig != "" {
		cmd := exec.Command("pkg-config", "--exists", req.PkgConfig)
		if err := cmd.Run(); err != nil {
			result.Status = SystemDepNotFound
			result.CanUse = req.IsRequired
			return result
		}

		cmd = exec.Command("pkg-config", "--modversion", req.PkgConfig)
		out, err := cmd.Output()
		if err != nil {
			result.Status = SystemDepNotFound
			result.CanUse = req.IsRequired
			return result
		}

		result.Found = true
		result.Version = strings.TrimSpace(string(out))

		if compareVersions(result.Version, req.MinVersion) >= 0 {
			result.Status = SystemDepCompatible
			result.CanUse = true
		} else {
			result.Status = SystemDepTooOld
			result.CanUse = false
		}
		return result
	}

	if len(req.VersionCmd) == 0 {
		result.Status = SystemDepNotFound
		result.CanUse = false
		return result
	}

	cmd := exec.Command(req.VersionCmd[0], req.VersionCmd[1:]...)
	out, err := cmd.Output()
	if err != nil {
		result.Status = SystemDepNotFound
		result.CanUse = req.IsRequired
		return result
	}

	output := string(out)

	re := regexp.MustCompile(req.VersionRegex)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		result.Status = SystemDepNotFound
		result.CanUse = req.IsRequired
		return result
	}

	result.Found = true
	result.Version = matches[1]

	if compareVersions(result.Version, req.MinVersion) >= 0 {
		result.Status = SystemDepCompatible
		result.CanUse = true
	} else {
		result.Status = SystemDepTooOld
		result.CanUse = false
	}

	return result
}

func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		aNum := 0
		bNum := 0

		if i < len(aParts) {
			aNum, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bNum, _ = strconv.Atoi(bParts[i])
		}

		if aNum > bNum {
			return 1
		}
		if aNum < bNum {
			return -1
		}
	}

	return 0
}

func ShouldUseSystemDeps(phpVersion Version) bool {
	if phpVersion.Major >= 8 && phpVersion.Minor >= 3 {
		return true
	}
	return false
}
