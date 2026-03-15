package dependency

import (
	"fmt"
	"strings"

	"github.com/supanadit/phpv/domain"
)

type DependencyPattern struct {
	URLTemplate    string
	Extension      string
	BuildCommands  []string
	ConfigureFlags []string
}

type DependencyURLConfig struct {
	Default DependencyPattern
	Exact   map[string]DependencyPattern
	Ranges  []VersionRange
}

type VersionRange struct {
	Min     string
	Max     string
	Pattern DependencyPattern
}

var urlConfigs = map[string]DependencyURLConfig{
	"perl": {
		Default: DependencyPattern{
			URLTemplate: "https://www.cpan.org/src/5.0/perl-%s.tar.gz",
			Extension:   ".tar.gz",
		},
	},
	"m4": {
		Default: DependencyPattern{
			URLTemplate: "https://mirror.freedif.org/GNU/m4/m4-%s.tar.xz",
			Extension:   ".tar.xz",
		},
	},
	"autoconf": {
		Default: DependencyPattern{
			URLTemplate: "https://mirror.freedif.org/GNU/autoconf/autoconf-%s.tar.xz",
			Extension:   ".tar.xz",
		},
		Exact: map[string]DependencyPattern{
			"2.13": {
				URLTemplate: "https://mirror.freedif.org/GNU/autoconf/autoconf-%s.tar.gz",
				Extension:   ".tar.gz",
			},
		},
	},
	"automake": {
		Default: DependencyPattern{
			URLTemplate: "https://mirror.freedif.org/GNU/automake/automake-%s.tar.xz",
			Extension:   ".tar.xz",
		},
		Exact: map[string]DependencyPattern{
			"1.4-p6": {
				URLTemplate: "https://mirror.freedif.org/GNU/automake/automake-%s.tar.gz",
				Extension:   ".tar.gz",
			},
		},
	},
	"libtool": {
		Default: DependencyPattern{
			URLTemplate: "https://mirror.freedif.org/GNU/libtool/libtool-%s.tar.xz",
			Extension:   ".tar.xz",
		},
		Exact: map[string]DependencyPattern{
			"1.5.26": {
				URLTemplate: "https://mirror.freedif.org/GNU/libtool/libtool-%s.tar.gz",
				Extension:   ".tar.gz",
			},
		},
	},
	"re2c": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/skvadrik/re2c/releases/download/%s/re2c-%s.tar.xz",
			Extension:   ".tar.xz",
		},
		Exact: map[string]DependencyPattern{
			"0.16": {
				URLTemplate: "https://github.com/skvadrik/re2c/releases/download/0.16/re2c-0.16.tar.gz",
				Extension:   ".tar.gz",
			},
			"0.14": {
				URLTemplate: "https://github.com/skvadrik/re2c/releases/download/0.14/re2c-0.14.tar.gz",
				Extension:   ".tar.gz",
			},
		},
		Ranges: []VersionRange{
			{
				Max: "1.0",
				Pattern: DependencyPattern{
					URLTemplate: "https://github.com/skvadrik/re2c/releases/download/%s/re2c-%s.tar.gz",
					Extension:   ".tar.gz",
				},
			},
		},
	},
	"zlib": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/madler/zlib/releases/download/v%s/zlib-%s.tar.gz",
			Extension:   ".tar.gz",
		},
	},
	"libxml2": {
		Default: DependencyPattern{
			URLTemplate: "https://download.gnome.org/sources/libxml2/%s/libxml2-%s.tar.xz",
			Extension:   ".tar.xz",
		},
	},
	"openssl": {
		Default: DependencyPattern{
			URLTemplate: "https://www.openssl.org/source/openssl-%s.tar.gz",
			Extension:   ".tar.gz",
		},
	},
	"curl": {
		Default: DependencyPattern{
			URLTemplate: "https://curl.se/download/curl-%s.tar.gz",
			Extension:   ".tar.gz",
		},
		Exact: map[string]DependencyPattern{
			"7.12.0": {
				URLTemplate: "https://curl.se/download/archeology/curl-7.12.0.tar.gz",
				Extension:   ".tar.gz",
			},
			"7.12.1": {
				URLTemplate: "https://curl.se/download/archeology/curl-7.12.1.tar.gz",
				Extension:   ".tar.gz",
			},
		},
		Ranges: []VersionRange{
			{
				Max: "7.20",
				Pattern: DependencyPattern{
					URLTemplate: "https://curl.se/download/archeology/curl-%s.tar.gz",
					Extension:   ".tar.gz",
				},
			},
		},
	},
	"oniguruma": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/kkos/oniguruma/releases/download/v%s/onig-%s.tar.gz",
			Extension:   ".tar.gz",
		},
	},
	"llvm": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/llvm/llvm-project/releases/download/llvmorg-%s/LLVM-%s-Linux-X64.tar.xz",
			Extension:   ".tar.xz",
		},
	},
	"cmake": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/Kitware/CMake/releases/download/v%s/cmake-%s-linux-x86_64.tar.gz",
			Extension:   ".tar.gz",
		},
	},
}

func (c *DependencyURLConfig) getPattern(version string) DependencyPattern {
	if pattern, ok := c.Exact[version]; ok {
		return pattern
	}

	for _, r := range c.Ranges {
		if inRange(version, r.Min, r.Max) {
			return r.Pattern
		}
	}

	return c.Default
}

func (c *DependencyURLConfig) buildURL(version string) string {
	pattern := c.getPattern(version)

	if strings.Contains(pattern.URLTemplate, "%s") && strings.Contains(pattern.URLTemplate, "%s") {
		return fmt.Sprintf(pattern.URLTemplate, version, version)
	}
	if strings.Contains(pattern.URLTemplate, "%s") {
		return fmt.Sprintf(pattern.URLTemplate, version)
	}
	return pattern.URLTemplate
}

func inRange(version, min, max string) bool {
	if min != "" && version < min {
		return false
	}
	if max != "" && version >= max {
		return false
	}
	return true
}

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

	PerlOverride      *DependencyPattern
	M4Override        *DependencyPattern
	AutoconfOverride  *DependencyPattern
	AutomakeOverride  *DependencyPattern
	LibtoolOverride   *DependencyPattern
	Re2cOverride      *DependencyPattern
	ZlibOverride      *DependencyPattern
	Libxml2Override   *DependencyPattern
	OpenSSLEverride   *DependencyPattern
	CurlOverride      *DependencyPattern
	OnigurumaOverride *DependencyPattern
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
		newCMakeDependency(config),
		newPerlDependency(config),
		newM4Dependency(config),
		newAutoconfDependency(config),
		newAutomakeDependency(config),
		newLibtoolDependency(config),
		newRe2cDependency(config),
		newZlibDependency(config),
		newLibxml2Dependency(config),
		newOpenSSLDependency(config),
		newCurlDependency(config),
		newOnigurumaDependency(config),
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

func getURL(config *DependencyURLConfig, version string, override *DependencyPattern) string {
	if override != nil && override.URLTemplate != "" {
		if strings.Contains(override.URLTemplate, "%s") && strings.Count(override.URLTemplate, "%s") >= 2 {
			return fmt.Sprintf(override.URLTemplate, version, version)
		}
		return fmt.Sprintf(override.URLTemplate, version)
	}
	return config.buildURL(version)
}

func getConfigureFlags(config *DependencyURLConfig, version string, override *DependencyPattern, defaults []string) []string {
	if override != nil && len(override.ConfigureFlags) > 0 {
		return override.ConfigureFlags
	}
	return defaults
}

func getBuildCommands(config *DependencyURLConfig, version string, override *DependencyPattern, defaults []string) []string {
	if override != nil && len(override.BuildCommands) > 0 {
		return override.BuildCommands
	}
	return defaults
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

func newCMakeDependency(config PHPVersionConfig) domain.Dependency {
	version := "3.30.0"

	return domain.Dependency{
		Name:           "cmake",
		Version:        version,
		DownloadURL:    fmt.Sprintf("https://github.com/Kitware/CMake/releases/download/v%s/cmake-%s-linux-x86_64.tar.gz", version, version),
		ConfigureFlags: []string{},
		BuildCommands:  []string{"prebuilt"},
		Dependencies:   []string{},
	}
}

func newPerlDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Perl
	urlConfig := urlConfigs["perl"]
	override := config.PerlOverride

	defaultFlags := []string{
		"-des",
		"-Dusethreads",
		"-Dccflags=-Wno-error=incompatible-pointer-types -Wno-error=pointer-arith -Wno-error=implicit-function-declaration -Wno-error=implicit-int -Wno-error=int-conversion -Wno-compound-token-split-by-macro -Wno-error=deprecated-declarations",
	}

	return domain.Dependency{
		Name:           "perl",
		Version:        version,
		DownloadURL:    getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, defaultFlags),
		BuildCommands:  []string{"./Configure"},
	}
}

func newM4Dependency(config PHPVersionConfig) domain.Dependency {
	version := config.M4
	urlConfig := urlConfigs["m4"]
	override := config.M4Override

	return domain.Dependency{
		Name:        "m4",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--disable-shared",
			"--enable-static",
		}),
	}
}

func newAutoconfDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Autoconf
	urlConfig := urlConfigs["autoconf"]
	override := config.AutoconfOverride

	return domain.Dependency{
		Name:        "autoconf",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--disable-shared",
			"--enable-static",
		}),
		Dependencies: []string{"m4"},
	}
}

func newAutomakeDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Automake
	urlConfig := urlConfigs["automake"]
	override := config.AutomakeOverride

	return domain.Dependency{
		Name:        "automake",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--disable-shared",
			"--enable-static",
		}),
		Dependencies: []string{"autoconf"},
	}
}

func newLibtoolDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Libtool
	urlConfig := urlConfigs["libtool"]
	override := config.LibtoolOverride

	return domain.Dependency{
		Name:        "libtool",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--disable-shared",
			"--enable-static",
		}),
		Dependencies: []string{"m4"},
	}
}

func newRe2cDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Re2c
	urlConfig := urlConfigs["re2c"]
	override := config.Re2cOverride

	return domain.Dependency{
		Name:        "re2c",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--disable-shared",
			"--enable-static",
		}),
		Dependencies: []string{"autoconf", "automake", "libtool"},
	}
}

func newZlibDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Zlib
	urlConfig := urlConfigs["zlib"]
	override := config.ZlibOverride

	defaultFlags := []string{
		"-DCMAKE_INSTALL_PREFIX=%s",
		"-DBUILD_SHARED_LIBS=OFF",
	}

	return domain.Dependency{
		Name:           "zlib",
		Version:        version,
		DownloadURL:    getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, defaultFlags),
		BuildCommands:  getBuildCommands(&urlConfig, version, override, []string{"cmake"}),
	}
}

func newLibxml2Dependency(config PHPVersionConfig) domain.Dependency {
	version := config.Libxml2
	dirVersion := config.Libxml2Dir
	urlConfig := urlConfigs["libxml2"]
	override := config.Libxml2Override

	url := getURL(&urlConfig, dirVersion, override)
	if !strings.Contains(url, "%s") {
		url = fmt.Sprintf("https://download.gnome.org/sources/libxml2/%s/libxml2-%s.tar.xz", dirVersion, version)
	}

	return domain.Dependency{
		Name:        "libxml2",
		Version:     version,
		DownloadURL: url,
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--without-python",
			"--without-readline",
			"--without-http",
			"--without-ftp",
			"--without-modules",
			"--without-lzma",
			"--disable-shared",
			"--enable-static",
		}),
		Dependencies: []string{"zlib"},
	}
}

func newOpenSSLDependency(config PHPVersionConfig) domain.Dependency {
	version := config.OpenSSL
	urlConfig := urlConfigs["openssl"]
	override := config.OpenSSLEverride

	return domain.Dependency{
		Name:        "openssl",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"no-shared",
			"no-tests",
		}),
		BuildCommands: getBuildCommands(&urlConfig, version, override, []string{"./config"}),
		Dependencies:  []string{"perl"},
	}
}

func newCurlDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Curl
	urlConfig := urlConfigs["curl"]
	override := config.CurlOverride

	return domain.Dependency{
		Name:        "curl",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--with-openssl",
			"--with-zlib",
			"--disable-shared",
			"--enable-static",
			"--without-libssh2",
			"--without-nghttp2",
			"--without-libidn2",
			"--without-libpsl",
			"--disable-ldap",
		}),
		Dependencies: []string{"openssl", "zlib", "autoconf", "automake", "libtool"},
	}
}

func newOnigurumaDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Oniguruma
	urlConfig := urlConfigs["oniguruma"]
	override := config.OnigurumaOverride

	return domain.Dependency{
		Name:        "oniguruma",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--disable-shared",
			"--enable-static",
		}),
	}
}
