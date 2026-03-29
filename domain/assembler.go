package domain

type VersionConstraint struct {
	VersionRange string
	Dependencies []Dependency
}

type Package struct {
	Package     string
	Default     []Dependency
	Constraints []VersionConstraint
}
