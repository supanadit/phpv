package pattern

import "github.com/supanadit/phpv/domain"

var DefaultURLPatterns = []domain.URLPattern{
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 4 },
		Template:   "https://museum.php.net/php4/php-{version}.tar.gz",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 5 && v.Minor <= 2 },
		Template:   "https://museum.php.net/php5/php-{version}.tar.gz",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major > 5 || (v.Major == 5 && v.Minor > 2) },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
	},

	{
		Name:       "zlib",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/madler/zlib/releases/download/v{version}/zlib-{version}.tar.gz",
	},

	{
		Name:       "re2c",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/skvadrik/re2c/releases/download/{version}/re2c-{version}.tar.xz",
	},

	{
		Name:       "perl",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://www.cpan.org/src/5.0/perl-{version}.{ext}",
		ExtensionFunc: func(v *domain.Version) string {
			if v.Raw < "5.20.0" {
				return "tar.bz2"
			}
			return "tar.gz"
		},
	},

	{
		Name:       "autoconf",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/autoconf/autoconf-{version}.tar.xz",
	},

	{
		Name:       "automake",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/automake/automake-{version}.tar.xz",
	},

	{
		Name:       "bison",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/bison/bison-{version}.tar.gz",
	},

	{
		Name:       "cmake",
		Type:       domain.SourceTypeBinary,
		OS:         domain.OSLinux,
		Arch:       domain.ArchX86_64,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}-linux-x86_64.tar.gz",
	},
	{
		Name:       "cmake",
		Type:       domain.SourceTypeBinary,
		OS:         domain.OSDarwin,
		Arch:       domain.ArchX86_64,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}-macos-universal.tar.gz",
	},
	{
		Name:       "cmake",
		Type:       domain.SourceTypeBinary,
		OS:         domain.OSDarwin,
		Arch:       domain.ArchArm64,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}-macos-arm64.tar.gz",
	},
	{
		Name:       "cmake",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}.tar.gz",
	},

	{
		Name:       "curl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://curl.se/download/curl-{version}.tar.gz",
	},

	{
		Name:       "flex",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/westes/flex/releases/download/flex-{version}/flex-{version}.tar.gz",
	},

	{
		Name:       "libtool",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/libtool/libtool-{version}.tar.xz",
	},

	{
		Name:       "libxml2",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://download.gnome.org/sources/libxml2/{major}.{minor}/libxml2-{version}.tar.xz",
		Fallbacks: []string{
			"https://xmlsoft.org/sources/libxml2-{version}.tar.xz",
			"https://ftp.linux.org.tw/pub/libxml/libxml2-{version}.tar.xz",
		},
	},

	{
		Name:       "m4",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://mirror.freedif.org/GNU/m4/m4-{version}.tar.xz",
	},

	{
		Name:       "oniguruma",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://github.com/kkos/oniguruma/releases/download/v{version}/onig-{version}.tar.gz",
	},

	{
		Name:       "openssl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major >= 3 },
		Template:   "https://github.com/openssl/openssl/releases/download/openssl-{version}/openssl-{version}.tar.gz",
	},
	{
		Name:       "openssl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 1 && v.Minor >= 1 },
		Template:   "https://www.openssl.org/source/openssl-{version}.tar.gz",
	},
	{
		Name:       "openssl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major < 1 || (v.Major == 1 && v.Minor < 1) },
		Template:   "https://www.openssl.org/source/openssl-{version}.tar.gz",
	},

	{
		Name:       "zig",
		Type:       domain.SourceTypeBinary,
		OS:         domain.OSLinux,
		Arch:       domain.ArchX86_64,
		Constraint: func(v *domain.Version) bool { return v.Major == 0 && v.Minor >= 13 },
		Template:   "https://ziglang.org/download/{version}/zig-linux-x86_64-{version}.tar.xz",
	},
	{
		Name:       "zig",
		Type:       domain.SourceTypeBinary,
		OS:         domain.OSDarwin,
		Arch:       domain.ArchX86_64,
		Constraint: func(v *domain.Version) bool { return v.Major == 0 && v.Minor >= 13 },
		Template:   "https://ziglang.org/download/{version}/zig-macos-x86_64-{version}.tar.xz",
	},
	{
		Name:       "zig",
		Type:       domain.SourceTypeBinary,
		OS:         domain.OSDarwin,
		Arch:       domain.ArchArm64,
		Constraint: func(v *domain.Version) bool { return v.Major == 0 && v.Minor >= 13 },
		Template:   "https://ziglang.org/download/{version}/zig-macos-aarch64-{version}.tar.xz",
	},
}
