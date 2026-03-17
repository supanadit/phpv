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
			"2.59": {
				URLTemplate: "https://mirror.freedif.org/GNU/autoconf/autoconf-%s.tar.gz",
				Extension:   ".tar.gz",
			},
			"2.69": {
				URLTemplate: "https://mirror.freedif.org/GNU/autoconf/autoconf-%s.tar.xz",
				Extension:   ".tar.xz",
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
			"1.9.6": {
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
	"flex": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/westes/flex/releases/download/v%s/flex-%s.tar.gz",
			Extension:   ".tar.gz",
		},
		Exact: map[string]DependencyPattern{
			"2.5.39": {
				URLTemplate: "https://github.com/westes/flex/releases/download/flex-2.5.39/flex-2.5.39.tar.gz",
				Extension:   ".tar.gz",
			},
			"2.6.4": {
				URLTemplate: "https://github.com/westes/flex/releases/download/v2.6.4/flex-2.6.4.tar.gz",
				Extension:   ".tar.gz",
			},
		},
	},
	"bison": {
		Default: DependencyPattern{
			URLTemplate: "https://mirror.freedif.org/GNU/bison/bison-%s.tar.xz",
			Extension:   ".tar.xz",
		},
		Exact: map[string]DependencyPattern{
			"1.28": {
				URLTemplate: "https://mirror.freedif.org/GNU/bison/bison-%s.tar.gz",
				Extension:   ".tar.gz",
			},
			"1.35": {
				URLTemplate: "https://mirror.freedif.org/GNU/bison/bison-%s.tar.gz",
				Extension:   ".tar.gz",
			},
			"2.4.1": {
				URLTemplate: "https://mirror.freedif.org/GNU/bison/bison-%s.tar.gz",
				Extension:   ".tar.gz",
			},
			"2.6.4": {
				URLTemplate: "https://mirror.freedif.org/GNU/bison/bison-%s.tar.xz",
				Extension:   ".tar.xz",
			},
			"3.0": {
				URLTemplate: "https://mirror.freedif.org/GNU/bison/bison-%s.tar.xz",
				Extension:   ".tar.xz",
			},
		},
	},
	"re2c": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/skvadrik/re2c/releases/download/%s/re2c-%s.tar.xz",
			Extension:   ".tar.xz",
		},
		Exact: map[string]DependencyPattern{
			"0.14": {
				URLTemplate: "https://github.com/skvadrik/re2c/releases/download/0.14/re2c-0.14.tar.gz",
				Extension:   ".tar.gz",
			},
			"0.16": {
				URLTemplate: "https://github.com/skvadrik/re2c/releases/download/0.16/re2c-0.16.tar.gz",
				Extension:   ".tar.gz",
			},
			"1.0": {
				URLTemplate: "https://github.com/skvadrik/re2c/releases/download/1.0/re2c-%s.tar.gz",
				Extension:   ".tar.gz",
			},
		},
	},
	"cmake": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/Kitware/CMake/releases/download/v%s/cmake-%s-linux-x86_64.tar.gz",
			Extension:   ".tar.gz",
		},
	},
	"libxml2": {
		Default: DependencyPattern{
			URLTemplate: "https://download.gnome.org/sources/libxml2/%s/libxml2-%s.tar.xz",
			Extension:   ".tar.xz",
		},
		Exact: map[string]DependencyPattern{
			"2.6.30": {
				URLTemplate: "https://github.com/GNOME/libxml2/archive/refs/tags/LIBXML2_2_6_30.tar.gz",
				Extension:   ".tar.gz",
			},
		},
	},
	"zlib": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/madler/zlib/releases/download/v%s/zlib-%s.tar.gz",
			Extension:   ".tar.gz",
		},
	},
	"openssl": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/openssl/openssl/releases/download/openssl-%s/openssl-%s.tar.gz",
			Extension:   ".tar.gz",
		},
		Exact: map[string]DependencyPattern{
			"1.0.2u": {
				URLTemplate: "https://www.openssl.org/source/openssl-1.0.2u.tar.gz",
				Extension:   ".tar.gz",
			},
			"1.0.1u": {
				URLTemplate: "https://www.openssl.org/source/openssl-1.0.1u.tar.gz",
				Extension:   ".tar.gz",
			},
			"0.9.8zh": {
				URLTemplate: "https://www.openssl.org/source/openssl-0.9.8zh.tar.gz",
				Extension:   ".tar.gz",
			},
		},
	},
	"curl": {
		Default: DependencyPattern{
			URLTemplate: "https://curl.se/download/curl-%s.tar.gz",
			Extension:   ".tar.gz",
		},
	},
	"oniguruma": {
		Default: DependencyPattern{
			URLTemplate: "https://github.com/kkos/oniguruma/releases/download/v%s/onig-%s.tar.gz",
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

	if strings.Count(pattern.URLTemplate, "%s") >= 2 {
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
	Perl       domain.DependencyVersionSpec
	M4         domain.DependencyVersionSpec
	Autoconf   domain.DependencyVersionSpec
	Automake   domain.DependencyVersionSpec
	Libtool    domain.DependencyVersionSpec
	Re2c       domain.DependencyVersionSpec
	Flex       domain.DependencyVersionSpec
	Bison      domain.DependencyVersionSpec
	Zlib       domain.DependencyVersionSpec
	Libxml2    domain.DependencyVersionSpec
	Libxml2Dir string
	OpenSSL    domain.DependencyVersionSpec
	Curl       domain.DependencyVersionSpec
	Oniguruma  domain.DependencyVersionSpec

	PerlOverride      *DependencyPattern
	M4Override        *DependencyPattern
	AutoconfOverride  *DependencyPattern
	AutomakeOverride  *DependencyPattern
	LibtoolOverride   *DependencyPattern
	Re2cOverride      *DependencyPattern
	FlexOverride      *DependencyPattern
	BisonOverride     *DependencyPattern
	ZlibOverride      *DependencyPattern
	Libxml2Override   *DependencyPattern
	OpenSSLEverride   *DependencyPattern
	CurlOverride      *DependencyPattern
	OnigurumaOverride *DependencyPattern
}

func parseDepSpec(constraint string, optional bool) domain.DependencyVersionSpec {
	spec := domain.DependencyVersionSpec{
		ConstraintStr: constraint,
		Optional:      optional,
	}
	c, err := domain.ParseConstraint(constraint)
	if err != nil {
		spec.Constraint = &domain.DependencyConstraint{
			Optional: optional,
		}
	} else {
		c.Optional = optional
		spec.Constraint = c
	}
	return spec
}

var versionRegistry = map[string]PHPVersionConfig{
	"8.3": {
		Perl:       parseDepSpec("5.38.2", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.72", false),
		Automake:   parseDepSpec("1.17", false),
		Libtool:    parseDepSpec("2.5.4", false),
		Re2c:       parseDepSpec("3.1", false),
		Zlib:       parseDepSpec("1.3.1", false),
		Libxml2:    parseDepSpec("2.12.7|~2.12.0", false),
		Libxml2Dir: "2.12",
		OpenSSL:    parseDepSpec("3.3.2", false),
		Curl:       parseDepSpec("8.10.1|>=8.0.0", false),
		Oniguruma:  parseDepSpec("6.9.9|~6.9.0", false),
	},
	"8.2": {
		Perl:       parseDepSpec("5.36.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.71", false),
		Automake:   parseDepSpec("1.16.5", false),
		Libtool:    parseDepSpec("2.4.7", false),
		Re2c:       parseDepSpec("2.2", false),
		Zlib:       parseDepSpec("1.3.1", false),
		Libxml2:    parseDepSpec("2.11.7|~2.11.0", false),
		Libxml2Dir: "2.11",
		OpenSSL:    parseDepSpec("3.0.14|>=3.0.0,<3.1.0", false),
		Curl:       parseDepSpec("8.10.1|>=8.0.0", false),
		Oniguruma:  parseDepSpec("6.9.9|~6.9.0", false),
	},
	"8.1": {
		Perl:       parseDepSpec("5.36.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.71", false),
		Automake:   parseDepSpec("1.16.5", false),
		Libtool:    parseDepSpec("2.4.7", false),
		Re2c:       parseDepSpec("2.2", false),
		Zlib:       parseDepSpec("1.3.1", false),
		Libxml2:    parseDepSpec("2.11.7|~2.11.0", false),
		Libxml2Dir: "2.11",
		OpenSSL:    parseDepSpec("3.0.14|>=3.0.0,<3.1.0", false),
		Curl:       parseDepSpec("8.10.1|>=8.0.0", false),
		Oniguruma:  parseDepSpec("6.9.9|~6.9.0", false),
	},
	"8.0": {
		Perl:       parseDepSpec("5.36.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.71", false),
		Automake:   parseDepSpec("1.16.5", false),
		Libtool:    parseDepSpec("2.4.7", false),
		Re2c:       parseDepSpec("2.2", false),
		Zlib:       parseDepSpec("1.3.1", false),
		Libxml2:    parseDepSpec("2.11.7|~2.11.0", false),
		Libxml2Dir: "2.11",
		OpenSSL:    parseDepSpec("3.0.14|>=3.0.0,<3.1.0", false),
		Curl:       parseDepSpec("8.10.1|>=8.0.0", false),
		Oniguruma:  parseDepSpec("6.9.9|~6.9.0", false),
	},
	"7.4": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.69", false),
		Automake:   parseDepSpec("1.15.1", false),
		Libtool:    parseDepSpec("2.4.6", false),
		Re2c:       parseDepSpec("1.3", false),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.1.1w|>=1.1.0,<1.2.0", false),
		Curl:       parseDepSpec("7.88.1|>=7.80.0", false),
		Oniguruma:  parseDepSpec("6.9.8|~6.9.0", false),
	},
	"7.3": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.69", false),
		Automake:   parseDepSpec("1.15.1", false),
		Libtool:    parseDepSpec("2.4.6", false),
		Re2c:       parseDepSpec("1.3", false),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.1.1w|>=1.1.0,<1.2.0", false),
		Curl:       parseDepSpec("7.88.1|>=7.80.0", false),
		Oniguruma:  parseDepSpec("6.9.8|~6.9.0", false),
	},
	"7.2": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.69", false),
		Automake:   parseDepSpec("1.15.1", false),
		Libtool:    parseDepSpec("2.4.6", false),
		Re2c:       parseDepSpec("1.3", false),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.1.1w|>=1.1.0,<1.2.0", false),
		Curl:       parseDepSpec("7.88.1|>=7.80.0", false),
		Oniguruma:  parseDepSpec("6.9.8|~6.9.0", false),
	},
	"7.1": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.69", false),
		Automake:   parseDepSpec("1.15.1", false),
		Libtool:    parseDepSpec("2.4.6", false),
		Re2c:       parseDepSpec("1.3", false),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.1.1w|>=1.1.0,<1.2.0", false),
		Curl:       parseDepSpec("7.88.1|>=7.80.0", false),
		Oniguruma:  parseDepSpec("6.9.8|~6.9.0", false),
	},
	"7.0": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.69", false),
		Automake:   parseDepSpec("1.15.1", false),
		Libtool:    parseDepSpec("2.4.6", false),
		Re2c:       parseDepSpec("1.3", false),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.1.1w|>=1.1.0,<1.2.0", false),
		Curl:       parseDepSpec("7.88.1|>=7.80.0", false),
		Oniguruma:  parseDepSpec("6.9.8|~6.9.0", false),
	},
	"5.6": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.59", false),
		Automake:   parseDepSpec("1.9.6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.16", false),
		Zlib:       parseDepSpec("1.3.1|>=1.2.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.0.2u|>=1.0.2,<1.0.3", false),
		Curl:       parseDepSpec("7.20.0|>=7.20.0,<7.21.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"5.5": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.59", false),
		Automake:   parseDepSpec("1.9.6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.16", false),
		Zlib:       parseDepSpec("1.3.1|>=1.2.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.0.2u|>=1.0.2,<1.0.3", false),
		Curl:       parseDepSpec("7.20.0|>=7.20.0,<7.21.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"5.4": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.59", false),
		Automake:   parseDepSpec("1.9.6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.16", false),
		Flex:       parseDepSpec("", true),
		Bison:      parseDepSpec("2.4.1", false),
		Zlib:       parseDepSpec("1.3.1|>=1.2.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.0.2u|>=1.0.2,<1.0.3", false),
		Curl:       parseDepSpec("7.20.0|>=7.20.0,<7.21.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"5.3": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.59", false),
		Automake:   parseDepSpec("1.9.6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.16", false),
		Flex:       parseDepSpec("2.6.4", false),
		Bison:      parseDepSpec("2.4.1", false),
		Zlib:       parseDepSpec("1.3.1|>=1.2.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.0.2u|>=1.0.2,<1.0.3", false),
		Curl:       parseDepSpec("7.20.0|>=7.20.0,<7.21.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"5.2": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.59", false),
		Automake:   parseDepSpec("1.9.6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("", false),
		Flex:       parseDepSpec("2.6.4", false),
		Bison:      parseDepSpec("2.4.1", false),
		Zlib:       parseDepSpec("1.3.1|>=1.2.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("1.0.2u|>=1.0.2,<1.0.3", false),
		Curl:       parseDepSpec("7.20.0|>=7.20.0,<7.21.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"5.1": {
		Perl:      parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:        parseDepSpec("1.4.19", false),
		Autoconf:  parseDepSpec("2.59", false),
		Automake:  parseDepSpec("1.9.6", false),
		Libtool:   parseDepSpec("1.5.26", false),
		Re2c:      parseDepSpec("", false),
		Flex:      parseDepSpec("", false),
		Bison:     parseDepSpec("2.4.1", false),
		Zlib:      parseDepSpec("1.3.1|>=1.2.0", false),
		Libxml2:   parseDepSpec("", false),
		OpenSSL:   parseDepSpec("1.0.2u|>=1.0.2,<1.0.3", false),
		Curl:      parseDepSpec("7.20.0|>=7.20.0,<7.21.0", false),
		Oniguruma: parseDepSpec("", false),
	},
	"5.0": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.59", false),
		Automake:   parseDepSpec("1.9.6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("", false),
		Flex:       parseDepSpec("2.6.4", false),
		Bison:      parseDepSpec("2.4.1", false),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("", false),
		Curl:       parseDepSpec("", false),
		Oniguruma:  parseDepSpec("", false),
	},
	"4.4": {
		Perl:      parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:        parseDepSpec("1.4.19", false),
		Autoconf:  parseDepSpec("2.13", false),
		Automake:  parseDepSpec("1.4-p6", false),
		Libtool:   parseDepSpec("1.5.26", false),
		Re2c:      parseDepSpec("0.14", false),
		Flex:      parseDepSpec("", true),
		Bison:     parseDepSpec("1.28", false),
		Zlib:      parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:   parseDepSpec("", false),
		OpenSSL:   parseDepSpec("", false),
		Curl:      parseDepSpec("7.88.1|>=7.80.0", false),
		Oniguruma: parseDepSpec("", false),
	},
	"4.3": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.13", false),
		Automake:   parseDepSpec("1.4-p6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.14", false),
		Flex:       parseDepSpec("", true),
		Bison:      parseDepSpec("", true),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("0.9.8zh|>=0.9.8,<1.0.0", false),
		Curl:       parseDepSpec("7.12.0|>=7.12.0,<7.13.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"4.2": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.13", false),
		Automake:   parseDepSpec("1.4-p6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.14", false),
		Flex:       parseDepSpec("", true),
		Bison:      parseDepSpec("", true),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("0.9.8zh|>=0.9.8,<1.0.0", false),
		Curl:       parseDepSpec("7.12.0|>=7.12.0,<7.13.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"4.1": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.13", false),
		Automake:   parseDepSpec("1.4-p6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.14", false),
		Flex:       parseDepSpec("", true),
		Bison:      parseDepSpec("", true),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("0.9.8zh|>=0.9.8,<1.0.0", false),
		Curl:       parseDepSpec("7.12.0|>=7.12.0,<7.13.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
	},
	"4.0": {
		Perl:       parseDepSpec("5.32.1|>=5.32.0,<5.33.0", false),
		M4:         parseDepSpec("1.4.19", false),
		Autoconf:   parseDepSpec("2.13", false),
		Automake:   parseDepSpec("1.4-p6", false),
		Libtool:    parseDepSpec("1.5.26", false),
		Re2c:       parseDepSpec("0.14", false),
		Flex:       parseDepSpec("", true),
		Bison:      parseDepSpec("", true),
		Zlib:       parseDepSpec("1.2.13|>=1.2.0,<1.3.0", false),
		Libxml2:    parseDepSpec("2.9.14|~2.9.0", false),
		Libxml2Dir: "2.9",
		OpenSSL:    parseDepSpec("0.9.8zh|>=0.9.8,<1.0.0", false),
		Curl:       parseDepSpec("7.12.0|>=7.12.0,<7.13.0", false),
		Oniguruma:  parseDepSpec("5.9.6|~5.9.0", false),
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
		newFlexDependency(config),
		newBisonDependency(config),
		newRe2cDependency(config),
		newZlibDependency(config),
		newLibxml2Dependency(config),
		newOpenSSLDependency(config),
		newCurlDependency(config),
		newOnigurumaDependency(config),
	}

	var filtered []domain.Dependency
	for _, dep := range deps {
		if dep.Name != "" {
			filtered = append(filtered, dep)
		}
	}

	return filtered
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
	if v.Major == 5 && v.Minor >= 3 {
		return versionRegistry["5.3"]
	}
	if v.Major == 4 {
		return versionRegistry["4.4"]
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

func getConfigureFlags(_ *DependencyURLConfig, _ string, override *DependencyPattern, defaults []string) []string {
	if override != nil && len(override.ConfigureFlags) > 0 {
		return override.ConfigureFlags
	}
	return defaults
}

func getBuildCommands(_ *DependencyURLConfig, _ string, override *DependencyPattern, defaults []string) []string {
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

func newCMakeDependency(_ PHPVersionConfig) domain.Dependency {
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
	version := config.Perl.GetRecommended()
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
	version := config.M4.GetRecommended()
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
	version := config.Autoconf.GetRecommended()
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
		Dependencies: []string{"m4", "perl"},
	}
}

func newAutomakeDependency(config PHPVersionConfig) domain.Dependency {
	version := config.Automake.GetRecommended()
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
	version := config.Libtool.GetRecommended()
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

func newFlexDependency(config PHPVersionConfig) domain.Dependency {
	// If Flex is not defined at all (empty constraint), skip it
	if config.Flex.ConstraintStr == "" {
		return domain.Dependency{}
	}
	version := config.Flex.GetRecommended()
	urlConfig := urlConfigs["flex"]
	override := config.FlexOverride

	return domain.Dependency{
		Name:        "flex",
		Version:     version,
		DownloadURL: getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, []string{
			"--disable-shared",
			"--enable-static",
		}),
		Dependencies: []string{"m4"},
	}
}

func newBisonDependency(config PHPVersionConfig) domain.Dependency {
	// If Bison is not defined at all (empty constraint), skip it
	if config.Bison.ConstraintStr == "" {
		return domain.Dependency{}
	}
	version := config.Bison.GetRecommended()
	urlConfig := urlConfigs["bison"]
	override := config.BisonOverride

	return domain.Dependency{
		Name:        "bison",
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
	// If Re2c is not defined at all (empty constraint), skip it
	if config.Re2c.ConstraintStr == "" {
		return domain.Dependency{}
	}
	version := config.Re2c.GetRecommended()
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
	version := config.Zlib.GetRecommended()
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
	// If Libxml2 is not defined at all (empty constraint), skip it
	if config.Libxml2.ConstraintStr == "" {
		return domain.Dependency{}
	}
	version := config.Libxml2.GetRecommended()
	dirVersion := config.Libxml2Dir
	urlConfig := urlConfigs["libxml2"]
	override := config.Libxml2Override

	url := getURL(&urlConfig, version, override)
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
	// If OpenSSL is not defined at all (empty constraint), skip it
	if config.OpenSSL.ConstraintStr == "" {
		return domain.Dependency{}
	}
	version := config.OpenSSL.GetRecommended()
	urlConfig := urlConfigs["openssl"]
	override := config.OpenSSLEverride

	flags := []string{
		"no-shared",
		"no-tests",
		"no-asm",
	}

	return domain.Dependency{
		Name:           "openssl",
		Version:        version,
		DownloadURL:    getURL(&urlConfig, version, override),
		ConfigureFlags: getConfigureFlags(&urlConfig, version, override, flags),
		BuildCommands:  getBuildCommands(&urlConfig, version, override, []string{"./config", "make", "build_libs"}),
		Dependencies:   []string{"perl"},
	}
}

func newCurlDependency(config PHPVersionConfig) domain.Dependency {
	// If Curl is not defined at all (empty constraint), skip it
	if config.Curl.ConstraintStr == "" {
		return domain.Dependency{}
	}
	version := config.Curl.GetRecommended()
	urlConfig := urlConfigs["curl"]
	override := config.CurlOverride

	var buildCommands []string
	if version >= "7.15" {
		buildCommands = []string{"./buildconf"}
	}

	return domain.Dependency{
		Name:          "curl",
		Version:       version,
		DownloadURL:   getURL(&urlConfig, version, override),
		BuildCommands: buildCommands,
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
	// If Oniguruma is not defined at all (empty constraint), skip it
	if config.Oniguruma.ConstraintStr == "" {
		return domain.Dependency{}
	}
	version := config.Oniguruma.GetRecommended()
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
