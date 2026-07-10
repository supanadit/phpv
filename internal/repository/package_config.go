package repository

import (
	"strings"

	"github.com/supanadit/phpv/domain"
)

// ExtOverride defines an extension override for versions older than Before.
// Overrides are evaluated in order — first match wins.
type ExtOverride struct {
	Before string
	Ext    string
}

// ExtensionConfig defines the default extension and optional overrides.
// Overrides are evaluated in order — first match wins. If no override matches,
// Default is used.
//
// Example:
//
//	ExtensionConfig{
//	    Default: "tar.gz",
//	    Override: []ExtOverride{
//	        {Before: "5.20.0", Ext: "tar.bz2"},
//	    },
//	}
type ExtensionConfig struct {
	Default  string
	Override []ExtOverride
}

// PackageConfig defines a package's registry entries declaratively.
// Provide either Ranges (for contiguous version ranges) or Versions (for
// explicit individual versions). BuildRegistries will generate or use the
// versions, build URLs from URLTemplate, and create domain.Registry entries.
//
// URLTemplate supports placeholders:
//   - {version} — replaced with the version string
//   - {ext}     — replaced by evaluating Extension rules
type PackageConfig struct {
	Name        string
	Source      string
	Ranges      []VersionRange
	Versions    []string
	Skip        []string
	URLTemplate string
	Extension   ExtensionConfig
}

// BuildRegistries generates domain.Registry entries from a PackageConfig.
// If Ranges is provided, versions are generated via GenerateVersions.
// Otherwise, Versions is used directly.
// Each version's URL is built from URLTemplate with {version} and {ext}
// placeholders resolved.
func BuildRegistries(cfg PackageConfig) []domain.Registry {
	var versions []string
	if len(cfg.Ranges) > 0 {
		versions = GenerateVersions(cfg.Ranges, cfg.Skip)
	} else {
		versions = cfg.Versions
	}

	registries := make([]domain.Registry, 0, len(versions))
	for _, v := range versions {
		url := RenderTemplate(cfg.URLTemplate, v)
		if cfg.Extension.Default != "" {
			ext := resolveExtension(cfg.Extension, v)
			url = strings.ReplaceAll(url, "{ext}", ext)
		}
		registries = append(registries, domain.Registry{
			Name:    cfg.Name,
			Source:  cfg.Source,
			URL:     url,
			Version: v,
		})
	}
	return registries
}

// resolveExtension evaluates Override rules in order and returns the first
// matching Ext. If no override matches, Default is returned.
func resolveExtension(cfg ExtensionConfig, version string) string {
	for _, rule := range cfg.Override {
		if CompareVersions(version, rule.Before) < 0 {
			return rule.Ext
		}
	}
	return cfg.Default
}