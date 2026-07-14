package memory

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository"
)

type RegistryRepository struct{}

func NewRegistryRepository() *RegistryRepository {
	return &RegistryRepository{}
}

func (reg *RegistryRepository) List(name string, checksum bool, os string) (result []domain.Registry, err error) {
	switch name {
	case "php":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "php",
			Type: "source_code",
			OS:   "all",
			Ranges: repository.BuildRanges(
				repository.BuildMinorRanges(8, []repository.MinorRange{
					{Minor: 0, PatchEnd: 30},
					{Minor: 1, PatchEnd: 33},
					{Minor: 2, PatchEnd: 29},
					{Minor: 3, PatchEnd: 27},
					{Minor: 4, PatchEnd: 19},
					{Minor: 5, PatchEnd: 8},
				}),
				repository.BuildMinorRanges(7, []repository.MinorRange{
					{Minor: 0, PatchEnd: 33},
					{Minor: 1, PatchEnd: 33},
					{Minor: 2, PatchEnd: 34},
					{Minor: 3, PatchEnd: 33},
					{Minor: 4, PatchEnd: 33},
				}),
				repository.BuildMinorRanges(5, []repository.MinorRange{
					{Minor: 0, PatchEnd: 5},
					{Minor: 1, PatchEnd: 6},
					{Minor: 2, PatchEnd: 17},
					{Minor: 3, PatchEnd: 29},
					{Minor: 4, PatchEnd: 45},
					{Minor: 5, PatchEnd: 38},
					{Minor: 6, PatchEnd: 40},
				}),
				repository.BuildMinorRanges(4, []repository.MinorRange{
					{Minor: 0, PatchEnd: 6},
					{Minor: 1, PatchEnd: 2},
					{Minor: 2, PatchEnd: 3},
					{Minor: 3, PatchEnd: 11},
					{Minor: 4, PatchEnd: 9},
				}),
			),
			URLTemplate: "https://www.php.net/distributions/php-{version}.tar.gz",
			Checksums: []repository.Checksum{
				{
					Version: "8.5.8",
					Type:    "sha256",
					Value:   "6ebc55e52af4396385e689f7af0f28944fbbf966843433b573e9dc1dc03df539",
				},
			},
		})

	case "cmake":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "cmake",
			Type: "binary",
			OS:   "linux",
			Ranges: repository.BuildRanges(
				repository.BuildMinorRanges(3, []repository.MinorRange{
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

	case "perl":
		result = repository.BuildRegistries(repository.PackageConfig{
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
			Extension: repository.ExtensionConfig{
				Default: "tar.gz",
				Override: []repository.ExtOverride{
					{Before: "5.20.0", Ext: "tar.bz2"},
				},
			},
		})

	case "openssl":
		result = repository.BuildRegistries(repository.PackageConfig{
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
			Checksums: []repository.Checksum{
				{Version: "1.1.1w", Type: "sha256", Value: "cf3098950cb4d853ad95c0841f1f9c6d3dc102dccfcacd521d93925208b76ac8"},
			},
		})

	case "zlib":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "zlib",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.3.1",
				"1.2.13", "1.2.12", "1.2.11", "1.2.9", "1.2.8", "1.2.7", "1.2.6",
				"1.2.5", "1.2.4", "1.2.3", "1.2.2", "1.2.1",
			},
			URLTemplate: "https://github.com/madler/zlib/releases/download/v{version}/zlib-{version}.tar.gz",
			Checksums: []repository.Checksum{
				{Version: "1.3.1", Type: "sha256", Value: "9a93b2b7dfdac77ceba5a558a580e74667dd6fede4585b91eefb60f03b72df23"},
			},
		})

	case "libxml2":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "libxml2",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.12.7", "2.12.6", "2.12.5", "2.11.7", "2.11.6", "2.11.5",
				"2.9.14", "2.9.13", "2.9.12", "2.9.11", "2.9.10", "2.9.9",
				"2.6.30",
			},
			URLTemplate: "https://download.gnome.org/sources/libxml2/{major}.{minor}/libxml2-{version}.tar.xz",
			Checksums: []repository.Checksum{
				{Version: "2.12.7", Type: "sha256", Value: "24ae78ff1363a973e6d8beba941a7945da2ac056e19b53956aeb6927fd6cfb56"},
			},
		})

	case "oniguruma":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "oniguruma",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"6.9.9", "6.9.8", "6.9.7", "6.9.6", "6.9.5",
				"5.9.6",
			},
			URLTemplate: "https://github.com/kkos/oniguruma/releases/download/v{version}/onig-{version}.tar.gz",
			Checksums: []repository.Checksum{
				{Version: "6.9.9", Type: "sha256", Value: "60162bd3b9fc6f4886d4c7a07925ffd374167732f55dce8c491bfd9cd818a6cf"},
			},
		})

	case "curl":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "curl",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"8.10.1", "8.10.0", "8.9.1", "8.8.0", "8.7.1", "8.6.0", "8.5.0",
				"8.4.0", "8.3.0", "8.2.1", "8.1.2", "8.0.1",
				"7.88.1", "7.88.0", "7.87.0", "7.86.0", "7.85.0", "7.84.0",
				"7.83.0", "7.82.0", "7.81.0", "7.80.0",
				"7.20.0",
				"7.12.1", "7.12.0",
			},
			URLTemplate: "https://curl.se/download/curl-{version}.tar.gz",
		})

	case "m4":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "m4",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.4.19", "1.4.18", "1.4.17", "1.4.16",
			},
			URLTemplate: "https://mirror.freedif.org/GNU/m4/m4-{version}.tar.xz",
			Checksums: []repository.Checksum{
				{Version: "1.4.19", Type: "sha256", Value: "63aede5c6d33b6d9b13511cd0be2cac046f2e70fd0a07aa9573a04a82783af96"},
			},
		})

	case "autoconf":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "autoconf",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.72", "2.71", "2.70", "2.69", "2.68", "2.67", "2.65",
				"2.63", "2.62", "2.61", "2.60", "2.59", "2.13",
			},
			URLTemplate: "https://mirror.freedif.org/GNU/autoconf/autoconf-{version}.tar.xz",
			Checksums: []repository.Checksum{
				{Version: "2.72", Type: "sha256", Value: "ba885c1319578d6c94d46e9b0dceb4014caafe2490e437a0dbca3f270a223f5a"},
			},
		})

	case "automake":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "automake",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.17", "1.16.5", "1.16.4", "1.16.3", "1.16.2", "1.16.1", "1.16",
				"1.15.1", "1.15",
				"1.14.1", "1.13.4",
				"1.11.6", "1.10.3", "1.9.6", "1.8.5",
				"1.4-p6",
			},
			URLTemplate: "https://mirror.freedif.org/GNU/automake/automake-{version}.tar.xz",
		})

	case "libtool":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "libtool",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.5.4", "2.5.3", "2.5.2", "2.5.1", "2.5.0",
				"2.4.7", "2.4.6", "2.4.5", "2.4.4", "2.4.2",
				"2.2.6b", "2.2.4",
				"1.5.26", "1.5.24", "1.5.22", "1.5.20",
				"1.4.3", "1.3.5",
			},
			URLTemplate: "https://mirror.freedif.org/GNU/libtool/libtool-{version}.tar.xz",
		})

	case "flex":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "flex",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.6.4", "2.6.3", "2.6.2", "2.6.1", "2.6.0",
				"2.5.39", "2.5.37", "2.5.35", "2.5.34", "2.5.33",
			},
			URLTemplate: "https://github.com/westes/flex/releases/download/v{version}/flex-{version}.tar.gz",
		})

	case "bison":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "bison",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"3.8.2", "3.8.1", "3.7.6", "3.7.5", "3.7.4", "3.7.3",
				"3.6.4", "3.5.3", "3.4.2", "3.3.2",
				"3.0.5", "3.0.4",
				"2.7.1", "2.5.37", "2.4.3", "2.3", "2.1",
				"1.35", "1.28",
			},
			URLTemplate: "https://mirror.freedif.org/GNU/bison/bison-{version}.tar.gz",
		})

	case "re2c":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "re2c",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"3.1", "3.0", "2.2", "2.1", "2.0.3", "2.0",
				"1.3", "1.2.1", "1.1.1", "1.0.3", "1.0.1",
				"0.16", "0.14.3",
			},
			URLTemplate: "https://github.com/skvadrik/re2c/releases/download/{version}/re2c-{version}.tar.xz",
		})

	case "icu":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "icu",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"75.1", "74.2", "74.1", "73.2", "73.1",
				"72.1", "71.1", "70.1",
				"69.1", "68.2", "67.1", "66.1",
				"58.2", "57.2", "57.1",
			},
			URLTemplate: "https://github.com/unicode-org/icu/releases/download/release-{major}-{minor}/icu4c-{major}_{minor}-src.tgz",
		})

	case "zig":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "zig",
			Type: "binary",
			OS:   "linux",
			Versions: []string{
				"0.14.0", "0.13.0", "0.12.0", "0.11.0",
			},
			URLTemplate: "https://ziglang.org/download/{version}/zig-linux-x86_64-{version}.tar.xz",
			Checksums: []repository.Checksum{
				{Version: "0.14.0", Type: "sha256", Value: "bd4c07e9dfe142d13f1a37ec7c0537e3c6c8c05f4d80c8e5f2d20e0a9c53c1be"},
			},
		})

	default:
		return nil, fmt.Errorf("unknown package: %s", name)
	}

	// Filter by OS when a specific OS is requested.
	// Entries with OS="all" are always included.
	if os != "" && os != "all" {
		filtered := make([]domain.Registry, 0, len(result))
		for _, r := range result {
			if r.OS == "all" || r.OS == os {
				filtered = append(filtered, r)
			}
		}
		result = filtered
	}

	if checksum {
		filtered := make([]domain.Registry, 0, len(result))
		for _, r := range result {
			if r.ChecksumType != "" {
				filtered = append(filtered, r)
			}
		}
		result = filtered
	}

	return result, err
}

// Get implements [registry.RegistryRepository].
func (reg *RegistryRepository) Get(name string, version string, checksum bool, os string) (r domain.Registry, err error) {
	registries, err := reg.List(name, checksum, os)
	if err != nil {
		return r, err
	}
	for _, registry := range registries {
		if registry.Version == version {
			return registry, nil
		}
	}
	return r, fmt.Errorf("registry %s version %s not found", name, version)
}
