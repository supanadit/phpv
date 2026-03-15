package dependency

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
)

type PHPVersionConfig struct {
	Perl       string
	M4         string
	Autoconf   string
	Automake   string
	Libtool    string
	Re2c       string
	Zlib       string
	Libxml2    string
	Libxml2Dir string
	OpenSSL    string
	Curl       string
	Oniguruma  string
}

var versionRegistry = map[string]PHPVersionConfig{
	"8.3": {
		Perl: "5.38.2", M4: "1.4.19", Autoconf: "2.72", Automake: "1.17",
		Libtool: "2.5.4", Re2c: "3.1", Zlib: "1.3.1",
		Libxml2: "2.12.7", Libxml2Dir: "2.12",
		OpenSSL: "3.3.2", Curl: "8.10.1", Oniguruma: "6.9.9",
	},
	"8.2": {
		Perl: "5.36.0", M4: "1.4.19", Autoconf: "2.71", Automake: "1.16.5",
		Libtool: "2.4.7", Re2c: "2.2", Zlib: "1.3.1",
		Libxml2: "2.11.7", Libxml2Dir: "2.11",
		OpenSSL: "3.0.14", Curl: "8.10.1", Oniguruma: "6.9.9",
	},
	"8.1": {
		Perl: "5.36.0", M4: "1.4.19", Autoconf: "2.71", Automake: "1.16.5",
		Libtool: "2.4.7", Re2c: "2.2", Zlib: "1.3.1",
		Libxml2: "2.11.7", Libxml2Dir: "2.11",
		OpenSSL: "3.0.14", Curl: "8.10.1", Oniguruma: "6.9.9",
	},
	"8.0": {
		Perl: "5.36.0", M4: "1.4.19", Autoconf: "2.71", Automake: "1.16.5",
		Libtool: "2.4.7", Re2c: "2.2", Zlib: "1.3.1",
		Libxml2: "2.11.7", Libxml2Dir: "2.11",
		OpenSSL: "3.0.14", Curl: "8.10.1", Oniguruma: "6.9.9",
	},
	"7.4": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.69", Automake: "1.15.1",
		Libtool: "2.4.6", Re2c: "1.3", Zlib: "1.2.13",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.1.1w", Curl: "7.88.1", Oniguruma: "6.9.8",
	},
	"7.3": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.69", Automake: "1.15.1",
		Libtool: "2.4.6", Re2c: "1.3", Zlib: "1.2.13",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.1.1w", Curl: "7.88.1", Oniguruma: "6.9.8",
	},
	"7.2": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.69", Automake: "1.15.1",
		Libtool: "2.4.6", Re2c: "1.3", Zlib: "1.2.13",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.1.1w", Curl: "7.88.1", Oniguruma: "6.9.8",
	},
	"7.1": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.69", Automake: "1.15.1",
		Libtool: "2.4.6", Re2c: "1.3", Zlib: "1.2.13",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.1.1w", Curl: "7.88.1", Oniguruma: "6.9.8",
	},
	"7.0": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.69", Automake: "1.15.1",
		Libtool: "2.4.6", Re2c: "1.3", Zlib: "1.2.13",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.1.1w", Curl: "7.88.1", Oniguruma: "6.9.8",
	},
	"5.6": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.13", Automake: "1.4-p6",
		Libtool: "1.5.26", Re2c: "0.16", Zlib: "1.3.1",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.0.1u", Curl: "7.12.0", Oniguruma: "5.9.6",
	},
	"5.5": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.13", Automake: "1.4-p6",
		Libtool: "1.5.26", Re2c: "0.16", Zlib: "1.3.1",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.0.1u", Curl: "7.12.0", Oniguruma: "5.9.6",
	},
	"5.4": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.13", Automake: "1.4-p6",
		Libtool: "1.5.26", Re2c: "0.16", Zlib: "1.3.1",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.0.1u", Curl: "7.12.0", Oniguruma: "5.9.6",
	},
	"5.3": {
		Perl: "5.32.1", M4: "1.4.19", Autoconf: "2.13", Automake: "1.4-p6",
		Libtool: "1.5.26", Re2c: "0.16", Zlib: "1.3.1",
		Libxml2: "2.9.14", Libxml2Dir: "2.9",
		OpenSSL: "1.0.1u", Curl: "7.12.0", Oniguruma: "5.9.6",
	},
}

func GetDependenciesForVersion(version domain.Version) []domain.Dependency {
	llvmVersion := domain.GetLLVMVersionForPHP(version)
	config := getConfigForVersion(version)

	deps := []domain.Dependency{
		newLLVMDependency(llvmVersion),
		newCMakeDependency(),
		newPerlDependency(config.Perl),
		newM4Dependency(config.M4),
		newAutoconfDependency(config.Autoconf),
		newAutomakeDependency(config.Automake),
		newLibtoolDependency(config.Libtool),
		newRe2cDependency(config.Re2c),
		newZlibDependency(config.Zlib),
		newLibxml2Dependency(config.Libxml2, config.Libxml2Dir),
		newOpenSSLDependency(config.OpenSSL),
		newCurlDependency(config.Curl),
		newOnigurumaDependency(config.Oniguruma),
	}

	return deps
}

func getConfigForVersion(v domain.Version) PHPVersionConfig {
	versionKey := fmt.Sprintf("%d.%d", v.Major, v.Minor)

	if cfg, ok := versionRegistry[versionKey]; ok {
		return cfg
	}

	if v.Major == 8 && v.Minor >= 3 {
		return versionRegistry["8.3"]
	}

	if v.Major == 8 {
		return versionRegistry["8.0"]
	}
	if v.Major == 7 {
		return versionRegistry["7.4"]
	}
	return versionRegistry["5.6"]
}

func newLLVMDependency(llvmVersion domain.LLVMVersion) domain.Dependency {
	return domain.Dependency{
		Name:           "llvm",
		Version:        llvmVersion.Version,
		DownloadURL:    llvmVersion.DownloadURL,
		ConfigureFlags: []string{},
		BuildCommands:  []string{"prebuilt"},
		Dependencies:   []string{},
	}
}

func newCMakeDependency() domain.Dependency {
	return domain.Dependency{
		Name:           "cmake",
		Version:        "3.30.0",
		DownloadURL:    "https://github.com/Kitware/CMake/releases/download/v3.30.0/cmake-3.30.0-linux-x86_64.tar.gz",
		ConfigureFlags: []string{},
		BuildCommands:  []string{"prebuilt"},
		Dependencies:   []string{},
	}
}

func newPerlDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "perl",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://www.cpan.org/src/5.0/perl-%s.tar.gz", version),
		ConfigureFlags: []string{
			"-des",
			"-Dusethreads",
			"-Dccflags=-Wno-error=incompatible-pointer-types -Wno-error=pointer-arith -Wno-error=implicit-function-declaration -Wno-error=implicit-int -Wno-error=int-conversion -Wno-compound-token-split-by-macro -Wno-error=deprecated-declarations",
		},
		BuildCommands: []string{"./Configure"},
	}
}

func newM4Dependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "m4",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://mirror.freedif.org/GNU/m4/m4-%s.tar.xz", version),
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
	}
}

func newAutoconfDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "autoconf",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://mirror.freedif.org/GNU/autoconf/autoconf-%s.tar.xz", version),
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func newAutomakeDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "automake",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://mirror.freedif.org/GNU/automake/automake-%s.tar.xz", version),
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf"},
	}
}

func newLibtoolDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "libtool",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://mirror.freedif.org/GNU/libtool/libtool-%s.tar.xz", version),
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"m4"},
	}
}

func newRe2cDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "re2c",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://github.com/skvadrik/re2c/releases/download/%s/re2c-%s.tar.xz", version, version),
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
		Dependencies: []string{"autoconf", "automake", "libtool"},
	}
}

func newZlibDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "zlib",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://github.com/madler/zlib/releases/download/v%s/zlib-%s.tar.gz", version, version),
		ConfigureFlags: []string{
			"-DCMAKE_INSTALL_PREFIX=%s",
			"-DBUILD_SHARED_LIBS=OFF",
		},
		BuildCommands: []string{"cmake"},
	}
}

func newLibxml2Dependency(version, dirVersion string) domain.Dependency {
	return domain.Dependency{
		Name:        "libxml2",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://download.gnome.org/sources/libxml2/%s/libxml2-%s.tar.xz", dirVersion, version),
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
	}
}

func newOpenSSLDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "openssl",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://www.openssl.org/source/openssl-%s.tar.gz", version),
		ConfigureFlags: []string{
			"no-shared",
			"no-tests",
		},
		BuildCommands: []string{"./config"},
		Dependencies:  []string{"perl"},
	}
}

func newCurlDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "curl",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://curl.se/download/curl-%s.tar.gz", version),
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
	}
}

func newOnigurumaDependency(version string) domain.Dependency {
	return domain.Dependency{
		Name:        "oniguruma",
		Version:     version,
		DownloadURL: fmt.Sprintf("https://github.com/kkos/oniguruma/releases/download/v%s/onig-%s.tar.gz", version, version),
		ConfigureFlags: []string{
			"--disable-shared",
			"--enable-static",
		},
	}
}
