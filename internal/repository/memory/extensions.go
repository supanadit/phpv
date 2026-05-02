package memory

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/extension"
	"github.com/supanadit/phpv/internal/utils"
)

func NewExtensionRepository() extension.Repository {
	return &extensionRepo{}
}

type extensionRepo struct{}

var bundledExtensions = map[string]domain.ExtensionDef{
	"bcmath": {
		Flag:   "--enable-bcmath",
		MinPHP: "5.0",
	},
	"bz2": {
		Flag:    "--with-bz2",
		MinPHP:  "5.0",
		Package: "bzip2",
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
		Flag:    "--with-curl",
		MinPHP:  "5.0",
		Package: "curl",
		Versions: []domain.VersionConstraintDef{
			{VersionRange: ">=8.0.0", Version: "8.10.1|>=8.0.0"},
			{VersionRange: ">=7.0.0 <8.0.0", Version: "7.88.1|>=7.80.0"},
			{VersionRange: ">=5.6.0 <7.0.0", Version: "7.20.0|>=7.20.0,<7.21.0"},
			{VersionRange: ">=5.1.0 <5.6.0", Version: "7.12.1|>=7.12.0,<7.13.0"},
			{VersionRange: ">=5.0.0 <5.1.0", Version: "7.12.0|>=7.12.0,<7.13.0"},
		},
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
		Flag:    "--enable-intl",
		MinPHP:  "5.0",
		Package: "icu",
		Versions: []domain.VersionConstraintDef{
			{VersionRange: ">=5.0", Version: "74.2|>=74.2"},
		},
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
		Flag:    "--with-libxml",
		MinPHP:  "5.0",
		Package: "libxml2",
		Versions: []domain.VersionConstraintDef{
			{VersionRange: ">=8.2.0", Version: "2.12.7|~2.12.0"},
			{VersionRange: ">=8.0.0 <8.2.0", Version: "2.11.7|~2.11.0"},
			{VersionRange: ">=5.0.0 <8.0.0", Version: "2.9.14|~2.9.0"},
		},
	},
	"mbstring": {
		Flag:    "--enable-mbstring",
		MinPHP:  "5.0",
		Package: "oniguruma",
		Versions: []domain.VersionConstraintDef{
			{VersionRange: ">=8.0.0", Version: "6.9.9|~6.9.0"},
			{VersionRange: ">=7.4.0 <8.0.0", Version: "6.9.8|~6.9.0"},
			{VersionRange: ">=5.0.0 <7.4.0", Version: "5.9.6|~5.9.0"},
		},
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
		Implied:   []string{"zlib"},
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
		Flag:    "--with-openssl",
		MinPHP:  "5.0",
		Package: "openssl",
		Versions: []domain.VersionConstraintDef{
			{VersionRange: ">=8.2.0", Version: "3.3.2|>=3.0.0,<4.0.0"},
			{VersionRange: ">=7.0.0 <8.2.0", Version: "1.1.1w|>=1.1.0,<1.2.0"},
			{VersionRange: ">=5.0.0 <7.0.0", Version: "1.0.1u|>=1.0.0,<1.1.0"},
		},
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
		Implied:   []string{"zlib"},
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
		Flag:    "--with-pdo-pgsql",
		MinPHP:  "5.0",
		Package: "libpq",
	},
	"pdo_sqlite": {
		Flag:   "--with-pdo-sqlite",
		MinPHP: "5.0",
	},
	"pgsql": {
		Flag:    "--with-pgsql",
		MinPHP:  "5.0",
		Package: "libpq",
	},
	"phar": {
		Flag:    "--enable-phar",
		MinPHP:  "5.0",
		Implied: []string{"json"},
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
		Flag:    "--with-zlib",
		MinPHP:  "5.0",
		Package: "zlib",
		Versions: []domain.VersionConstraintDef{
			{VersionRange: ">=8.0.0", Version: "1.3.1|>=1.3.0"},
			{VersionRange: ">=5.0.0 <8.0.0", Version: "1.2.13|>=1.2.0,<1.3.0"},
		},
	},
}

func (r *extensionRepo) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	ext, ok := bundledExtensions[name]
	return ext, ok
}

func (r *extensionRepo) IsExtensionValidForPHPVersion(name string, phpVersion string) bool {
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

func (r *extensionRepo) GetConflictingExtensions(name string) []string {
	ext, ok := bundledExtensions[name]
	if !ok {
		return nil
	}
	return ext.Conflicts
}

func (r *extensionRepo) GetExtensionDependency(name string) (string, bool) {
	ext, ok := bundledExtensions[name]
	if !ok || ext.Package == "" {
		return "", false
	}
	return ext.Package, true
}

func (r *extensionRepo) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	ext, ok := bundledExtensions[extName]
	if !ok || ext.Package == "" {
		return "", "", false
	}

	for _, v := range ext.Versions {
		if utils.MatchVersionRange(v.VersionRange, phpVersion) {
			return ext.Package, v.Version, true
		}
	}

	return "", "", false
}

func (r *extensionRepo) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	var unknown []string
	for _, ext := range extensions {
		if _, ok := bundledExtensions[ext]; !ok {
			unknown = append(unknown, ext)
		} else if !r.IsExtensionValidForPHPVersion(ext, phpVersion) {
			unknown = append(unknown, ext)
		}
	}
	if len(unknown) > 0 {
		return unknown, nil
	}
	return nil, nil
}

func (r *extensionRepo) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	var conflicts []string
	var conflictPairs [][]string

	extSet := make(map[string]bool)
	for _, ext := range extensions {
		extSet[ext] = true
	}

	for _, ext := range extensions {
		conflictsList := r.GetConflictingExtensions(ext)
		for _, conflict := range conflictsList {
			if extSet[conflict] {
				conflictPairs = append(conflictPairs, []string{ext, conflict})
				if !contains(conflicts, ext) {
					conflicts = append(conflicts, ext)
				}
				if !contains(conflicts, conflict) {
					conflicts = append(conflicts, conflict)
				}
			}
		}
	}

	return conflicts, conflictPairs
}

func (r *extensionRepo) ListExtensions() []domain.ExtensionInfo {
	extensions := make([]domain.ExtensionInfo, 0, len(bundledExtensions))
	for name, ext := range bundledExtensions {
		extensions = append(extensions, domain.ExtensionInfo{
			Name:        name,
			Flag:        ext.Flag,
			MinPHP:      ext.MinPHP,
			MaxPHP:      ext.MaxPHP,
			Package:     ext.Package,
			HasConflict: len(ext.Conflicts) > 0,
			Conflicts:   ext.Conflicts,
		})
	}
	return extensions
}

func (r *extensionRepo) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	extensions := make([]domain.ExtensionInfo, 0, len(bundledExtensions))
	for name, ext := range bundledExtensions {
		if !r.IsExtensionValidForPHPVersion(name, phpVersion) {
			continue
		}
		extensions = append(extensions, domain.ExtensionInfo{
			Name:        name,
			Flag:        ext.Flag,
			MinPHP:      ext.MinPHP,
			MaxPHP:      ext.MaxPHP,
			Package:     ext.Package,
			HasConflict: len(ext.Conflicts) > 0,
			Conflicts:   ext.Conflicts,
		})
	}
	return extensions
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
