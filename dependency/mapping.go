package dependency

import "github.com/supanadit/phpv/domain"

// GetDependenciesForVersion returns the required dependencies for a PHP version
func GetDependenciesForVersion(version domain.Version) []domain.Dependency {
	// For PHP 8.3+, we need these dependencies
	if version.Major == 8 && version.Minor >= 3 {
		return getPHP83Dependencies()
	}

	// For PHP 8.0-8.2
	if version.Major == 8 {
		return getPHP80Dependencies()
	}

	// For PHP 7.x
	if version.Major == 7 {
		return getPHP7Dependencies()
	}

	// Default set for older versions (PHP 5.x, etc.)
	return getDefaultDependencies()
}

func getPHP83Dependencies() []domain.Dependency {
	return []domain.Dependency{
		{
			Name:        "zlib",
			Version:     "1.3.1",
			DownloadURL: "https://github.com/madler/zlib/releases/download/v1.3.1/zlib-1.3.1.tar.gz",
			ConfigureFlags: []string{
				"-DCMAKE_INSTALL_PREFIX=%s",
				"-DBUILD_SHARED_LIBS=OFF",
			},
			BuildCommands: []string{
				"cmake",
			},
		},
		{
			Name:        "libxml2",
			Version:     "2.12.7",
			DownloadURL: "https://download.gnome.org/sources/libxml2/2.12/libxml2-2.12.7.tar.xz",
			ConfigureFlags: []string{
				"--without-python",
				"--without-readline",
				"--without-http",
				"--without-ftp",
				"--without-modules",
				"--without-lzma",
				"--disable-shared",
				"--enable-static",
			},
			Dependencies: []string{"zlib"},
		},
		{
			Name:        "openssl",
			Version:     "3.3.2",
			DownloadURL: "https://www.openssl.org/source/openssl-3.3.2.tar.gz",
			ConfigureFlags: []string{
				"no-shared",
				"no-tests",
			},
			BuildCommands: []string{
				// OpenSSL uses ./config instead of ./configure
				"./config",
			},
		},
		{
			Name:        "curl",
			Version:     "8.10.1",
			DownloadURL: "https://curl.se/download/curl-8.10.1.tar.gz",
			ConfigureFlags: []string{
				"--with-openssl",
				"--with-zlib",
				"--disable-shared",
				"--enable-static",
				"--without-libssh2",
				"--without-nghttp2",
				"--without-libidn2",
				"--disable-ldap",
			},
			Dependencies: []string{"openssl", "zlib"},
		},
		{
			Name:        "oniguruma",
			Version:     "6.9.9",
			DownloadURL: "https://github.com/kkos/oniguruma/releases/download/v6.9.9/onig-6.9.9.tar.gz",
			ConfigureFlags: []string{
				"--disable-shared",
				"--enable-static",
			},
		},
	}
}

func getPHP80Dependencies() []domain.Dependency {
	return []domain.Dependency{
		{
			Name:        "zlib",
			Version:     "1.3.1",
			DownloadURL: "https://github.com/madler/zlib/releases/download/v1.3.1/zlib-1.3.1.tar.gz",
			ConfigureFlags: []string{
				"-DCMAKE_INSTALL_PREFIX=%s",
				"-DBUILD_SHARED_LIBS=OFF",
			},
			BuildCommands: []string{
				"cmake",
			},
		},
		{
			Name:        "libxml2",
			Version:     "2.11.7",
			DownloadURL: "https://download.gnome.org/sources/libxml2/2.11/libxml2-2.11.7.tar.xz",
			ConfigureFlags: []string{
				"--without-python",
				"--without-readline",
				"--without-http",
				"--without-ftp",
				"--without-modules",
				"--without-lzma",
				"--disable-shared",
				"--enable-static",
			},
			Dependencies: []string{"zlib"},
		},
		{
			Name:        "openssl",
			Version:     "3.0.14",
			DownloadURL: "https://www.openssl.org/source/openssl-3.0.14.tar.gz",
			ConfigureFlags: []string{
				"no-shared",
				"no-tests",
			},
			BuildCommands: []string{
				"./config",
			},
		},
		{
			Name:        "curl",
			Version:     "8.10.1",
			DownloadURL: "https://curl.se/download/curl-8.10.1.tar.gz",
			ConfigureFlags: []string{
				"--with-openssl",
				"--with-zlib",
				"--disable-shared",
				"--enable-static",
				"--without-libssh2",
				"--without-nghttp2",
				"--without-libidn2",
				"--disable-ldap",
			},
			Dependencies: []string{"openssl", "zlib"},
		},
		{
			Name:        "oniguruma",
			Version:     "6.9.9",
			DownloadURL: "https://github.com/kkos/oniguruma/releases/download/v6.9.9/onig-6.9.9.tar.gz",
			ConfigureFlags: []string{
				"--disable-shared",
				"--enable-static",
			},
		},
	}
}

func getDefaultDependencies() []domain.Dependency {
	// Same as PHP 8.0 for now
	return getPHP80Dependencies()
}

func getPHP7Dependencies() []domain.Dependency {
	return []domain.Dependency{
		{
			Name:        "zlib",
			Version:     "1.2.13",
			DownloadURL: "https://github.com/madler/zlib/releases/download/v1.2.13/zlib-1.2.13.tar.gz",
			ConfigureFlags: []string{
				"--static",
			},
		},
		{
			Name:        "libxml2",
			Version:     "2.9.14",
			DownloadURL: "https://download.gnome.org/sources/libxml2/2.9/libxml2-2.9.14.tar.xz",
			ConfigureFlags: []string{
				"--without-python",
				"--without-readline",
				"--without-http",
				"--without-ftp",
				"--without-modules",
				"--without-lzma",
				"--disable-shared",
				"--enable-static",
			},
			Dependencies: []string{"zlib"},
		},
		{
			Name:        "openssl",
			Version:     "1.1.1w",
			DownloadURL: "https://www.openssl.org/source/openssl-1.1.1w.tar.gz",
			ConfigureFlags: []string{
				"no-shared",
				"no-tests",
			},
			BuildCommands: []string{
				// OpenSSL uses ./config instead of ./configure
				"./config",
			},
		},
		{
			Name:        "curl",
			Version:     "7.88.1",
			DownloadURL: "https://curl.se/download/curl-7.88.1.tar.gz",
			ConfigureFlags: []string{
				"--with-openssl",
				"--with-zlib",
				"--disable-shared",
				"--enable-static",
				"--without-libssh2",
				"--without-nghttp2",
				"--without-libidn2",
				"--disable-ldap",
			},
			Dependencies: []string{"openssl", "zlib"},
		},
		{
			Name:        "oniguruma",
			Version:     "6.9.8",
			DownloadURL: "https://github.com/kkos/oniguruma/releases/download/v6.9.8/onig-6.9.8.tar.gz",
			ConfigureFlags: []string{
				"--disable-shared",
				"--enable-static",
			},
		},
	}
}
