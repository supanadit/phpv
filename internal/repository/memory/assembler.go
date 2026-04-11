package memory

import (
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
)

func NewMemoryAssemblerRepository() assembler.AssemblerRepository {
	svc := assembler.NewAssemblerService()
	registerPackages(svc)
	return svc
}

func registerPackages(svc *assembler.AssemblerService) {
	packages := []domain.Package{
		{
			Package: "php",
			Default: []domain.Dependency{},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=8.2.0",
					Dependencies: []domain.Dependency{},
				},
				{
					VersionRange: ">=5.0.0 <8.2.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.0,<3.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "6.9.9|~6.9.0"},
						{Name: "curl", Version: "8.5.0|>=7.80.0"},
					},
				},
				{
					VersionRange: ">=4.4.0 <5.0.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.0.1u|>=1.0.0,<1.1.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.88.1|>=7.80.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
				{
					VersionRange: ">=4.3.0 <4.4.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "0.9.8zh|>=0.9.8,<1.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.12.0|>=7.12.0,<7.13.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
				{
					VersionRange: ">=4.2.0 <4.3.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "0.9.8zh|>=0.9.8,<1.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.12.0|>=7.12.0,<7.13.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
				{
					VersionRange: ">=4.1.0 <4.2.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "0.9.8zh|>=0.9.8,<1.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.12.0|>=7.12.0,<7.13.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
				{
					VersionRange: ">=4.0.0 <4.1.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "0.9.8zh|>=0.9.8,<1.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.12.0|>=7.12.0,<7.13.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
			},
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
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.4.7"},
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
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=2.12.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=2.11.0 <2.12.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.4.7"},
					},
				},
				{
					VersionRange: ">=2.9.0 <2.11.0",
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
			Package: "zlib",
			Default: []domain.Dependency{},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=1.3.0",
					Dependencies: []domain.Dependency{},
				},
				{
					VersionRange: ">=1.2.0 <1.3.0",
					Dependencies: []domain.Dependency{},
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
						{Name: "libtool", Version: "2.4.7"},
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
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=8.0.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=7.80.0 <8.0.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.4.7"},
					},
				},
				{
					VersionRange: ">=7.20.0 <7.80.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
				{
					VersionRange: ">=7.12.0 <7.20.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.59"},
						{Name: "automake", Version: "1.9.6"},
						{Name: "libtool", Version: "1.5.26"},
					},
				},
			},
		},
		{
			Package: "re2c",
			Default: []domain.Dependency{
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
					VersionRange: ">=2.0.0 <3.0.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.4.7"},
					},
				},
				{
					VersionRange: ">=1.3.0 <2.0.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
				{
					VersionRange: ">=0.14.0 <1.3.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.59"},
						{Name: "automake", Version: "1.9.6"},
						{Name: "libtool", Version: "1.5.26"},
					},
				},
			},
		},
		{
			Package: "autoconf",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package: "automake",
			Default: []domain.Dependency{
				{Name: "autoconf", Version: "2.69"},
				{Name: "m4", Version: "1.4.19"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package: "libtool",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "m4",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
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
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "zig",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
	}

	for _, pkg := range packages {
		svc.RegisterPackage(pkg)
	}
}
