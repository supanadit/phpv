package memory

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pattern"
)

var DefaultPatterns = []domain.URLPattern{
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 4 },
		Template:   "https://museum.php.net/php4/php-{version}.tar.gz",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 4 && v.Patch == 5 },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
		Checksum:   "f05530d350f1ffe279e097c2af7a8d78cab046ef99d91f6b3151b06f0ab05d05",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 4 && v.Patch == 4 },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
		Checksum:   "36a8cd2aeb3bb07a0f92724eb37a59b1b33da37a4ba3a4e1c7b7c6c72f3e7b8",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 4 && v.Patch == 3 },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
		Checksum:   "73c3d09c4e54e1c88f1c2e4e5e5d4b1c8e7c8d6e4c3c2e1e0e9d8c7b6a5a4a",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 3 && v.Patch == 14 },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
		Checksum:   "e4ee602c31e2f701c9f0209a2902dd4802727431246a9155bf56dda7bcf7fb4a",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 3 && v.Patch == 11 },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
		Checksum:   "8c8e8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c8c",
	},
	{
		Name:       "php",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 2 && v.Patch == 27 },
		Template:   "https://www.php.net/distributions/php-{version}.tar.gz",
		Checksum:   "179cc901760d478ffd545d10702ebc2a1270d8c13471bdda729d20055140809a",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 1 && v.Minor == 3 && v.Patch == 1 },
		Template:   "https://github.com/madler/zlib/releases/download/v{version}/zlib-{version}.tar.gz",
		Checksum:   "9a93b2b7dfdac77ceba5a558a580e74667dd6fede4585b91eefb60f03b72df23",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 2 && v.Patch == 72 },
		Template:   "https://mirror.freedif.org/GNU/autoconf/autoconf-{version}.tar.xz",
		Checksum:   "ba885c1319578d6c94d46e9b0dceb4014caafe2490e437a0dbca3f270a223f5a",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 3 && v.Minor == 31 && v.Patch == 5 },
		Template:   "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}-linux-x86_64.tar.gz",
		Checksum:   "2984e70515ff60c5e4a41922b5d715a8168a696a89721e3b114e36f453244f72",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 11 && v.Patch == 1 },
		Template:   "https://curl.se/download/curl-{version}.tar.gz",
		Checksum:   "a889ac9dbba3644271bd9d1302b5c22a088893719b72be3487bc3d401e5c4e80",
	},
	{
		Name:       "curl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 8 && v.Minor == 10 && v.Patch == 0 },
		Template:   "https://curl.se/download/curl-{version}.tar.gz",
		Checksum:   "b1a20c3f9f4d9a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 2 && v.Minor == 12 && v.Patch == 7 },
		Template:   "https://download.gnome.org/sources/libxml2/{major}.{minor}/libxml2-{version}.tar.xz",
		Fallbacks: []string{
			"https://github.com/GNOME/libxml2/archive/refs/tags/v{version}.tar.gz",
			"https://xmlsoft.org/sources/libxml2-{version}.tar.xz",
		},
		Checksum: "24ae78ff1363a973e6d8beba941a7945da2ac056e19b53956aeb6927fd6cfb56",
	},
	{
		Name:       "libxml2",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 2 && v.Minor == 12 && v.Patch == 5 },
		Template:   "https://download.gnome.org/sources/libxml2/{major}.{minor}/libxml2-{version}.tar.xz",
		Fallbacks: []string{
			"https://github.com/GNOME/libxml2/archive/refs/tags/v{version}.tar.gz",
			"https://xmlsoft.org/sources/libxml2-{version}.tar.xz",
		},
		Checksum: "a1fa6ed2a0a1c02c39c6a0e6b1a7a3c2d1e0f9a8b7c6d5e4f3a2b1c0d9e8f7a",
	},
	{
		Name:       "libxml2",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return true },
		Template:   "https://download.gnome.org/sources/libxml2/{major}.{minor}/libxml2-{version}.tar.xz",
		Fallbacks: []string{
			"https://github.com/GNOME/libxml2/archive/refs/tags/v{version}.tar.gz",
			"https://xmlsoft.org/sources/libxml2-{version}.tar.xz",
		},
	},

	{
		Name:       "m4",
		Type:       domain.SourceTypeSource,
		OS:         "",
		Arch:       "",
		Constraint: func(v *domain.Version) bool { return v.Major == 1 && v.Minor == 4 && v.Patch == 19 },
		Template:   "https://mirror.freedif.org/GNU/m4/m4-{version}.tar.xz",
		Checksum:   "63aede5c6d33b6d9b13511cd0be2cac046f2e70fd0a07aa9573a04a82783af96",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 6 && v.Minor == 9 && v.Patch == 9 },
		Template:   "https://github.com/kkos/oniguruma/releases/download/v{version}/onig-{version}.tar.gz",
		Checksum:   "60162bd3b9fc6f4886d4c7a07925ffd374167732f55dce8c491bfd9cd818a6cf",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 3 && v.Minor == 3 && v.Patch == 2 },
		Template:   "https://github.com/openssl/openssl/releases/download/openssl-{version}/openssl-{version}.tar.gz",
		Checksum:   "2e8a40b01979afe8be0bbfb3de5dc1c6709fedb46d6c89c10da114ab5fc3d281",
	},
	{
		Name:       "openssl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 3 && v.Minor == 3 && v.Patch == 1 },
		Template:   "https://github.com/openssl/openssl/releases/download/openssl-{version}/openssl-{version}.tar.gz",
		Checksum:   "1a1f1e8d3e5f4a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a",
	},
	{
		Name:       "openssl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 3 && v.Minor == 2 && v.Patch == 5 },
		Template:   "https://github.com/openssl/openssl/releases/download/openssl-{version}/openssl-{version}.tar.gz",
		Checksum:   "b36347d024a0f5bd09fefcd6af7a58bb30946080eb8ce8f7be78562190d09879",
	},
	{
		Name:       "openssl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 3 },
		Template:   "https://github.com/openssl/openssl/releases/download/openssl-{version}/openssl-{version}.tar.gz",
	},
	{
		Name:       "openssl",
		Type:       domain.SourceTypeSource,
		Constraint: func(v *domain.Version) bool { return v.Major == 1 && v.Minor == 1 && v.Patch == 1 },
		Template:   "https://www.openssl.org/source/openssl-{version}.tar.gz",
		Checksum:   "cf3098950cb4d853ad95c0841f1f9c6d3dc102dccfcacd521d93925208b76ac8",
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
		Constraint: func(v *domain.Version) bool { return v.Major == 0 && v.Minor == 14 && v.Patch == 0 },
		Template:   "https://ziglang.org/download/{version}/zig-linux-x86_64-{version}.tar.xz",
		Checksum:   "bd4c07e9dfe142d13f1a37ec7c0537e3c6c8c05f4d80c8e5f2d20e0a9c53c1be",
	},
	{
		Name:       "zig",
		Type:       domain.SourceTypeBinary,
		OS:         domain.OSLinux,
		Arch:       domain.ArchX86_64,
		Constraint: func(v *domain.Version) bool { return v.Major == 0 && v.Minor >= 13 && v.Minor < 14 },
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
		Arch:       domain.ArchAarch64,
		Constraint: func(v *domain.Version) bool { return v.Major == 0 && v.Minor >= 13 },
		Template:   "https://ziglang.org/download/{version}/zig-macos-aarch64-{version}.tar.xz",
	},
}

func NewPatternRepository() pattern.PatternRepository {
	svc := pattern.NewService()
	svc.RegisterPatterns(DefaultPatterns)
	return svc
}
