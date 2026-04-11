package flagresolver

import (
	"errors"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

var ErrUnknownExtension = errors.New("unknown extension")
var ErrExtensionConflict = errors.New("extension conflict")

type Repository interface {
	GetExtensionDef(name string) (domain.ExtensionDef, bool)
	IsExtensionValidForPHPVersion(name string, phpVersion string) bool
	GetConflictingExtensions(name string) []string
	GetExtensionDependency(name string) (string, bool)
	GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool)
	ValidateExtensions(extensions []string, phpVersion string) ([]string, error)
	CheckExtensionConflicts(extensions []string) ([]string, [][]string)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetConfigureFlags(name string, version string) []string {
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

func (s *Service) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	flags := []string{
		"--disable-all",
		"--enable-cli",
	}

	if len(extensions) == 0 {
		return flags
	}

	v := utils.ParseVersion(phpVersion)

	for _, ext := range extensions {
		if extDef, ok := s.repo.GetExtensionDef(ext); ok {
			if s.repo.IsExtensionValidForPHPVersion(ext, phpVersion) {
				flags = append(flags, extDef.Flag)
			}
		}
	}

	if contains(extensions, "opcache") && v.Major >= 7 {
		flags = append(flags, "--enable-opcache")
	}

	return flags
}

func (s *Service) ValidateExtensions(extensions []string, phpVersion string) error {
	unknown, err := s.repo.ValidateExtensions(extensions, phpVersion)
	if err != nil {
		return err
	}
	if len(unknown) > 0 {
		return ErrUnknownExtension
	}
	return nil
}

func (s *Service) CheckExtensionConflicts(extensions []string) ([]string, [][]string, error) {
	conflicts, conflictPairs := s.repo.CheckExtensionConflicts(extensions)
	if len(conflicts) > 0 {
		return conflicts, conflictPairs, ErrExtensionConflict
	}
	return nil, nil, nil
}

func (s *Service) GetExtensionDependency(ext string) (string, bool) {
	return s.repo.GetExtensionDependency(ext)
}

func (s *Service) GetExtensionDependencyWithVersion(ext, phpVersion string) (string, string, bool) {
	return s.repo.GetExtensionDependencyWithVersion(ext, phpVersion)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
