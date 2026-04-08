package memory

import (
	"github.com/supanadit/phpv/internal/utils"
)

type extensionDef struct {
	Flag      string
	MinPHP    string
	MaxPHP    string
	Conflicts []string
}

var bundledExtensions = map[string]extensionDef{
	"bcmath": {
		Flag:   "--enable-bcmath",
		MinPHP: "5.0",
	},
	"bz2": {
		Flag:   "--with-bz2",
		MinPHP: "5.0",
	},
	"calendar": {
		Flag:   "--enable-calendar",
		MinPHP: "5.0",
	},
	"ctype": {
		Flag:   "--enable-ctype",
		MinPHP: "5.0",
	},
	"curl": {
		Flag:   "--with-curl",
		MinPHP: "5.0",
	},
	"dba": {
		Flag:   "--enable-dba",
		MinPHP: "5.0",
	},
	"dom": {
		Flag:   "--enable-dom",
		MinPHP: "5.0",
	},
	"enchant": {
		Flag:   "--with-enchant",
		MinPHP: "5.0",
	},
	"exif": {
		Flag:   "--enable-exif",
		MinPHP: "5.0",
	},
	"fileinfo": {
		Flag:   "--enable-fileinfo",
		MinPHP: "5.0",
	},
	"filter": {
		Flag:   "--enable-filter",
		MinPHP: "5.0",
	},
	"ftp": {
		Flag:   "--enable-ftp",
		MinPHP: "5.0",
	},
	"gd": {
		Flag:   "--with-gd",
		MinPHP: "5.0",
	},
	"gettext": {
		Flag:   "--with-gettext",
		MinPHP: "5.0",
	},
	"gmp": {
		Flag:   "--with-gmp",
		MinPHP: "5.0",
	},
	"hash": {
		Flag:   "--enable-hash",
		MinPHP: "5.0",
	},
	"iconv": {
		Flag:   "--with-iconv",
		MinPHP: "5.0",
	},
	"imap": {
		Flag:   "--with-imap",
		MinPHP: "5.0",
	},
	"interbase": {
		Flag:   "--with-interbase",
		MinPHP: "5.0",
	},
	"intl": {
		Flag:   "--enable-intl",
		MinPHP: "5.0",
	},
	"json": {
		Flag:   "--enable-json",
		MinPHP: "5.2",
	},
	"ldap": {
		Flag:   "--with-ldap",
		MinPHP: "5.0",
	},
	"libxml": {
		Flag:   "--with-libxml",
		MinPHP: "5.0",
	},
	"mbstring": {
		Flag:   "--enable-mbstring",
		MinPHP: "5.0",
	},
	"mysql": {
		Flag:      "--with-mysql",
		MinPHP:    "5.0",
		MaxPHP:    "5.6",
		Conflicts: []string{"mysqli", "pdo_mysql"},
	},
	"mysqli": {
		Flag:      "--with-mysqli",
		MinPHP:    "5.0",
		Conflicts: []string{"mysql", "pdo_mysql"},
	},
	"odbc": {
		Flag:   "--with-odbc",
		MinPHP: "5.0",
	},
	"opcache": {
		Flag:   "--enable-opcache",
		MinPHP: "7.0",
	},
	"openssl": {
		Flag:   "--with-openssl",
		MinPHP: "5.0",
	},
	"pcntl": {
		Flag:   "--enable-pcntl",
		MinPHP: "5.0",
	},
	"pcre": {
		Flag:   "--with-pcre-regex",
		MinPHP: "5.0",
	},
	"pdo": {
		Flag:   "--enable-pdo",
		MinPHP: "5.0",
	},
	"pdo_dblib": {
		Flag:   "--with-pdo-dblib",
		MinPHP: "5.0",
	},
	"pdo_firebird": {
		Flag:   "--with-pdo-firebird",
		MinPHP: "5.0",
	},
	"pdo_mysql": {
		Flag:      "--with-pdo-mysql",
		MinPHP:    "5.0",
		Conflicts: []string{"mysql", "mysqli"},
	},
	"pdo_oci": {
		Flag:   "--with-pdo-oci",
		MinPHP: "5.0",
	},
	"pdo_odbc": {
		Flag:   "--with-pdo-odbc",
		MinPHP: "5.0",
	},
	"pdo_pgsql": {
		Flag:   "--with-pdo-pgsql",
		MinPHP: "5.0",
	},
	"pdo_sqlite": {
		Flag:   "--with-pdo-sqlite",
		MinPHP: "5.0",
	},
	"pgsql": {
		Flag:   "--with-pgsql",
		MinPHP: "5.0",
	},
	"phar": {
		Flag:   "--enable-phar",
		MinPHP: "5.0",
	},
	"posix": {
		Flag:   "--enable-posix",
		MinPHP: "5.0",
	},
	"pspell": {
		Flag:   "--with-pspell",
		MinPHP: "5.0",
	},
	"random": {
		Flag:   "--with-random",
		MinPHP: "7.0",
	},
	"readline": {
		Flag:   "--with-readline",
		MinPHP: "5.0",
	},
	"recode": {
		Flag:   "--with-recode",
		MinPHP: "5.0",
	},
	"session": {
		Flag:   "--enable-session",
		MinPHP: "5.0",
	},
	"shmop": {
		Flag:   "--enable-shmop",
		MinPHP: "5.0",
	},
	"simplexml": {
		Flag:   "--enable-simplexml",
		MinPHP: "5.0",
	},
	"snmp": {
		Flag:   "--with-snmp",
		MinPHP: "5.0",
	},
	"soap": {
		Flag:   "--enable-soap",
		MinPHP: "5.0",
	},
	"sockets": {
		Flag:   "--enable-sockets",
		MinPHP: "5.0",
	},
	"sodium": {
		Flag:   "--with-sodium",
		MinPHP: "7.2",
	},
	"sqlite3": {
		Flag:   "--enable-sqlite3",
		MinPHP: "5.0",
	},
	"standard": {
		Flag:   "--enable-standard",
		MinPHP: "5.0",
	},
	"sysvmsg": {
		Flag:   "--enable-sysvmsg",
		MinPHP: "5.0",
	},
	"sysvsem": {
		Flag:   "--enable-sysvsem",
		MinPHP: "5.0",
	},
	"sysvshm": {
		Flag:   "--enable-sysvshm",
		MinPHP: "5.0",
	},
	"tidy": {
		Flag:   "--with-tidy",
		MinPHP: "5.0",
	},
	"tokenizer": {
		Flag:   "--enable-tokenizer",
		MinPHP: "5.0",
	},
	"tokenizer_all": {
		Flag:   "--enable-tokenizer-all",
		MinPHP: "7.0",
	},
	"xml": {
		Flag:   "--enable-xml",
		MinPHP: "5.0",
	},
	"xmlreader": {
		Flag:   "--enable-xmlreader",
		MinPHP: "5.0",
	},
	"xmlrpc": {
		Flag:   "--enable-xmlrpc",
		MinPHP: "5.0",
	},
	"xmlwriter": {
		Flag:   "--enable-xmlwriter",
		MinPHP: "5.0",
	},
	"xsl": {
		Flag:   "--with-xsl",
		MinPHP: "5.0",
	},
	"zend_test": {
		Flag:   "--enable-zend-test",
		MinPHP: "7.0",
	},
	"zip": {
		Flag:   "--enable-zip",
		MinPHP: "5.0",
	},
	"zlib": {
		Flag:   "--with-zlib",
		MinPHP: "5.0",
	},
}

func GetBundledExtensionDef(name string) (extensionDef, bool) {
	ext, ok := bundledExtensions[name]
	return ext, ok
}

func IsExtensionValidForPHPVersion(name string, phpVersion string) bool {
	ext, ok := bundledExtensions[name]
	if !ok {
		return false
	}

	phpVer := utils.ParseVersion(phpVersion)

	if ext.MinPHP != "" {
		minVer := utils.ParseVersion(ext.MinPHP)
		if utils.CompareVersions(phpVer, minVer) < 0 {
			return false
		}
	}

	if ext.MaxPHP != "" {
		maxVer := utils.ParseVersion(ext.MaxPHP)
		if utils.CompareVersions(phpVer, maxVer) > 0 {
			return false
		}
	}

	return true
}

func GetConflictingExtensions(name string) []string {
	ext, ok := bundledExtensions[name]
	if !ok {
		return nil
	}
	return ext.Conflicts
}
