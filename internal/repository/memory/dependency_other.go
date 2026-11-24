package memory

import "github.com/supanadit/phpv/domain"

func packageToolchain() []domain.Package {
	return []domain.Package{
		{
			Package: "autoconf",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
			},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=2.69", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package: "automake",
			Default: []domain.Dependency{
				{Name: "autoconf", Version: "2.69"},
				{Name: "m4", Version: "1.4.19"},
			},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=1.15", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package: "libtool",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.71"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=2.5.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
					},
				},
				{
					VersionRange: ">=1.5.0 <2.5.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
					},
				},
			},
		},
		{
			Package: "m4",
			Default: []domain.Dependency{},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=1.4.19", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package:     "perl",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package: "flex",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package: "bison",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
			},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=3.0", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package:     "icu",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "zig",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			// nginx links PCRE2 for rewrite support. The system libpcre2
			// satisfies it when present; otherwise the assembler builds the
			// pinned version from source.
			Package: "pcre2",
			Default: []domain.Dependency{
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package: "nginx",
			Default: []domain.Dependency{
				{Name: "pcre2", Version: "10.44|>=10.42"},
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			// httpd vendors apr/apr-util into its source tree at configure
			// time (--with-included-apr): they are downloaded so the
			// assembler can copy them into srclib/, but never built
			// standalone. pcre2 IS built standalone (httpd invokes
			// pcre2-config from the install prefix).
			Package: "httpd",
			Default: []domain.Dependency{
				{Name: "apr", Version: "1.7.4"},
				{Name: "apr-util", Version: "1.6.3"},
				// apr-util hard-requires expat (its configure refuses
				// --without-expat), consumed as an installed prefix.
				{Name: "expat", Version: "2.6.3|>=2.6.0"},
				{Name: "pcre2", Version: "10.44|>=10.42"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "apr",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "apr-util",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "expat",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "caddy",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
	}
}
