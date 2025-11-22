package dependency

import "github.com/supanadit/phpv/domain"

// GetDependenciesForVersion returns the required dependencies for a PHP version
func GetDependenciesForVersion(version domain.Version) []domain.Dependency {
	// Get LLVM version for this PHP version
	llvmVersion := domain.GetLLVMVersionForPHP(version)
	llvmDep := getLLVMDependency(llvmVersion)

	// For PHP 8.3+, we need these dependencies
	if version.Major == 8 && version.Minor >= 3 {
		deps := getPHP83Dependencies()
		return append([]domain.Dependency{llvmDep}, deps...)
	}

	// For PHP 8.0-8.2
	if version.Major == 8 {
		deps := getPHP80Dependencies()
		return append([]domain.Dependency{llvmDep}, deps...)
	}

	// For PHP 7.x
	if version.Major == 7 {
		deps := getPHP7Dependencies()
		return append([]domain.Dependency{llvmDep}, deps...)
	}

	// Default set for older versions (PHP 5.x, etc.)
	deps := getDefaultDependencies()
	return append([]domain.Dependency{llvmDep}, deps...)
}

func getLLVMDependency(llvmVersion domain.LLVMVersion) domain.Dependency {
	return domain.Dependency{
		Name:           "llvm",
		Version:        llvmVersion.Version,
		DownloadURL:    llvmVersion.DownloadURL,
		ConfigureFlags: []string{},
		BuildCommands:  []string{"prebuilt"},
		Dependencies:   []string{},
	}
}

func getCMakeDependency() domain.Dependency {
	return domain.Dependency{
		Name:           "cmake",
		Version:        "3.30.0",
		DownloadURL:    "https://github.com/Kitware/CMake/releases/download/v3.30.0/cmake-3.30.0-linux-x86_64.tar.gz",
		ConfigureFlags: []string{},
		BuildCommands:  []string{"prebuilt"},
		Dependencies:   []string{},
	}
}

func getPerlDependency() domain.Dependency {
	return domain.Dependency{
		Name:        "perl",
		Version:     "5.38.2",
		DownloadURL: "https://www.cpan.org/src/5.0/perl-5.38.2.tar.gz",
		ConfigureFlags: []string{
			"-des",
			"-Dusethreads",
		},
		BuildCommands: []string{
			"./Configure",
		},
	}
}

func getRe2cDependency() domain.Dependency {
	return domain.Dependency{
		Name:        "re2c",
		Version:     "3.1",
		DownloadURL: "https://github.com/skvadrik/re2c/releases/download/3.1/re2c-3.1.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf", "automake", "libtool"},
	}
}

func getM4Dependency() domain.Dependency {
	return domain.Dependency{
		Name:        "m4",
		Version:     "1.4.19",
		DownloadURL: "https://mirror.freedif.org/GNU/m4/m4-1.4.19.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
	}
}

func getAutoconfDependency() domain.Dependency {
	return domain.Dependency{
		Name:        "autoconf",
		Version:     "2.72",
		DownloadURL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.72.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func getAutomakeDependency() domain.Dependency {
	return domain.Dependency{
		Name:        "automake",
		Version:     "1.17",
		DownloadURL: "https://mirror.freedif.org/GNU/automake/automake-1.17.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf"},
	}
}

func getLibtoolDependency() domain.Dependency {
	return domain.Dependency{
		Name:        "libtool",
		Version:     "2.5.4",
		DownloadURL: "https://mirror.freedif.org/GNU/libtool/libtool-2.5.4.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func getPHP83Dependencies() []domain.Dependency {
	return []domain.Dependency{
		getCMakeDependency(),
		getPerlDependency(),
		getM4Dependency(),
		getAutoconfDependency(),
		getAutomakeDependency(),
		getLibtoolDependency(),
		getRe2cDependency(),
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
			Dependencies: []string{"perl"},
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
				"--without-libpsl",
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
		getCMakeDependency(),
		getPerlDependency(),
		getM4Dependency(),
		getAutoconfDependency(),
		getAutomakeDependency(),
		getLibtoolDependency(),
		getRe2cDependency(),
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
			Dependencies: []string{"perl"},
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
				"--without-libpsl",
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
		getCMakeDependency(),
		getPerlDependency(),
		getM4Dependency(),
		getAutoconfDependency(),
		getAutomakeDependency(),
		getLibtoolDependency(),
		getRe2cDependency(),
		{
			Name:        "zlib",
			Version:     "1.2.13",
			DownloadURL: "https://github.com/madler/zlib/releases/download/v1.2.13/zlib-1.2.13.tar.gz",
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
			Dependencies: []string{"perl"},
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
				"--without-libpsl",
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
