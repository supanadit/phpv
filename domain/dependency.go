package domain

import "strings"

// Dependency describes a single dependency with an optional flag.
// Version uses the format "exactVersion|constraint" where:
//   - exactVersion is the specific version to download
//   - constraint is the compatibility range (e.g., ">=1.0.2,<4.0.0")
//
// If Version is empty, the dependency has no specific version requirement.
type Dependency struct {
	Name     string
	Version  string
	Optional bool
}

// SplitVersionSpec splits a Dependency.Version spec ("exactVersion|constraint")
// into its exact version and compatibility constraint. When the spec has no
// "|", the whole string is the version and the constraint is empty.
func SplitVersionSpec(spec string) (version, constraint string) {
	version, constraint, _ = strings.Cut(spec, "|")
	return version, constraint
}

// VersionConstraint binds a version range to a set of dependencies.
// When a package version matches VersionRange, the listed Dependencies apply
// instead of the package's Default set.
type VersionConstraint struct {
	VersionRange string
	Dependencies []Dependency
}

// Package defines a package's dependency rules.
// Default applies when no Constraint matches the package version.
type Package struct {
	Package     string
	Default     []Dependency
	Constraints []VersionConstraint
}
