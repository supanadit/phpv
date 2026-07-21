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

// Checksum binds a specific version to its checksum algorithm and value.
type Checksum struct {
	Version string
	Type    string
	Value   string
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

// URLOverride defines a URL template override for versions older than Before.
// Overrides are evaluated in order — first match wins.
type URLOverride struct {
	Before string
	URL    string
}

// PackageConfig defines a package's registry entries declaratively.
// Provide either Ranges (for contiguous version ranges) or Versions (for
// explicit individual versions). BuildRegistries will generate or use the
// versions, build URLs from URLTemplate, and create domain.Registry entries.
//
// URLTemplate supports placeholders:
//   - {version} — replaced with the version string
//   - {ext}     — replaced by evaluating Extension rules
//
// URLOverrides allows version-conditional URL templates. The first override
// whose Before version is greater than the current version wins. This is
// useful for packages that moved their archive host (e.g., PHP < 5.3 is on
// museum.php.net instead of www.php.net).
//
// OS sets the OS property on every generated entry. Use "all" (or
// leave empty) for packages that are OS-agnostic (e.g., source code).
// Use a specific OS value such as "linux", "darwin", or "windows" for
// pre-built binaries that target a single platform.
type PackageConfig struct {
	Name         string
	Type         string
	Ranges       []VersionRange
	Versions     []string
	Skip         []string
	URLTemplate  string
	URLOverrides []URLOverride
	Extension    ExtensionConfig
	OS           string
	Checksums    []Checksum
}

// BuildRegistries generates domain.Registry entries from a PackageConfig.
// If Ranges is provided, versions are generated via GenerateVersions.
// Otherwise, Versions is used directly.
// Each version's URL is built from URLTemplate with {version} and {ext}
// placeholders resolved. The OS field is set from cfg.OS, defaulting to
// "all" when empty.
func BuildRegistries(cfg PackageConfig) []domain.Registry {
	var versions []string
	if len(cfg.Ranges) > 0 {
		versions = GenerateVersions(cfg.Ranges, cfg.Skip)
	} else {
		versions = cfg.Versions
	}

	checksumMap := make(map[string]Checksum, len(cfg.Checksums))
	for _, c := range cfg.Checksums {
		checksumMap[c.Version] = c
	}

	os := cfg.OS
	if os == "" {
		os = "all"
	}

	registries := make([]domain.Registry, 0, len(versions))
	for _, v := range versions {
		tmpl := cfg.URLTemplate
		for _, override := range cfg.URLOverrides {
			if CompareVersions(v, override.Before) < 0 {
				tmpl = override.URL
				break
			}
		}
		url := RenderTemplate(tmpl, v)
		if cfg.Extension.Default != "" {
			ext := resolveExtension(cfg.Extension, v)
			url = strings.ReplaceAll(url, "{ext}", ext)
		}

		entry := domain.Registry{
			Name:    cfg.Name,
			Type:    cfg.Type,
			URL:     url,
			Version: v,
			OS:      os,
		}
		if c, ok := checksumMap[v]; ok {
			entry.ChecksumType = c.Type
			entry.ChecksumValue = c.Value
		}
		registries = append(registries, entry)
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
