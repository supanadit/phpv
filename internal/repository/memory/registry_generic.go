package memory

import (
	"runtime"

	"github.com/supanadit/phpv/domain"
)

// listSecondary resolves the less common packages kept in a separate driver file.
func (reg *RegistryRepository) listSecondary(name string) ([]domain.Registry, error) {
	var result []domain.Registry
	switch name {
	case "autoconf":
		result = BuildRegistries(PackageConfig{
			Name: "autoconf",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.72", "2.71", "2.70", "2.69", "2.68", "2.67", "2.65",
				"2.63", "2.62", "2.61", "2.60", "2.59", "2.13",
			},
			URLTemplate: "https://mirror.freedif.org/GNU/autoconf/autoconf-{version}.tar.xz",
			Checksums: []Checksum{
				{Version: "2.72", Type: "sha256", Value: "ba885c1319578d6c94d46e9b0dceb4014caafe2490e437a0dbca3f270a223f5a"},
			},
		})

	case "automake":
		result = BuildRegistries(PackageConfig{
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
		result = BuildRegistries(PackageConfig{
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
		result = BuildRegistries(PackageConfig{
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
		result = BuildRegistries(PackageConfig{
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
		result = BuildRegistries(PackageConfig{
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

	case "pcre2":
		result = BuildRegistries(PackageConfig{
			Name: "pcre2",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"10.44", "10.43", "10.42",
			},
			URLTemplate: "https://github.com/PCRE2Project/pcre2/releases/download/pcre2-{version}/pcre2-{version}.tar.gz",
		})

	case "expat":
		result = BuildRegistries(PackageConfig{
			Name: "expat",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.6.3", "2.6.2",
			},
			URLTemplate: "https://github.com/libexpat/libexpat/releases/download/R_{major}_{minor}_{patch}/expat-{version}.tar.gz",
		})

	case "icu":
		result = BuildRegistries(PackageConfig{
			Name: "icu",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"75.1", "74.2", "74.1", "73.2", "73.1",
				"72.1", "71.1", "70.1",
				"69.1", "68.2", "67.1", "66.1",
				"65.1", "64.2", "63.2", "63.1",
				"58.2", "57.2", "57.1",
			},
			URLTemplate: "https://github.com/unicode-org/icu/releases/download/release-{major}-{minor}/icu4c-{major}_{minor}-src.tgz",
		})

	case "zig":
		result = BuildRegistries(PackageConfig{
			Name: "zig",
			Type: "binary",
			OS:   "linux",
			Versions: []string{
				"0.14.0", "0.13.0", "0.12.0", "0.11.0",
			},
			URLTemplate: "https://ziglang.org/download/{version}/zig-linux-x86_64-{version}.tar.xz",
			Checksums: []Checksum{
				{Version: "0.14.0", Type: "sha256", Value: "bd4c07e9dfe142d13f1a37ec7c0537e3c6c8c05f4d80c8e5f2d20e0a9c53c1be"},
			},
		})

	case "caddy":
		// Caddy publishes a prebuilt static binary per OS/arch. The
		// {os}/{arch} placeholders render to runtime.GOOS/GOARCH at
		// registry build time (OS is set to runtime.GOOS so non-matching
		// platforms simply see no entries).
		result = BuildRegistries(PackageConfig{
			Name: "caddy",
			Type: "binary",
			OS:   runtime.GOOS,
			Versions: []string{
				"2.10.2", "2.10.1", "2.10.0",
				"2.8.4", "2.8.3",
			},
			URLTemplate: "https://github.com/caddyserver/caddy/releases/download/v{version}/caddy_{version}_{os}_{arch}.tar.gz",
			Checksums: []Checksum{
				{Version: "2.10.2", Type: "sha256", Value: "5c218bc34c9197369263da7e9317a83acdbd80ef45d94dca5eff76e727c67cdd"},
			},
		})

	case "nginx":
		result = BuildRegistries(PackageConfig{
			Name: "nginx",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.28.0", "1.27.4",
				"1.26.3", "1.26.2",
			},
			URLTemplate: "https://nginx.org/download/nginx-{version}.tar.gz",
		})

	case "httpd":
		result = BuildRegistries(PackageConfig{
			Name: "httpd",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"2.4.62", "2.4.61",
			},
			URLTemplate: "https://archive.apache.org/dist/httpd/httpd-{version}.tar.gz",
		})

	case "apr":
		result = BuildRegistries(PackageConfig{
			Name: "apr",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.7.4", "1.7.2",
			},
			URLTemplate: "https://archive.apache.org/dist/apr/apr-{version}.tar.gz",
		})

	case "apr-util":
		result = BuildRegistries(PackageConfig{
			Name: "apr-util",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"1.6.3",
			},
			URLTemplate: "https://archive.apache.org/dist/apr/apr-util-{version}.tar.gz",
		})

	default:
	}
	return result, nil
}
