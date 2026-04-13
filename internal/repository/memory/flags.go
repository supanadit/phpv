package memory

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/extension"
	"github.com/supanadit/phpv/flagresolver"
	"github.com/supanadit/phpv/internal/utils"
)

func NewFlagRepository(extRepo extension.Repository) flagresolver.Repository {
	return &flagRepo{extRepo: extRepo}
}

type flagRepo struct {
	extRepo extension.Repository
}

var packageFlags = map[string][]string{
	"m4":        {"--disable-maintainer-mode"},
	"php":       {"--disable-all", "--enable-cli", "--with-openssl", "--with-curl", "--with-zlib", "--with-libxml2", "--with-onig"},
	"openssl":   {"shared", "no-ssl3"},
	"curl":      {"--with-openssl", "--without-brotli", "--disable-ldap"},
	"libxml2":   {"--disable-shared", "--enable-static", "--without-lzma", "--without-python", "--disable-dependency-tracking"},
	"zlib":      {},
	"oniguruma": {},
	"icu":       {},
	"re2c":      {},
	"autoconf":  {},
	"automake":  {},
	"libtool":   {},
	"flex":      {},
	"bison":     {},
	"perl":      {},
	"cmake":     {},
}

func (r *flagRepo) GetConfigureFlags(name string, version string) []string {
	flags, ok := packageFlags[name]
	if !ok {
		return []string{}
	}

	if name == "openssl" {
		result := make([]string, len(flags))
		copy(result, flags)
		if v := utils.ParseVersion(version); v.Major >= 3 {
			result = append(result, "no-legacy")
		}
		return result
	}

	return flags
}

func (r *flagRepo) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	flags := []string{
		"--disable-all",
		"--enable-cli",
	}

	if len(extensions) == 0 {
		return flags
	}

	v := utils.ParseVersion(phpVersion)

	for _, ext := range extensions {
		if extDef, ok := r.extRepo.GetExtensionDef(ext); ok {
			if r.extRepo.IsExtensionValidForPHPVersion(ext, phpVersion) {
				flags = append(flags, extDef.Flag)
			}
		}
	}

	if contains(extensions, "opcache") && v.Major >= 7 {
		flags = append(flags, "--enable-opcache")
	}

	return flags
}

func (r *flagRepo) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	return r.extRepo.GetExtensionDef(name)
}

func (r *flagRepo) IsExtensionValidForPHPVersion(name string, phpVersion string) bool {
	return r.extRepo.IsExtensionValidForPHPVersion(name, phpVersion)
}

func (r *flagRepo) GetConflictingExtensions(name string) []string {
	return r.extRepo.GetConflictingExtensions(name)
}

func (r *flagRepo) GetExtensionDependency(name string) (string, bool) {
	return r.extRepo.GetExtensionDependency(name)
}

func (r *flagRepo) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	return r.extRepo.GetExtensionDependencyWithVersion(extName, phpVersion)
}

func (r *flagRepo) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	return r.extRepo.ValidateExtensions(extensions, phpVersion)
}

func (r *flagRepo) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	return r.extRepo.CheckExtensionConflicts(extensions)
}
