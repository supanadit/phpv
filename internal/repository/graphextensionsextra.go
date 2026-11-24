package repository

import "github.com/supanadit/phpv/domain"

func builtInExtensionDefs2() []domain.ExtensionDef {
	return []domain.ExtensionDef{
		{
			Name:          "pdo_mysql",
			Description:   "MySQL driver for PDO",
			Flag:          "--with-pdo-mysql=mysqlnd",
			MinPHPVersion: "5.0",
			Conflicts:     []string{"mysql", "mysqli"},
			Implied:       []string{"pdo", "zlib"},
		},
		{
			Name:          "pdo_oci",
			Description:   "Oracle driver for PDO",
			Flag:          "--with-pdo-oci",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:          "pdo_odbc",
			Description:   "ODBC driver for PDO",
			Flag:          "--with-pdo-odbc",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:             "pdo_pgsql",
			Description:      "PostgreSQL driver for PDO",
			Flag:             "--with-pdo-pgsql",
			MinPHPVersion:    "5.0",
			RequiresPackage:  "libpq",
			RequiresPackages: []string{"libpq"},
			Implied:          []string{"pdo"},
		},
		{
			Name:          "pdo_sqlite",
			Description:   "SQLite driver for PDO",
			Flag:          "--with-pdo-sqlite",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:             "pgsql",
			Description:      "PostgreSQL support",
			Flag:             "--with-pgsql",
			MinPHPVersion:    "5.0",
			RequiresPackage:  "libpq",
			RequiresPackages: []string{"libpq"},
		},
		{
			Name:          "phar",
			Description:   "Phar archive support",
			Flag:          "--enable-phar",
			MinPHPVersion: "5.0",
			Implied:       []string{"json", "hash"},
		},
		{
			Name:          "posix",
			Description:   "POSIX system calls",
			Flag:          "--enable-posix",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pspell",
			Description:   "PSpell support",
			Flag:          "--with-pspell",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "random",
			Description:   "Random number generator",
			Flag:          "--with-random",
			MinPHPVersion: "7.0",
		},
		{
			Name:          "readline",
			Description:   "Readline support",
			Flag:          "--with-readline",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "recode",
			Description:   "Recode support",
			Flag:          "--with-recode",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "session",
			Description:   "Session support",
			Flag:          "--enable-session",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "shmop",
			Description:   "Shared memory operations",
			Flag:          "--enable-shmop",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "simplexml",
			Description:   "SimpleXML support",
			Flag:          "--enable-simplexml",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "snmp",
			Description:   "SNMP support",
			Flag:          "--with-snmp",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "soap",
			Description:   "SOAP support",
			Flag:          "--enable-soap",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "sockets",
			Description:   "Socket support",
			Flag:          "--enable-sockets",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sodium",
			Description:   "Sodium support",
			Flag:          "--with-sodium",
			MinPHPVersion: "7.2",
		},
		{
			Name:          "sqlite3",
			Description:   "SQLite3 support",
			Flag:          "--with-sqlite3",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "standard",
			Description:   "Standard PHP functions",
			Flag:          "--enable-standard",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sysvmsg",
			Description:   "System V message queues",
			Flag:          "--enable-sysvmsg",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sysvsem",
			Description:   "System V semaphores",
			Flag:          "--enable-sysvsem",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sysvshm",
			Description:   "System V shared memory",
			Flag:          "--enable-sysvshm",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "tidy",
			Description:   "Tidy HTML cleaner",
			Flag:          "--with-tidy",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "tokenizer",
			Description:   "Tokenizer support",
			Flag:          "--enable-tokenizer",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "tokenizer_all",
			Description:   "Tokenizer all tokens",
			Flag:          "--enable-tokenizer-all",
			MinPHPVersion: "7.0",
		},
		{
			Name:          "xml",
			Description:   "XML support",
			Flag:          "--enable-xml",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xmlreader",
			Description:   "XMLReader support",
			Flag:          "--enable-xmlreader",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xmlrpc",
			Description:   "XML-RPC support",
			Flag:          "--enable-xmlrpc",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xmlwriter",
			Description:   "XMLWriter support",
			Flag:          "--enable-xmlwriter",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xsl",
			Description:   "XSL support",
			Flag:          "--with-xsl",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "zend_test",
			Description:   "Zend test extension",
			Flag:          "--enable-zend-test",
			MinPHPVersion: "7.0",
		},
		{
			Name:             "zip",
			Description:      "ZIP archive support",
			Flag:             "--enable-zip",
			MinPHPVersion:    "5.0",
			RequiresPackage:  "libzip",
			RequiresPackages: []string{"libzip"},
			FlagVersions: []domain.FlagVersionDef{
				{VersionRange: ">=7.4", Flag: "--with-zip"},
			},
			// PHP >= 7.4 no longer bundles libzip: the zip extension is
			// built against the system libzip (pkg-config "libzip >= 0.11").
			// Without this dependency, `phpv install 7.4` fails at configure
			// time with "Package libzip was not found in the pkg-config
			// search path" on systems that do not ship libzip-devel.
			Versions: []domain.VersionConstraintDef{
				// libzip >= 1.8 removed APIs used by PHP < 8.1 (zip_source_file
				// with start/len args), so PHP < 8.1 must use libzip 1.7.3.
				{VersionRange: ">=8.1.0", Version: "1.11.3|>=1.7.3"},
				{VersionRange: ">=7.4.0 <8.1.0", Version: "1.7.3|>=0.11.0,<1.8.0"},
			},
		},
		{
			Name:            "zlib",
			Description:     "Zlib compression",
			Flag:            "--with-zlib",
			MinPHPVersion:   "5.0",
			RequiresPackage: "zlib",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0.0", Version: "1.3.1|>=1.3.0"},
				{VersionRange: ">=5.0.0 <8.0.0", Version: "1.2.13|>=1.2.0,<1.3.0"},
			},
		},
	}
}
