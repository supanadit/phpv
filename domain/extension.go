package domain

// ExtensionDef defines a PHP extension that can be enabled at build time.
type ExtensionDef struct {
	Name           string
	Description    string
	RequiresPackage string
	MinPHPVersion  string
	MaxPHPVersion  string
	Implied        []string
}

// ExtensionInfo describes an extension for listing/display purposes.
type ExtensionInfo struct {
	Name        string
	Description string
	ValidForPHP []string
}
