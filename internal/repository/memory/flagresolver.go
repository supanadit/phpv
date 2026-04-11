package memory

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func NewFlagResolverRepository() domain.FlagResolverRepository {
	return &flagResolverRepo{}
}

type flagResolverRepo struct{}

func (r *flagResolverRepo) GetConfigureFlags(name string, version string) []string {
	switch name {
	case "m4":
		return []string{"--disable-maintainer-mode"}
	case "php":
		return []string{
			"--disable-all",
			"--enable-cli",
			"--with-openssl",
			"--with-curl",
			"--with-zlib",
			"--with-libxml2",
			"--with-onig",
		}
	case "openssl":
		flags := []string{"shared", "no-ssl3"}
		if v := utils.ParseVersion(version); v.Major >= 3 {
			flags = append(flags, "no-legacy")
		}
		return flags
	case "curl":
		return []string{"--with-openssl", "--without-brotli", "--disable-ldap"}
	case "libxml2":
		return []string{"--disable-shared", "--enable-static", "--without-lzma", "--without-python"}
	case "zlib", "oniguruma", "re2c", "autoconf", "automake", "libtool", "flex", "bison", "perl", "cmake":
		return []string{}
	}
	return []string{}
}

func (r *flagResolverRepo) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	flags := []string{
		"--disable-all",
		"--enable-cli",
	}

	if len(extensions) == 0 {
		return flags
	}

	v := utils.ParseVersion(phpVersion)
	extSet := make(map[string]bool)
	for _, ext := range extensions {
		extSet[ext] = true
	}

	for _, ext := range extensions {
		if extFlags, ok := GetBundledExtensionDef(ext); ok {
			if IsExtensionValidForPHPVersion(ext, phpVersion) {
				flags = append(flags, extFlags.Flag)
			}
		}
	}

	if extSet["opcache"] && v.Major >= 7 {
		flags = append(flags, "--enable-opcache")
	}

	return flags
}

func (r *flagResolverRepo) GetExtensionFlag(name string) (string, bool) {
	ext, ok := GetBundledExtensionDef(name)
	if !ok {
		return "", false
	}
	return ext.Flag, true
}

func (r *flagResolverRepo) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	var unknown []string
	for _, ext := range extensions {
		if _, ok := GetBundledExtensionDef(ext); !ok {
			unknown = append(unknown, ext)
		} else if !IsExtensionValidForPHPVersion(ext, phpVersion) {
			unknown = append(unknown, ext)
		}
	}
	if len(unknown) > 0 {
		return unknown, &domain.UnknownExtensionError{Extension: unknown[0]}
	}
	return nil, nil
}

func (r *flagResolverRepo) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	var conflicts []string
	var conflictPairs [][]string

	extSet := make(map[string]bool)
	for _, ext := range extensions {
		extSet[ext] = true
	}

	for _, ext := range extensions {
		conflictsList := GetConflictingExtensions(ext)
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

func (r *flagResolverRepo) GetExtensionDependency(ext string) (string, bool) {
	return GetExtensionDependency(ext)
}

func (r *flagResolverRepo) GetExtensionDependencyWithVersion(ext, phpVersion string) (string, string, bool) {
	return GetExtensionDependencyWithVersion(ext, phpVersion)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
