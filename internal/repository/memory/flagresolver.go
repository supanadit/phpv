package memory

import (
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pattern"
)

func NewFlagResolverRepository() domain.FlagResolverRepository {
	return &flagResolverRepo{}
}

type flagResolverRepo struct{}

func (r *flagResolverRepo) GetConfigureFlags(name string) []string {
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
	}
	return nil
}

func (r *flagResolverRepo) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	v := pattern.ParseVersion(phpVersion)

	flags := []string{
		"--disable-all",
		"--enable-cli",
		"--with-openssl",
		"--with-curl",
		"--with-zlib",
		"--with-libxml2",
		"--with-onig",
	}

	if v.Major >= 8 {
		flags = append(flags, "--enable-opcache")
	}

	return flags
}
