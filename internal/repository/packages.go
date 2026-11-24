package repository

import "github.com/supanadit/phpv/domain"

// BuiltInPackages returns the hardcoded dependency definitions for all
// packages that phpv knows how to build from source.
func BuiltInPackages() []domain.Package {
	return append(packageLibraries(), packageToolchain()...)
}

func packageLibraries() []domain.Package {
	return []domain.Package{
		{
			Package:     "php",
			Default:     []domain.Dependency{},
			Constraints: nil, // deps are now derived from extension RequiresPackage+Versions
		},
		{
			Package: "openssl",
			Default: []domain.Dependency{
				{Name: "perl", Version: "5.38.2|>=5.32.0"},
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=3.0.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=1.1.0 <3.0.0",
					Dependencies: []domain.Dependency{
						{Name: "perl", Version: "5.38.2|>=5.32.0"},
						{Name: "m4", Version: "1.4.19|>=1.4.19"},
						{Name: "autoconf", Version: "2.71|>=2.71,<2.74"},
						{Name: "automake", Version: "1.16.5|>=1.16"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=1.0.0 <1.1.0",
					Dependencies: []domain.Dependency{
						{Name: "perl", Version: "5.38.2|>=5.32.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
				{
					VersionRange: ">=0.9.0 <1.0.0",
					Dependencies: []domain.Dependency{
						{Name: "perl", Version: "5.38.2|>=5.32.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.13"},
						{Name: "automake", Version: "1.4-p6"},
						{Name: "libtool", Version: "1.5.26"},
					},
				},
			},
		},
		{
			Package: "libxml2",
			Default: []domain.Dependency{
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=2.12.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=2.11.0 <2.12.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=2.9.0 <2.11.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
			},
		},
		{
			Package: "zlib",
			Default: []domain.Dependency{},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=1.3.0", Dependencies: []domain.Dependency{}},
				{VersionRange: ">=1.2.0 <1.3.0", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package: "libzip",
			Default: []domain.Dependency{
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=1.9.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.3.1|>=1.3.0"},
					},
				},
				{
					VersionRange: ">=1.7.0 <1.9.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
					},
				},
				{
					VersionRange: ">=0.11.0 <1.7.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
					},
				},
			},
		},
		{
			Package: "oniguruma",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=6.9.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=5.9.0 <6.9.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
			},
		},
		{
			Package: "curl",
			Default: []domain.Dependency{
				{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=8.0.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=7.80.0 <8.0.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=7.20.0 <7.80.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.0,<3.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
				{
					VersionRange: ">=7.12.0 <7.20.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.0,<3.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.59"},
						{Name: "automake", Version: "1.9.6"},
						{Name: "libtool", Version: "1.5.26"},
					},
				},
			},
		},
	}
}
