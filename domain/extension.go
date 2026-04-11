package domain

type ExtensionDef struct {
	Flag      string
	MinPHP    string
	MaxPHP    string
	Conflicts []string
	Package   string
	Versions  []VersionConstraintDef
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

type ExtensionType string

const (
	ExtensionTypePECL ExtensionType = "pecl"
)
