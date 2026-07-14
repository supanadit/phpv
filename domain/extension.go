package domain

// FlagVersionDef defines a version-gated override for an extension's configure flag.
// When the PHP version matches VersionRange, Flag replaces the extension's default flag.
type FlagVersionDef struct {
	VersionRange string
	Flag         string
}

// VersionConstraintDef defines a version constraint for a dependency package.
// VersionRange is the PHP version range where this constraint applies.
// Version is the package version to use (format: "exactVersion|constraint").
type VersionConstraintDef struct {
	VersionRange string
	Version      string
}

// ExtensionDef defines a PHP extension that can be enabled at build time.
type ExtensionDef struct {
	Name            string
	Description     string
	Flag            string
	RequiresPackage string
	MinPHPVersion   string
	MaxPHPVersion   string
	Implied         []string
	Conflicts       []string
	ConfigureFlags  []string
	FlagVersions    []FlagVersionDef
	Versions        []VersionConstraintDef
}

// ExtensionInfo describes an extension for listing/display purposes.
type ExtensionInfo struct {
	Name        string
	Description string
	ValidForPHP []string
}
