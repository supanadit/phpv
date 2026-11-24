package repository

import "github.com/supanadit/phpv/domain"

func BuiltInExtensions() []domain.ExtensionDef {
	return append(builtInExtensionDefs(), builtInExtensionDefs2()...)
}

func builtInExtensionDefs() []domain.ExtensionDef {
	return []domain.ExtensionDef{
		{
			Name:          "bcmath",
			Description:   "BC Math arbitrary precision",
			Flag:          "--enable-bcmath",
			MinPHPVersion: "5.0",
		},
		{
			Name:            "bz2",
			Description:     "BZip2 compression",
			Flag:            "--with-bz2",
			MinPHPVersion:   "5.0",
			RequiresPackage: "bzip2",
		},
		{
			Name:          "calendar",
			Description:   "Calendar conversion support",
			Flag:          "--enable-calendar",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "ctype",
			Description:   "Character type checking",
			Flag:          "--enable-ctype",
			MinPHPVersion: "5.0",
		},
		{
			Name:            "curl",
			Description:     "cURL support",
			Flag:            "--with-curl",
			MinPHPVersion:   "5.0",
			RequiresPackage: "curl",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0.0", Version: "8.10.1|>=8.0.0"},
				{VersionRange: ">=7.0.0 <8.0.0", Version: "7.88.1|>=7.80.0"},
				{VersionRange: ">=5.6.0 <7.0.0", Version: "7.20.0|>=7.20.0,<7.21.0"},
				// curl 7.12.1 was never published on curl.se (not even in the
				// archeology archive); 7.12.0 is the nearest available release.
				{VersionRange: ">=5.1.0 <5.6.0", Version: "7.12.0|>=7.12.0,<7.13.0"},
				{VersionRange: ">=5.0.0 <5.1.0", Version: "7.12.0|>=7.12.0,<7.13.0"},
			},
		},
		{
			Name:          "dba",
			Description:   "Database (dbm-style) abstraction layer",
			Flag:          "--enable-dba",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "dom",
			Description:   "DOM support",
			Flag:          "--enable-dom",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "enchant",
			Description:   "Enchant spelling library",
			Flag:          "--with-enchant",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "exif",
			Description:   "EXIF headers",
			Flag:          "--enable-exif",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "fileinfo",
			Description:   "File information",
			Flag:          "--enable-fileinfo",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "filter",
			Description:   "Data filtering",
			Flag:          "--enable-filter",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "ftp",
			Description:   "FTP support",
			Flag:          "--enable-ftp",
			MinPHPVersion: "5.0",
		},
		{
			Name:             "gd",
			Description:      "GD image processing",
			Flag:             "--with-gd",
			MinPHPVersion:    "5.0",
			RequiresPackages: []string{"libpng", "libjpeg", "freetype"},
			FlagVersions: []domain.FlagVersionDef{
				{VersionRange: ">=7.4", Flag: "--enable-gd"},
			},
		},
		{
			Name:          "gettext",
			Description:   "Gettext support",
			Flag:          "--with-gettext",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "gmp",
			Description:   "GNU MP support",
			Flag:          "--with-gmp",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "hash",
			Description:   "HASH message digest",
			Flag:          "--enable-hash",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "iconv",
			Description:   "Iconv support",
			MinPHPVersion: "5.0",
			FlagVersions: []domain.FlagVersionDef{
				// PHP < 8.5: iconv is a configure-time extension (--with-iconv).
				{VersionRange: ">=5.0 <8.5.0", Flag: "--with-iconv"},
				// PHP >= 8.5: iconv is NOT compiled into the binary. It ships as a
				// shared .so in the PHP source tree (ext/iconv/) and must be built
				// with phpize after the main PHP build. Flag: "" means no configure
				// flag is needed; IsBuiltIn: false means it's shared-only.
				//
				// IMPORTANT: IsBuiltIn: false + Flag: "" = shared-only (needs phpize).
				// Do NOT set IsBuiltIn: true here unless iconv is actually compiled
				// into the PHP binary for this version.
				{VersionRange: ">=8.5.0", Flag: "", IsBuiltIn: false},
			},
		},
		{
			Name:          "imap",
			Description:   "IMAP support",
			Flag:          "--with-imap",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "interbase",
			Description:   "InterBase support",
			Flag:          "--with-interbase",
			MinPHPVersion: "5.0",
		},
		{
			Name:             "intl",
			Description:      "Internationalization support",
			Flag:             "--enable-intl",
			MinPHPVersion:    "5.0",
			RequiresPackage:  "icu",
			RequiresPackages: []string{"icu"},
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0", Version: "74.2|>=74.2"},
				{VersionRange: ">=7.4", Version: "63.1|>=63.1,<74"},
				{VersionRange: ">=5.0 <7.4", Version: "58.2|>=58.0,<60"},
			},
		},
		{
			Name:          "json",
			Description:   "JSON support",
			Flag:          "--enable-json",
			MinPHPVersion: "5.2",
		},
		{
			Name:          "ldap",
			Description:   "LDAP support",
			Flag:          "--with-ldap",
			MinPHPVersion: "5.0",
		},
		{
			Name:             "libxml",
			Description:      "LIBXML support",
			Flag:             "--enable-libxml",
			MinPHPVersion:    "5.0",
			RequiresPackage:  "libxml2",
			RequiresPackages: []string{"libxml2"},
			FlagVersions: []domain.FlagVersionDef{
				{VersionRange: ">=7.4", Flag: "--with-libxml"},
			},
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.2.0", Version: "2.12.7|~2.12.0"},
				{VersionRange: ">=8.0.0 <8.2.0", Version: "2.11.7|~2.11.0"},
				{VersionRange: ">=5.0.0 <8.0.0", Version: "2.9.14|~2.9.0"},
			},
		},
		{
			Name:            "mbstring",
			Description:     "Multibyte String support",
			Flag:            "--enable-mbstring",
			MinPHPVersion:   "5.0",
			RequiresPackage: "oniguruma",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0.0", Version: "6.9.9|~6.9.0"},
				{VersionRange: ">=7.4.0 <8.0.0", Version: "6.9.8|~6.9.0"},
				{VersionRange: ">=5.0.0 <7.4.0", Version: "5.9.6|~5.9.0"},
			},
		},
		{
			Name:          "mysql",
			Description:   "MySQL support (deprecated)",
			Flag:          "--with-mysql",
			MinPHPVersion: "5.0",
			MaxPHPVersion: "5.6",
			Conflicts:     []string{"mysqli", "pdo_mysql"},
			Implied:       []string{"zlib"},
		},
		{
			Name:          "mysqli",
			Description:   "MySQL Improved Extension",
			Flag:          "--with-mysqli",
			MinPHPVersion: "5.0",
			Conflicts:     []string{"mysql", "pdo_mysql"},
			Implied:       []string{"zlib"},
		},
		{
			Name:          "odbc",
			Description:   "ODBC support",
			Flag:          "--with-odbc",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "opcache",
			Description:   "OPcache",
			Flag:          "--enable-opcache",
			MinPHPVersion: "7.0",
		},
		{
			Name:            "openssl",
			Description:     "OpenSSL support",
			Flag:            "--with-openssl",
			MinPHPVersion:   "5.0",
			RequiresPackage: "openssl",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.4.0", Version: "1.1.1w|>=1.1.1,<4.0.0"},
				{VersionRange: ">=8.1.0 <8.4.0", Version: "1.1.1w|>=1.0.2,<4.0.0"},
				{VersionRange: ">=7.1.0 <8.1.0", Version: "1.1.1w|>=1.1.1,<1.3.0"},
				{VersionRange: ">=7.0.0 <7.1.0", Version: "1.1.1w|>=1.1.1,<1.3.0"},
				{VersionRange: ">=5.0.0 <7.0.0", Version: "1.0.1u|>=1.0.0,<1.1.0"},
			},
		},
		{
			Name:          "pcntl",
			Description:   "PCNTL process control",
			Flag:          "--enable-pcntl",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pcre",
			Description:   "PCRE regex support",
			Flag:          "--with-pcre-regex",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pdo",
			Description:   "PHP Data Objects",
			Flag:          "--enable-pdo",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pdo_dblib",
			Description:   "Sybase driver for PDO",
			Flag:          "--with-pdo-dblib",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:          "pdo_firebird",
			Description:   "Firebird driver for PDO",
			Flag:          "--with-pdo-firebird",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
	}
}
