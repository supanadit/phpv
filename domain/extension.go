package domain

// FlagVersionDef defines a version-gated override for an extension's configure flag.
// When the PHP version matches VersionRange, Flag replaces the extension's default flag.
//
// IMPORTANT — Flag: "" is ambiguous and means TWO different things depending on IsBuiltIn:
//
//  1. IsBuiltIn == false (default): "shared-only" — the extension is NOT compiled into
//     the PHP binary for this version. It must be built as a shared .so via phpize
//     after the main PHP build. Example: iconv in PHP 8.5+ is shared-only (Flag: "",
//     IsBuiltIn: false) — it ships as a .so in the PHP source tree and must be
//     built separately with phpize.
//
//  2. IsBuiltIn == true: "built-in" — the extension IS compiled into the PHP binary
//     for this version. No configure flag is needed (Flag: ""), and no phpize build
//     is required. The extension is already available at runtime. Example: if a
//     future PHP version compiles iconv into the binary, set IsBuiltIn: true.
//
// DO NOT use Flag: "" without setting IsBuiltIn correctly. The SharedOnlyExtensions
// function relies on this distinction to decide which extensions need a phpize build.
type FlagVersionDef struct {
	VersionRange string
	// Flag is the configure flag to pass to PHP's ./configure for this extension.
	// Empty string means no flag is needed (either built-in or shared-only — see IsBuiltIn).
	Flag string
	// IsBuiltIn marks this extension as compiled into the PHP binary for this version.
	// When true: the extension is already part of the PHP binary, no configure flag
	// is needed (Flag should be ""), and no phpize build is required.
	// When false (default): Flag: "" means the extension is shared-only — it ships
	// as a .so in the PHP source tree and must be built with phpize after the main build.
	IsBuiltIn bool
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
