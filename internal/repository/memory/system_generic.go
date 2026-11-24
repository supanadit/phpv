package memory

import (
	"bufio"
	"os"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// SystemCatalog is the built-in, in-memory source of distro package data.
type SystemCatalog struct{}

func NewSystemCatalog() *SystemCatalog { return &SystemCatalog{} }

func (c *SystemCatalog) AdvisoryLibraries() []string {
	return append([]string(nil), baseAdvisoryLibraries...)
}

// extension asks for arrive through ExtensionDef.RequiresPackages instead.
var baseAdvisoryLibraries = []string{"sqlite3", "readline", "pcre2", "sodium"}

type pkgMap map[string]string

var fedoraPackages = pkgMap{
	"openssl":   "openssl-devel",
	"libxml2":   "libxml2-devel",
	"zlib":      "zlib-devel",
	"oniguruma": "oniguruma-devel",
	"curl":      "libcurl-devel",
	"sqlite3":   "sqlite-devel",
	"readline":  "readline-devel",
	"icu":       "libicu-devel",
	"pcre2":     "pcre2-devel",
	"argon2":    "libargon2-devel",
	"sodium":    "libsodium-devel",
	"libpng":    "libpng-devel",
	"libjpeg":   "libjpeg-turbo-devel",
	"freetype":  "freetype-devel",
	"libzip":    "libzip-devel",
	"libpq":     "postgresql-devel",
}

var ubuntuPackages = pkgMap{
	"openssl":   "libssl-dev",
	"libxml2":   "libxml2-dev",
	"zlib":      "zlib1g-dev",
	"oniguruma": "libonig-dev",
	"curl":      "libcurl4-openssl-dev",
	"sqlite3":   "libsqlite3-dev",
	"readline":  "libreadline-dev",
	"icu":       "libicu-dev",
	"pcre2":     "libpcre2-dev",
	"argon2":    "libargon2-dev",
	"sodium":    "libsodium-dev",
	"libpng":    "libpng-dev",
	"libjpeg":   "libjpeg-dev",
	"freetype":  "libfreetype-dev",
	"libzip":    "libzip-dev",
	"libpq":     "libpq-dev",
}

var alpinePackages = pkgMap{
	"openssl":   "openssl-dev",
	"libxml2":   "libxml2-dev",
	"zlib":      "zlib-dev",
	"oniguruma": "oniguruma-dev",
	"curl":      "curl-dev",
	"sqlite3":   "sqlite-dev",
	"readline":  "readline-dev",
	"icu":       "icu-dev",
	"pcre2":     "pcre2-dev",
	"argon2":    "argon2-dev",
	"sodium":    "libsodium-dev",
	"libpng":    "libpng-dev",
	"libjpeg":   "libjpeg-dev",
	"freetype":  "freetype-dev",
	"libzip":    "libzip-dev",
	"libpq":     "libpq-dev",
}

var archPackages = pkgMap{
	"openssl":   "openssl",
	"libxml2":   "libxml2",
	"zlib":      "zlib",
	"oniguruma": "oniguruma",
	"curl":      "curl",
	"sqlite3":   "sqlite",
	"readline":  "readline",
	"icu":       "icu",
	"pcre2":     "pcre2",
	"argon2":    "argon2",
	"sodium":    "libsodium",
	"libpng":    "libpng",
	"libjpeg":   "libjpeg-turbo",
	"freetype":  "freetype2",
	"libzip":    "libzip",
	"libpq":     "postgresql-libs",
}

var fedoraBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "gcc-c++",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkgconf-pkg-config",
	"xz":         "xz-devel",
}

var ubuntuBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "g++",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkg-config",
	"xz":         "xz-utils",
}

var alpineBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "g++",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkgconfig",
	"xz":         "xz",
}

var archBuildTools = pkgMap{
	"gcc":        "gcc",
	"g++":        "gcc",
	"make":       "make",
	"cmake":      "cmake",
	"autoconf":   "autoconf",
	"automake":   "automake",
	"m4":         "m4",
	"perl":       "perl",
	"bison":      "bison",
	"re2c":       "re2c",
	"flex":       "flex",
	"pkg-config": "pkg-config",
	"xz":         "xz",
}

func (c *SystemCatalog) PackagesForDistro(distro string) map[string]string {
	switch distro {
	case "fedora", "rhel", "centos":
		return fedoraPackages
	case "ubuntu", "debian":
		return ubuntuPackages
	case "alpine":
		return alpinePackages
	case "arch":
		return archPackages
	default:
		return nil
	}
}

func (c *SystemCatalog) BuildToolsForDistro(distro string) map[string]string {
	switch distro {
	case "fedora", "rhel", "centos":
		return fedoraBuildTools
	case "ubuntu", "debian":
		return ubuntuBuildTools
	case "alpine":
		return alpineBuildTools
	case "arch":
		return archBuildTools
	default:
		return nil
	}
}

func (c *SystemCatalog) Detect() (domain.Distro, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return domain.Distro{Name: "unknown", PM: "unknown"}, nil
	}
	defer f.Close()

	var d domain.Distro
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			d.Name = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			d.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
		}
	}

	switch d.Name {
	case "ubuntu", "debian":
		d.PM = "apt"
	case "fedora", "rhel", "centos":
		d.PM = "dnf"
	case "alpine":
		d.PM = "apk"
	case "arch":
		d.PM = "pacman"
	default:
		d.PM = "unknown"
	}

	return d, nil
}

func (c *SystemCatalog) InstallArgs(pm string, names []string) []string {
	if os.Geteuid() == 0 {
		switch pm {
		case "apt":
			return append([]string{"apt-get", "install", "-y"}, names...)
		case "dnf":
			return append([]string{"dnf", "install", "-y"}, names...)
		case "apk":
			return append([]string{"apk", "add"}, names...)
		case "pacman":
			return append([]string{"pacman", "-S", "--noconfirm"}, names...)
		default:
			return nil
		}
	}
	switch pm {
	case "apt":
		return append([]string{"sudo", "apt-get", "install", "-y"}, names...)
	case "dnf":
		return append([]string{"sudo", "dnf", "install", "-y"}, names...)
	case "apk":
		return append([]string{"sudo", "apk", "add"}, names...)
	case "pacman":
		return append([]string{"sudo", "pacman", "-S", "--noconfirm"}, names...)
	default:
		return nil
	}
}

// baseAdvisoryLibraries are development libraries PHP can use at build time but
// which are not modelled as buildable dependencies: they are expected to come
