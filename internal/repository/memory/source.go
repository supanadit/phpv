package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type SourceRepository struct {
	// In Memory doesn't receive any external input
}

func NewSourceRepository() *SourceRepository {
	return &SourceRepository{}
}

func (r *SourceRepository) GetVersions() ([]domain.Source, error) {
	// TODO: Add RCx, beta, alpha versions
	versions := r.generateRangeVersions(8, 5, 0, 4)
	versions = append(
		versions,
		r.generateRangeVersions(8, 4, 0, 19)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 3, 0, 27)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 2, 0, 29)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 1, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 0, 0, 30)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 4, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 3, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 2, 0, 34)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 1, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 0, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 6, 0, 40)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 5, 0, 38)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 4, 0, 45)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 3, 0, 29)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 2, 0, 17)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 1, 0, 6)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 0, 0, 5)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 4, 0, 9)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 3, 0, 11)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 2, 0, 3)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 1, 0, 2)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 0, 0, 6)...,
	)
	// ZLib
	versions = append(
		versions,
		[]domain.Source{
			{Name: "zlib", Version: "1.3.1", URL: "https://github.com/madler/zlib/releases/download/v1.3.1/zlib-1.3.1.tar.gz"},
			{Name: "zlib", Version: "1.2.13", URL: "https://github.com/madler/zlib/releases/download/v1.2.13/zlib-1.2.13.tar.gz"},
		}...,
	)
	// Re2C
	versions = append(
		versions,
		[]domain.Source{
			{Name: "re2c", Version: "3.1", URL: "https://github.com/skvadrik/re2c/releases/download/3.1/re2c-3.1.tar.xz"},
			{Name: "re2c", Version: "2.2", URL: "https://github.com/skvadrik/re2c/releases/download/2.2/re2c-2.2.tar.xz"},
			{Name: "re2c", Version: "1.3", URL: "https://github.com/skvadrik/re2c/releases/download/1.3/re2c-1.3.tar.xz"},
			{Name: "re2c", Version: "0.16", URL: "https://github.com/skvadrik/re2c/releases/download/0.16/re2c-0.16.tar.gz"},
			{Name: "re2c", Version: "0.14", URL: "https://github.com/skvadrik/re2c/releases/download/0.14/re2c-0.14.tar.gz"},
		}...,
	)
	// Perl Source
	versions = append(
		versions,
		[]domain.Source{
			r.perlSource("5.42.1"),
			r.perlSource("5.40.3"),
			r.perlSource("5.38.5"),
			r.perlSource("5.36.3"),
			r.perlSource("5.34.3"),
			r.perlSource("5.32.1"),
			r.perlSource("5.30.3"),
			r.perlSource("5.28.3"),
			r.perlSource("5.26.3"),
			r.perlSource("5.24.4"),
			r.perlSource("5.22.3"),
			r.perlSource("5.20.0"),
			r.perlSource("5.18.4"),
			r.perlSource("5.16.3"),
			r.perlSource("5.14.4"),
			r.perlSource("5.12.5"),
			r.perlSource("5.10.1"),
			r.perlSource("5.8.9"),
			r.perlSource("5.6.2"),
			{Name: "perl", Version: "5.5.30", URL: "https://www.cpan.org/src/5.0/perl5.005_03.tar.gz"},
			{Name: "perl", Version: "5.4.50", URL: "https://www.cpan.org/src/5.0/perl5.004_05.tar.gz"},
		}...,
	)
	// Autoconf
	versions = append(
		versions,
		[]domain.Source{
			{Name: "autoconf", Version: "2.72", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.72.tar.xz"},
			{Name: "autoconf", Version: "2.71", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.71.tar.xz"},
			{Name: "autoconf", Version: "2.69", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.69.tar.xz"},
			{Name: "autoconf", Version: "2.59", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.59.tar.gz"},
			{Name: "autoconf", Version: "2.13", URL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.13.tar.gz"},
		}...,
	)
	// Automake
	versions = append(
		versions,
		[]domain.Source{
			{Name: "automake", Version: "1.17", URL: "https://mirror.freedif.org/GNU/automake/automake-1.17.tar.xz"},
			{Name: "automake", Version: "1.16.5", URL: "https://mirror.freedif.org/GNU/automake/automake-1.16.5.tar.xz"},
			{Name: "automake", Version: "1.15.1", URL: "https://mirror.freedif.org/GNU/automake/automake-1.15.1.tar.xz"},
			{Name: "automake", Version: "1.9.6", URL: "https://mirror.freedif.org/GNU/automake/automake-1.9.6.tar.gz"},
			{Name: "automake", Version: "1.4-p6", URL: "https://mirror.freedif.org/GNU/automake/automake-1.4-p6.tar.gz"},
		}...,
	)
	// Bison
	versions = append(
		versions,
		[]domain.Source{
			{Name: "bison", Version: "1.28", URL: "https://mirror.freedif.org/GNU/bison/bison-1.28.tar.gz"},
			{Name: "bison", Version: "1.35", URL: "https://mirror.freedif.org/GNU/bison/bison-1.35.tar.gz"},
		}...,
	)
	// CMake
	versions = append(versions, r.rangeVersionCMake(3, 27, 0, 6)...)
	versions = append(versions, r.rangeVersionCMake(3, 26, 0, 5)...)
	versions = append(versions, r.rangeVersionCMake(3, 25, 0, 3)...)
	versions = append(versions, r.rangeVersionCMake(3, 24, 0, 2)...)
	versions = append(versions, r.rangeVersionCMake(3, 23, 0, 3)...)
	versions = append(versions, r.rangeVersionCMake(3, 22, 0, 1)...)
	versions = append(versions, r.rangeVersionCMake(3, 21, 0, 4)...)
	// CURL
	versions = append(
		versions,
		[]domain.Source{
			{Name: "curl", Version: "8.10.1", URL: "https://curl.se/download/curl-8.10.1.tar.gz"},
			{Name: "curl", Version: "7.88.1", URL: "https://curl.se/download/curl-7.88.1.tar.gz"},
			{Name: "curl", Version: "7.20.0", URL: "https://curl.se/download/curl-7.20.0.tar.gz"},
			{Name: "curl", Version: "7.12.1", URL: "https://curl.se/download/curl-7.12.1.tar.gz"},
			{Name: "curl", Version: "7.12.0", URL: "https://curl.se/download/curl-7.12.0.tar.gz"},
		}...,
	)
	// Flex
	versions = append(
		versions,
		domain.Source{Name: "flex", Version: "2.5.39", URL: "https://github.com/westes/flex/releases/download/flex-2.5.39/flex-2.5.39.tar.gz"},
	)
	// Libtool
	versions = append(
		versions,
		[]domain.Source{
			{Name: "libtool", Version: "2.5.4", URL: "https://mirror.freedif.org/GNU/libtool/libtool-2.5.4.tar.xz"},
			{Name: "libtool", Version: "2.4.7", URL: "https://mirror.freedif.org/GNU/libtool/libtool-2.4.7.tar.xz"},
			{Name: "libtool", Version: "2.4.6", URL: "https://mirror.freedif.org/GNU/libtool/libtool-2.4.6.tar.xz"},
			{Name: "libtool", Version: "1.5.26", URL: "https://mirror.freedif.org/GNU/libtool/libtool-1.5.26.tar.gz"},
		}...,
	)
	// LibXML2
	versions = append(
		versions,
		[]domain.Source{
			{Name: "libxml2", Version: "2.12.7", URL: "https://download.gnome.org/sources/libxml2/2.12/libxml2-2.12.7.tar.xz"},
			{Name: "libxml2", Version: "2.11.7", URL: "https://download.gnome.org/sources/libxml2/2.11/libxml2-2.11.7.tar.xz"},
			{Name: "libxml2", Version: "2.9.14", URL: "https://download.gnome.org/sources/libxml2/2.9/libxml2-2.9.14.tar.xz"},
			{Name: "libxml2", Version: "2.6.30", URL: "https://github.com/GNOME/libxml2/archive/refs/tags/LIBXML2_2_6_30.tar.gz"},
		}...,
	)
	// M4
	versions = append(
		versions,
		domain.Source{Name: "m4", Version: "1.4.19", URL: "https://mirror.freedif.org/GNU/m4/m4-1.4.19.tar.xz"},
	)
	// Oniguruma
	versions = append(
		versions,
		[]domain.Source{
			{Name: "oniguruma", Version: "6.9.9", URL: "https://github.com/kkos/oniguruma/releases/download/v6.9.9/onig-6.9.9.tar.gz"},
			{Name: "oniguruma", Version: "6.9.8", URL: "https://github.com/kkos/oniguruma/releases/download/v6.9.8/onig-6.9.8.tar.gz"},
			{Name: "oniguruma", Version: "5.9.6", URL: "https://github.com/kkos/oniguruma/releases/download/v5.9.6/onig-5.9.6.tar.gz"},
		}...,
	)
	// OpenSSL
	versions = append(
		versions,
		[]domain.Source{
			{Name: "openssl", Version: "3.3.2", URL: "https://github.com/openssl/openssl/releases/download/openssl-3.3.2/openssl-3.3.2.tar.gz"},
			{Name: "openssl", Version: "3.0.14", URL: "https://github.com/openssl/openssl/releases/download/openssl-3.0.14/openssl-3.0.14.tar.gz"},
			{Name: "openssl", Version: "1.1.1w", URL: "https://github.com/openssl/openssl/releases/download/openssl-1.1.1w/openssl-1.1.1w.tar.gz"},
			{Name: "openssl", Version: "1.0.1u", URL: "https://www.openssl.org/source/openssl-1.0.1u.tar.gz"},
			{Name: "openssl", Version: "0.9.8zh", URL: "https://www.openssl.org/source/openssl-0.9.8zh.tar.gz"},
		}...,
	)
	// Sort versions descending
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *SourceRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Source {
	count := endPatch - startPatch + 1
	versions := make([]domain.Source, 0, count)
	for patch := startPatch; patch <= endPatch; patch++ {
		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		var url string
		if major == 4 {
			url = fmt.Sprintf("https://museum.php.net/php4/php-%s.tar.gz", versionStr)
		} else if major == 5 && minor <= 2 {
			url = fmt.Sprintf("https://museum.php.net/php5/php-%s.tar.gz", versionStr)
		} else {
			url = fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", versionStr)
		}
		versions = append(versions, domain.Source{
			Name:    "php",
			Version: versionStr,
			URL:     url,
		})
	}
	return versions
}

func (r *SourceRepository) perlSource(version string) domain.Source {
	ext := "tar.gz"
	if version < "5.20.0" {
		ext = "tar.bz2"
	}
	return domain.Source{
		Name:    "perl",
		Version: version,
		URL:     "https://www.cpan.org/src/5.0/perl-" + version + "." + ext,
	}
}

func (r *SourceRepository) rangeVersionCMake(major, minor, startPatch, endPatch int) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Source{
			Name:    "cmake",
			Version: fmt.Sprintf("%d.%d.%d", major, minor, patch),
			URL:     fmt.Sprintf("https://github.com/Kitware/CMake/releases/download/v%d.%d.%d/cmake-%d.%d.%d-linux-x86_64.tar.gz", major, minor, patch, major, minor, patch),
		})
	}
	return versions
}
