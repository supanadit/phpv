package platform

import (
	"runtime"

	"github.com/supanadit/phpv/internal/utils"
)

// PlatformService provides unified platform detection and package management
type PlatformService struct {
	osInfo utils.OSInfo
}

// NewPlatformService creates a new instance of PlatformService
func NewPlatformService() *PlatformService {
	return &PlatformService{
		osInfo: utils.DetectOSInfo(),
	}
}

// GetPackageManager returns the detected package manager for the current platform
func (p *PlatformService) GetPackageManager() string {
	return p.osInfo.PkgMgr
}

// GetInstallCommand returns the appropriate install command for the current platform
func (p *PlatformService) GetInstallCommand() string {
	return p.osInfo.InstallCmd
}

// GetPackageName returns the appropriate package name for the given tool on this platform
func (p *PlatformService) GetPackageName(tool string) string {
	// Define package names for different package managers
	packageNames := map[string]map[string]string{
		"apt": {
			"libxml2":   "libxml2-dev",
			"openssl":   "libssl-dev",
			"curl":      "libcurl4-openssl-dev",
			"zlib":      "zlib1g-dev",
			"oniguruma": "libonig-dev",
			"icu":       "libicu-dev",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"dnf": {
			"libxml2":   "libxml2-devel",
			"openssl":   "openssl-devel",
			"curl":      "libcurl-devel",
			"zlib":      "zlib-devel",
			"oniguruma": "oniguruma-devel",
			"icu":       "libicu-devel",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"pacman": {
			"libxml2":   "libxml2",
			"openssl":   "openssl",
			"curl":      "curl",
			"zlib":      "zlib",
			"oniguruma": "oniguruma",
			"icu":       "icu",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"zypper": {
			"libxml2":   "libxml2-devel",
			"openssl":   "libssl-devel",
			"curl":      "libcurl-devel",
			"zlib":      "zlib-devel",
			"oniguruma": "libonig-devel",
			"icu":       "libicu-devel",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"apk": {
			"libxml2":   "libxml2-dev",
			"openssl":   "openssl-dev",
			"curl":      "curl-dev",
			"zlib":      "zlib-dev",
			"oniguruma": "onig-dev",
			"icu":       "icu-dev",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"xbps": {
			"libxml2":   "libxml2-devel",
			"openssl":   "openssl-devel",
			"curl":      "libcurl-devel",
			"zlib":      "zlib-devel",
			"oniguruma": "onig-devel",
			"icu":       "libicu-devel",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
		"brew": {
			"libxml2":   "libxml2",
			"openssl":   "openssl",
			"curl":      "curl",
			"zlib":      "zlib",
			"oniguruma": "oniguruma",
			"icu":       "icu4c",
			"m4":        "m4",
			"autoconf":  "autoconf",
			"automake":  "automake",
			"libtool":   "libtool",
			"perl":      "perl",
			"bison":     "bison",
			"flex":      "flex",
			"re2c":      "re2c",
			"zig":       "zig",
		},
	}

	if pmNames, ok := packageNames[p.osInfo.PkgMgr]; ok {
		if pkg, ok := pmNames[tool]; ok {
			return pkg
		}
	}
	
	// Fallback to the tool name if no package mapping found
	return tool
}

// GetInstallSuggestion returns a formatted install suggestion for the given tool
func (p *PlatformService) GetInstallSuggestion(tool string) string {
	if p.osInfo.PkgMgr == "" {
		return ""
	}
	
	pkgName := p.GetPackageName(tool)
	return p.osInfo.InstallCmd + " " + pkgName
}

// IsWindows returns whether the current platform is Windows
func (p *PlatformService) IsWindows() bool {
	return p.osInfo.GOOS == "windows"
}

// IsMacOS returns whether the current platform is macOS
func (p *PlatformService) IsMacOS() bool {
	return p.osInfo.GOOS == "darwin"
}

// IsLinux returns whether the current platform is Linux
func (p *PlatformService) IsLinux() bool {
	return p.osInfo.GOOS == "linux"
}

// GetArchitecture returns the system architecture
func (p *PlatformService) GetArchitecture() string {
	return runtime.GOARCH
}

// GetOSInfo returns the full OS information
func (p *PlatformService) GetOSInfo() utils.OSInfo {
	return p.osInfo
}