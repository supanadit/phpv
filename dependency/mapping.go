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

	// For PHP 5.x
	if version.Major == 5 {
		deps := getPHP5Dependencies()
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

// Version-specific dependency builders for PHP 8.3+
func getPerlDependency_PHP83() domain.Dependency {
	return domain.Dependency{
		Name:        "perl",
		Version:     "5.38.2",
		DownloadURL: "https://www.cpan.org/src/5.0/perl-5.38.2.tar.gz",
		ConfigureFlags: []string{
			"-des",
			"-Dusethreads",
			"-Dccflags=-Wno-error=incompatible-pointer-types -Wno-error=pointer-arith -Wno-error=implicit-function-declaration -Wno-error=implicit-int -Wno-error=int-conversion -Wno-compound-token-split-by-macro -Wno-error=deprecated-declarations",
		},
		BuildCommands: []string{
			"./Configure",
		},
	}
}

func getRe2cDependency_PHP83() domain.Dependency {
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

func getM4Dependency_PHP83() domain.Dependency {
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

func getAutoconfDependency_PHP83() domain.Dependency {
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

func getAutomakeDependency_PHP83() domain.Dependency {
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

func getLibtoolDependency_PHP83() domain.Dependency {
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

// Version-specific dependency builders for PHP 8.0-8.2
func getPerlDependency_PHP80() domain.Dependency {
	return domain.Dependency{
		Name:        "perl",
		Version:     "5.36.0",
		DownloadURL: "https://www.cpan.org/src/5.0/perl-5.36.0.tar.gz",
		ConfigureFlags: []string{
			"-des",
			"-Dusethreads",
			"-Dccflags=-Wno-error=incompatible-pointer-types -Wno-error=pointer-arith -Wno-error=implicit-function-declaration -Wno-error=implicit-int -Wno-error=int-conversion -Wno-compound-token-split-by-macro -Wno-error=deprecated-declarations",
		},
		BuildCommands: []string{
			"./Configure",
		},
	}
}

func getRe2cDependency_PHP80() domain.Dependency {
	return domain.Dependency{
		Name:        "re2c",
		Version:     "2.2",
		DownloadURL: "https://github.com/skvadrik/re2c/releases/download/2.2/re2c-2.2.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf", "automake", "libtool"},
	}
}

func getM4Dependency_PHP80() domain.Dependency {
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

func getAutoconfDependency_PHP80() domain.Dependency {
	return domain.Dependency{
		Name:        "autoconf",
		Version:     "2.71",
		DownloadURL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.71.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func getAutomakeDependency_PHP80() domain.Dependency {
	return domain.Dependency{
		Name:        "automake",
		Version:     "1.16.5",
		DownloadURL: "https://mirror.freedif.org/GNU/automake/automake-1.16.5.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf"},
	}
}

func getLibtoolDependency_PHP80() domain.Dependency {
	return domain.Dependency{
		Name:        "libtool",
		Version:     "2.4.7",
		DownloadURL: "https://mirror.freedif.org/GNU/libtool/libtool-2.4.7.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

// Version-specific dependency builders for PHP 7.x (older, stable versions)
func getPerlDependency_PHP7() domain.Dependency {
	return domain.Dependency{
		Name:        "perl",
		Version:     "5.32.1",
		DownloadURL: "https://www.cpan.org/src/5.0/perl-5.32.1.tar.gz",
		ConfigureFlags: []string{
			"-des",
			"-Dusethreads",
			"-Dccflags=-Wno-error=incompatible-pointer-types -Wno-error=pointer-arith -Wno-error=implicit-function-declaration -Wno-error=implicit-int -Wno-error=int-conversion -Wno-error=deprecated-declarations -Wno-error=address -Wno-error=sequence-point",
		},
		BuildCommands: []string{
			"./Configure",
		},
	}
}

func getRe2cDependency_PHP7() domain.Dependency {
	return domain.Dependency{
		Name:        "re2c",
		Version:     "1.3",
		DownloadURL: "https://github.com/skvadrik/re2c/releases/download/1.3/re2c-1.3.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf", "automake", "libtool"},
	}
}

func getM4Dependency_PHP7() domain.Dependency {
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

func getAutoconfDependency_PHP7() domain.Dependency {
	return domain.Dependency{
		Name:        "autoconf",
		Version:     "2.69",
		DownloadURL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.69.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func getAutomakeDependency_PHP7() domain.Dependency {
	return domain.Dependency{
		Name:        "automake",
		Version:     "1.15.1",
		DownloadURL: "https://mirror.freedif.org/GNU/automake/automake-1.15.1.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf"},
	}
}

func getLibtoolDependency_PHP7() domain.Dependency {
	return domain.Dependency{
		Name:        "libtool",
		Version:     "2.4.6",
		DownloadURL: "https://mirror.freedif.org/GNU/libtool/libtool-2.4.6.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func getPerlDependency_PHP5() domain.Dependency {
	return domain.Dependency{
		Name:        "perl",
		Version:     "5.32.1",
		DownloadURL: "https://www.cpan.org/src/5.0/perl-5.32.1.tar.gz",
		ConfigureFlags: []string{
			"-des",
			"-Dusethreads",
			"-Dccflags=-Wno-error=incompatible-pointer-types -Wno-error=pointer-arith -Wno-error=implicit-function-declaration -Wno-error=implicit-int -Wno-error=int-conversion -Wno-error=deprecated-declarations -Wno-error=address -Wno-error=sequence-point",
		},
		BuildCommands: []string{
			"./Configure",
		},
	}
}

func getRe2cDependency_PHP5() domain.Dependency {
	return domain.Dependency{
		Name:        "re2c",
		Version:     "0.16",
		DownloadURL: "https://github.com/skvadrik/re2c/releases/download/0.16/re2c-0.16.tar.gz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf", "automake", "libtool"},
	}
}

func getM4Dependency_PHP5() domain.Dependency {
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

func getAutoconfDependency_PHP5() domain.Dependency {
	return domain.Dependency{
		Name:        "autoconf",
		Version:     "2.69",
		DownloadURL: "https://mirror.freedif.org/GNU/autoconf/autoconf-2.69.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func getAutomakeDependency_PHP5() domain.Dependency {
	return domain.Dependency{
		Name:        "automake",
		Version:     "1.15",
		DownloadURL: "https://mirror.freedif.org/GNU/automake/automake-1.15.tar.xz",
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf"},
	}
}

func getLibtoolDependency_PHP5() domain.Dependency {
	return domain.Dependency{
		Name:        "libtool",
		Version:     "2.4.6",
		DownloadURL: "https://mirror.freedif.org/GNU/libtool/libtool-2.4.6.tar.xz",
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
		getPerlDependency_PHP83(),
		getM4Dependency_PHP83(),
		getAutoconfDependency_PHP83(),
		getAutomakeDependency_PHP83(),
		getLibtoolDependency_PHP83(),
		getRe2cDependency_PHP83(),
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
			Dependencies: []string{"openssl", "zlib", "autoconf", "automake", "libtool"},
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
		getPerlDependency_PHP80(),
		getM4Dependency_PHP80(),
		getAutoconfDependency_PHP80(),
		getAutomakeDependency_PHP80(),
		getLibtoolDependency_PHP80(),
		getRe2cDependency_PHP80(),
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
			Dependencies: []string{"openssl", "zlib", "autoconf", "automake", "libtool"},
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

func getPHP5Dependencies() []domain.Dependency {
	return []domain.Dependency{
		getCMakeDependency(),
		getPerlDependency_PHP5(),
		getM4Dependency_PHP5(),
		getAutoconfDependency_PHP5(),
		getAutomakeDependency_PHP5(),
		getLibtoolDependency_PHP5(),
		getRe2cDependency_PHP5(),
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
			Version:     "1.0.1u",
			DownloadURL: "https://www.openssl.org/source/openssl-1.0.1u.tar.gz",
			ConfigureFlags: []string{
				"no-shared",
				"no-tests",
				"no-gost",
			},
			BuildCommands: []string{
				"./config",
			},
		},
		{
			Name:        "curl",
			Version:     "7.12.0",
			DownloadURL: "https://curl.se/download/archeology/curl-7.12.0.tar.gz",
			ConfigureFlags: []string{
				"--with-openssl",
				"--with-zlib",
				"--disable-shared",
				"--enable-static",
				"--without-libssh2",
				"--disable-ldap",
				"--disable-ldaps",
			},
			BuildCommands: []string{
				"ac_cv_func_select=yes",
				"ac_cv_func_socket=yes",
			},
			Dependencies: []string{"openssl", "zlib"},
		},
		{
			Name:        "oniguruma",
			Version:     "5.9.6",
			DownloadURL: "https://github.com/kkos/oniguruma/releases/download/v5.9.6/onig-5.9.6.tar.gz",
			ConfigureFlags: []string{
				"--disable-shared",
				"--enable-static",
			},
		},
	}
}

func getDefaultDependencies() []domain.Dependency {
	// Same as PHP 5 for now
	return getPHP5Dependencies()
}

func getPHP7Dependencies() []domain.Dependency {
	return []domain.Dependency{
		getCMakeDependency(),
		getPerlDependency_PHP7(),
		getM4Dependency_PHP7(),
		getAutoconfDependency_PHP7(),
		getAutomakeDependency_PHP7(),
		getLibtoolDependency_PHP7(),
		getRe2cDependency_PHP7(),
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
			Dependencies: []string{"openssl", "zlib", "autoconf", "automake", "libtool"},
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
