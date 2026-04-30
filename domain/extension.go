package domain

type ExtensionDef struct {
	Flag      string
	MinPHP    string
	MaxPHP    string
	Conflicts []string
	Package   string
	Versions  []VersionConstraintDef
	Implied   []string
}

type VersionConstraintDef struct {
	VersionRange string
	Version      string
}

type Extension struct {
	Name    string
	Type    ExtensionType
	Version string
}

type ExtensionInfo struct {
	Name        string
	Flag        string
	MinPHP      string
	MaxPHP      string
	Package     string
	HasConflict bool
	Conflicts   []string
}

type ExtensionType string

const (
	ExtensionTypePECL ExtensionType = "pecl"
)
