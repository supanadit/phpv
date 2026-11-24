package memory

import (
	"github.com/supanadit/phpv/domain"
	vsn "github.com/supanadit/phpv/pkg/version"
)

// listPrimary resolves the common packages kept in a separate driver file.
func (reg *RegistryRepository) listPrimary(name string) ([]domain.Registry, error) {
	var result []domain.Registry
	switch name {
	case "php":
		result = BuildRegistries(PackageConfig{
			Name: "php",
			Type: "source_code",
			OS:   "all",
			Ranges: vsn.BuildRanges(
				vsn.BuildMinorRanges(8, []vsn.MinorRange{
					{Minor: 0, PatchEnd: 30},
					{Minor: 1, PatchEnd: 33},
					{Minor: 2, PatchEnd: 36},
					{Minor: 3, PatchEnd: 37},
					{Minor: 4, PatchEnd: 28},
					{Minor: 5, PatchEnd: 13},
					{Minor: 6, PatchEnd: 0},
				}),
				vsn.BuildMinorRanges(7, []vsn.MinorRange{
					{Minor: 0, PatchEnd: 33},
					{Minor: 1, PatchEnd: 33},
					{Minor: 2, PatchEnd: 34},
					{Minor: 3, PatchEnd: 33},
					{Minor: 4, PatchEnd: 33},
				}),
				vsn.BuildMinorRanges(5, []vsn.MinorRange{
					{Minor: 0, PatchEnd: 5},
					{Minor: 1, PatchEnd: 6},
					{Minor: 2, PatchEnd: 17},
					{Minor: 3, PatchEnd: 29},
					{Minor: 4, PatchEnd: 45},
					{Minor: 5, PatchEnd: 38},
					{Minor: 6, PatchEnd: 40},
				}),
				vsn.BuildMinorRanges(4, []vsn.MinorRange{
					{Minor: 0, PatchEnd: 6},
					{Minor: 1, PatchEnd: 2},
					{Minor: 2, PatchEnd: 3},
					{Minor: 3, PatchEnd: 11},
					{Minor: 4, PatchEnd: 9},
				}),
			),
			URLTemplate: "https://www.php.net/distributions/php-{version}.tar.gz",
			URLOverrides: []URLOverride{
				{Before: "5.0.0", URL: "https://museum.php.net/php4/php-{version}.tar.gz"},
				{Before: "5.3.0", URL: "https://museum.php.net/php5/php-{version}.tar.gz"},
			},
			Checksums: []Checksum{
				{
					Version: "8.6.0",
					Type:    "sha256",
					Value:   "62337dccde3d9fcc39797d4529f42637613036eaf3875353c206874e5813a3c1",
				},
				{
					Version: "8.5.13",
					Type:    "sha256",
					Value:   "3aeb20a7e667e197929747f76047409196f91f0172357439319cba344df8b3e1",
				},
				{
					Version: "8.4.28",
					Type:    "sha256",
					Value:   "deffc88c60fcfb99f683a9f6689e9c91cf41b7cba50bd5f01a232a98c27f2cd4",
				},
				{
					Version: "8.3.37",
					Type:    "sha256",
					Value:   "cdd2ed867cb104ececff7c0292142c807c68a6b6da1b114ead4d586fba628a4f",
				},
				{
					Version: "8.2.36",
					Type:    "sha256",
					Value:   "c871dbdc1cd3d898e48632c8b5fcc73f1b0e1cbb15ef816579ced7f2ae1696a8",
				},
				{
					Version: "8.5.12",
					Type:    "sha256",
					Value:   "81a0f450349e84b08eb43faa85f8a4c835684f3cc1c50471775d1317a046c525",
				},
				{
					Version: "8.4.27",
					Type:    "sha256",
					Value:   "5bafed9994c1a49796b25b5f61c22c9183bfe046b6acc7b363842d03c77f972f",
				},
				{
					Version: "8.3.36",
					Type:    "sha256",
					Value:   "c93f3441f87f6ab156bef4432db215d7e61ca2c0183bb3a2aa28d9886ab59f53",
				},
				{
					Version: "8.2.35",
					Type:    "sha256",
					Value:   "9db4204f0a2d1911277ae832185e07bb4c3b3559772753a70d2968ca346d6910",
				},
				{
					Version: "8.5.10",
					Type:    "sha256",
					Value:   "f5c0ac99b85b3d677de475c2e4f509f9b4f54663f3ee5a84d6d9481a521d4100",
				},
				{
					Version: "8.4.25",
					Type:    "sha256",
					Value:   "5cde1da976b728fa09e6b549a4c8afcd0c180337028dacf34f73b11be50253db",
				},
				{
					Version: "8.5.8",
					Type:    "sha256",
					Value:   "6ebc55e52af4396385e689f7af0f28944fbbf966843433b573e9dc1dc03df539",
				},
			},
		})

	case "cmake":
		result = BuildRegistries(PackageConfig{
			Name: "cmake",
			Type: "binary",
			OS:   "linux",
			Ranges: vsn.BuildRanges(
				vsn.BuildMinorRanges(3, []vsn.MinorRange{
					{Minor: 21, PatchEnd: 4},
					{Minor: 22, PatchEnd: 1},
					{Minor: 23, PatchEnd: 3},
					{Minor: 24, PatchEnd: 2},
					{Minor: 25, PatchEnd: 3},
					{Minor: 26, PatchEnd: 5},
					{Minor: 27, PatchEnd: 6},
				}),
			),
			URLTemplate: "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}-linux-x86_64.tar.gz",
		})

	case "libzip":
		result = BuildRegistries(PackageConfig{
			Name: "libzip",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.11.3", "1.10.1", "1.9.2", "1.7.3",
			},
			URLTemplate: "https://libzip.org/download/libzip-{version}.tar.gz",
		})

	case "perl":
		result = BuildRegistries(PackageConfig{
			Name: "perl",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"5.42.1", "5.40.3", "5.38.5", "5.38.2", "5.36.3", "5.34.3",
				"5.32.1", "5.30.3", "5.28.3", "5.26.3", "5.24.4",
				"5.22.3", "5.20.0", "5.18.4", "5.16.3", "5.14.4",
				"5.12.5", "5.10.1", "5.8.9", "5.6.2", "5.5.30",
				"5.4.50",
			},
			URLTemplate: "https://www.cpan.org/src/5.0/perl-{version}.{ext}",
			Extension: ExtensionConfig{
				Default: "tar.gz",
				Override: []ExtOverride{
					{Before: "5.20.0", Ext: "tar.bz2"},
				},
			},
		})

	case "openssl":
		result = BuildRegistries(PackageConfig{
			Name: "openssl",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"3.3.2", "3.3.1", "3.3.0",
				"3.2.5", "3.2.4", "3.2.3", "3.2.2", "3.2.1", "3.2.0",
				"3.1.7", "3.1.6", "3.1.5", "3.1.4", "3.1.3", "3.1.2", "3.1.1", "3.1.0",
				"3.0.15", "3.0.14", "3.0.13", "3.0.12", "3.0.11", "3.0.10", "3.0.9",
				"1.1.1w", "1.1.1v", "1.1.1u", "1.1.1t", "1.1.1s", "1.1.1n", "1.1.1m", "1.1.1l",
				"1.1.1k", "1.1.1j", "1.1.1i", "1.1.1h", "1.1.1g", "1.1.1f", "1.1.1e", "1.1.1d",
				"1.1.1c", "1.1.1b", "1.1.1a", "1.1.1",
				"1.0.2u", "1.0.2t", "1.0.2s", "1.0.2r", "1.0.2n", "1.0.2k", "1.0.2j", "1.0.2g",
				"1.0.1u", "1.0.1t", "1.0.1p", "1.0.1e",
				"0.9.8zh", "0.9.8zc",
			},
			URLTemplate: "https://www.openssl.org/source/openssl-{version}.tar.gz",
			Checksums: []Checksum{
				{Version: "1.1.1w", Type: "sha256", Value: "cf3098950cb4d853ad95c0841f1f9c6d3dc102dccfcacd521d93925208b76ac8"},
			},
		})

	case "zlib":
		result = BuildRegistries(PackageConfig{
			Name: "zlib",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.3.1",
				"1.2.13", "1.2.12", "1.2.11", "1.2.9", "1.2.8", "1.2.7", "1.2.6",
				"1.2.5", "1.2.4", "1.2.3", "1.2.2", "1.2.1",
			},
			URLTemplate: "https://github.com/madler/zlib/releases/download/v{version}/zlib-{version}.tar.gz",
			Checksums: []Checksum{
				{Version: "1.3.1", Type: "sha256", Value: "9a93b2b7dfdac77ceba5a558a580e74667dd6fede4585b91eefb60f03b72df23"},
			},
		})

	case "libxml2":
		result = BuildRegistries(PackageConfig{
			Name: "libxml2",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.12.7", "2.12.6", "2.12.5", "2.11.7", "2.11.6", "2.11.5",
				"2.9.14", "2.9.13", "2.9.12", "2.9.11", "2.9.10", "2.9.9",
				"2.6.30",
			},
			URLTemplate: "https://download.gnome.org/sources/libxml2/{major}.{minor}/libxml2-{version}.tar.xz",
			Checksums: []Checksum{
				{Version: "2.12.7", Type: "sha256", Value: "24ae78ff1363a973e6d8beba941a7945da2ac056e19b53956aeb6927fd6cfb56"},
			},
		})

	case "oniguruma":
		result = BuildRegistries(PackageConfig{
			Name: "oniguruma",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"6.9.9", "6.9.8", "6.9.7", "6.9.6", "6.9.5",
				"5.9.6",
			},
			URLTemplate: "https://github.com/kkos/oniguruma/releases/download/v{version}/onig-{version}.tar.gz",
			Checksums: []Checksum{
				{Version: "6.9.9", Type: "sha256", Value: "60162bd3b9fc6f4886d4c7a07925ffd374167732f55dce8c491bfd9cd818a6cf"},
			},
		})

	case "curl":
		result = BuildRegistries(PackageConfig{
			Name: "curl",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"8.10.1", "8.10.0", "8.9.1", "8.8.0", "8.7.1", "8.6.0", "8.5.0",
				"8.4.0", "8.3.0", "8.2.1", "8.1.2", "8.0.1",
				"7.88.1", "7.88.0", "7.87.0", "7.86.0", "7.85.0", "7.84.0",
				"7.83.0", "7.82.0", "7.81.0", "7.80.0",
				"7.20.0",
				"7.12.0",
			},
			URLTemplate: "https://curl.se/download/curl-{version}.tar.gz",
			// curl.se keeps releases <= 7.29.0 only under /download/archeology/;
			// the flat /download/ path returns 404 for them (verified 7.30.0 is
			// the oldest version still served from /download/).
			URLOverrides: []URLOverride{
				{Before: "7.30.0", URL: "https://curl.se/download/archeology/curl-{version}.tar.gz"},
			},
			Checksums: []Checksum{
				{Version: "7.20.0", Type: "sha256", Value: "e12d06b551c5c9d3420b0eda20eb2861a6daab032c4d6d8fab07b56f34c8a848"},
				{Version: "7.12.0", Type: "sha256", Value: "43dcb0419453e48d6b4f3883e80a55de5da5811c0a519133f10bcc59d9412e07"},
			},
		})

	case "m4":
		result = BuildRegistries(PackageConfig{
			Name: "m4",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.4.19", "1.4.18", "1.4.17", "1.4.16",
			},
			URLTemplate: "https://mirror.freedif.org/GNU/m4/m4-{version}.tar.xz",
			Checksums: []Checksum{
				{Version: "1.4.19", Type: "sha256", Value: "63aede5c6d33b6d9b13511cd0be2cac046f2e70fd0a07aa9573a04a82783af96"},
			},
		})

	}
	return result, nil
}
