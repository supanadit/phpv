package system

import "github.com/supanadit/phpv/domain"

// baseAdvisoryLibraries are development libraries PHP can use at build time but
// which are not modelled as buildable dependencies: they are expected to come
// from the system, and no version constraint is checked for them. Libraries an
// extension asks for arrive through ExtensionDef.RequiresPackages instead.
var baseAdvisoryLibraries = []string{"sqlite3", "readline", "pcre2", "sodium"}

// DependencySet returns the system packages to check for a build of the given
// extensions. It is the single source of truth shared by `phpv install`,
// `phpv extension add` and `phpv doctor`, so the three cannot disagree about
// what a build needs.
//
// A package is marked Required when an extension names it in RequiresPackage,
// because that field also feeds the source-build dependency resolver: the
// package really is part of the build plan and its version is checked before
// use. Packages from RequiresPackages and baseAdvisoryLibraries are advisory —
// dev libraries the build can consume but whose version is not gated.
//
// Only packages the build can actually consume are listed: a library with no
// buildable package definition and no extension asking for it is not included.
func DependencySet(exts []domain.ExtensionDef) []Package {
	seen := make(map[string]bool)
	var pkgs []Package

	add := func(name string, required bool) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		pkgs = append(pkgs, Package{Name: name, SystemName: name, Required: required})
	}

	for _, def := range exts {
		add(def.RequiresPackage, true)
	}
	for _, def := range exts {
		for _, name := range def.RequiresPackages {
			add(name, false)
		}
	}
	for _, name := range baseAdvisoryLibraries {
		add(name, false)
	}
	return pkgs
}

// Names returns the dependency names of pkgs, in order. It is the form
// Service.Check expects.
func Names(pkgs []Package) []string {
	names := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		names = append(names, p.Name)
	}
	return names
}
