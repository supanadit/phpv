package domain

var DefaultURLPatterns = []URLPattern{
	// PHP - 3 patterns based on version range
	{
		Name:       "php",
		Constraint: func(v *Version) bool { return v.Major == 4 },
		Template:   "https://museum.php.net/php4/php-{version}.tar.gz",
	},
	{
		Name:       "php",
		Constraint: func(v *Version) bool { return v.Major == 5 && v.Minor <= 2 },
		Template:   "https://museum.php.net/php5/php-{version}.tar.gz",
	},
	{
		Name:       "php",
		Constraint: func(v *Version) bool { return v.Major > 5 || (v.Major == 5 && v.Minor > 2) },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
	},

	// zlib
	{
		Name:       "zlib",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://github.com/madler/zlib/releases/download/v{version}/zlib-{version}.tar.gz",
	},

	// re2c
	{
		Name:       "re2c",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://github.com/skvadrik/re2c/releases/download/{version}/re2c-{version}.tar.xz",
	},

	// perl - extension varies by version
	{
		Name:       "perl",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://www.cpan.org/src/5.0/perl-{version}.{ext}",
		ExtensionFunc: func(v *Version) string {
			if v.Raw < "5.20.0" {
				return "tar.bz2"
			}
			return "tar.gz"
		},
	},

	// autoconf
	{
		Name:       "autoconf",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/autoconf/autoconf-{version}.tar.xz",
	},

	// automake
	{
		Name:       "automake",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/automake/automake-{version}.tar.xz",
	},

	// bison
	{
		Name:       "bison",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/bison/bison-{version}.tar.gz",
	},

	// cmake
	{
		Name:       "cmake",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}-linux-x86_64.tar.gz",
	},

	// curl
	{
		Name:       "curl",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://curl.se/download/curl-{version}.tar.gz",
	},

	// flex
	{
		Name:       "flex",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://github.com/westes/flex/releases/download/flex-{version}/flex-{version}.tar.gz",
	},

	// libtool
	{
		Name:       "libtool",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/libtool/libtool-{version}.tar.xz",
	},

	// libxml2
	{
		Name:       "libxml2",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://download.gnome.org/sources/libxml2/{major}.{minor}/libxml2-{version}.tar.xz",
	},

	// m4
	{
		Name:       "m4",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/m4/m4-{version}.tar.xz",
	},

	// oniguruma
	{
		Name:       "oniguruma",
		Constraint: func(v *Version) bool { return true },
		Template:   "https://github.com/kkos/oniguruma/releases/download/v{version}/onig-{version}.tar.gz",
	},

	// openssl - 3 patterns for different version ranges
	{
		Name:       "openssl",
		Constraint: func(v *Version) bool { return v.Major >= 3 },
		Template:   "https://github.com/openssl/openssl/releases/download/openssl-{version}/openssl-{version}.tar.gz",
	},
	{
		Name:       "openssl",
		Constraint: func(v *Version) bool { return v.Major == 1 && v.Minor >= 1 },
		Template:   "https://github.com/openssl/openssl/releases/download/openssl-{version}/openssl-{version}.tar.gz",
	},
	{
		Name:       "openssl",
		Constraint: func(v *Version) bool { return v.Major < 1 || (v.Major == 1 && v.Minor < 1) },
		Template:   "https://www.openssl.org/source/openssl-{version}.tar.gz",
	},
}

func init() {
	RegisterPatterns(DefaultURLPatterns)
}
