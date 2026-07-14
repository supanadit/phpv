package system

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
	"pkg-config": "pkgconfig",
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

func packagesForDistro(distro string) pkgMap {
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

func buildToolsForDistro(distro string) pkgMap {
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
