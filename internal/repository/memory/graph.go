package memory

import (
	"fmt"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository"
)

// GraphRepository is an in-memory implementation of graph.GraphRepository.
// It holds hardcoded build knowledge: dependency graph, extension definitions,
// flag rules, and compiler rules.
type GraphRepository struct {
	packages      map[string]domain.Package
	extensions    map[string]domain.ExtensionDef
	flagRules     []domain.FlagRule
	compilerRules []domain.CompilerRule
	conflicts     map[string][]string
	implied       map[string][]string
}

// NewGraphRepository creates a new in-memory graph repository with
// the built-in knowledge pre-registered.
func NewGraphRepository() *GraphRepository {
	r := &GraphRepository{
		packages:   make(map[string]domain.Package),
		extensions: make(map[string]domain.ExtensionDef),
		conflicts:  make(map[string][]string),
		implied:    make(map[string][]string),
	}
	r.registerPackages()
	return r
}

// GetOrderedDependencies returns all transitive dependencies for
// (name, version) in dependency order — dependencies before dependents.
// The root package itself is excluded from the result.
// Circular dependencies are detected and return an error.
func (r *GraphRepository) GetOrderedDependencies(name string, version string) ([]domain.Dependency, error) {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	seen := make(map[string]bool)
	var result []domain.Dependency

	var resolve func(pkgName, pkgVersion string) error
	resolve = func(pkgName, pkgVersion string) error {
		if visited[pkgName] {
			return nil
		}
		if visiting[pkgName] {
			return fmt.Errorf("circular dependency detected involving %s", pkgName)
		}

		visiting[pkgName] = true

		deps, err := r.getDependencies(pkgName, pkgVersion)
		if err != nil {
			visiting[pkgName] = false
			return fmt.Errorf("failed to get dependencies for %s@%s: %w", pkgName, pkgVersion, err)
		}

		for _, dep := range deps {
			depVersion := extractVersion(dep.Version)
			if err := resolve(dep.Name, depVersion); err != nil {
				if dep.Optional {
					continue
				}
				visiting[pkgName] = false
				return err
			}
		}

		visiting[pkgName] = false
		visited[pkgName] = true

		if pkgName != name {
			key := pkgName + "@" + pkgVersion
			if !seen[key] {
				seen[key] = true
				result = append(result, domain.Dependency{
					Name:    pkgName,
					Version: pkgVersion,
				})
			}
		}

		return nil
	}

	if err := resolve(name, version); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *GraphRepository) GetExtensionDef(name string) (domain.ExtensionDef, bool) {
	def, ok := r.extensions[name]
	return def, ok
}

func (r *GraphRepository) IsExtensionValidForPHPVersion(name, phpVersion string) bool {
	return true
}

func (r *GraphRepository) GetConflictingExtensions(name string) []string {
	return r.conflicts[name]
}

func (r *GraphRepository) GetExtensionDependency(name string) (string, bool) {
	return "", false
}

func (r *GraphRepository) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	return "", "", false
}

func (r *GraphRepository) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	return nil, nil
}

func (r *GraphRepository) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	return nil, nil
}

func (r *GraphRepository) ListExtensions() []domain.ExtensionInfo {
	return nil
}

func (r *GraphRepository) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	return nil
}

func (r *GraphRepository) ExpandImplied(extensions []string) ([]string, []string) {
	return extensions, nil
}

func (r *GraphRepository) GetConfigureFlags(name, version string) []string {
	return nil
}

func (r *GraphRepository) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	return nil
}

func (r *GraphRepository) GetCompilerStdRule(phpVersion string) domain.CompilerRule {
	return domain.CompilerRule{}
}

func (r *GraphRepository) GetCompilerFlags(compiler, phpVersion string) []string {
	return nil
}

// getDependencies returns the dependency list for a package at a specific version.
func (r *GraphRepository) getDependencies(name string, version string) ([]domain.Dependency, error) {
	pkg, ok := r.packages[name]
	if !ok {
		return nil, nil
	}

	for _, c := range pkg.Constraints {
		if repository.MatchVersionRange(c.VersionRange, version) {
			return c.Dependencies, nil
		}
	}

	return pkg.Default, nil
}

func (r *GraphRepository) registerPackages() {
	for _, pkg := range builtInPackages() {
		r.packages[pkg.Package] = pkg
	}
}

func extractVersion(v string) string {
	if v == "" {
		return ""
	}
	if before, _, found := strings.Cut(v, "|"); found {
		return before
	}
	return v
}

// builtInPackages returns the hardcoded dependency definitions for all
// packages that phpv knows how to build from source.
func builtInPackages() []domain.Package {
	return []domain.Package{
		{
			Package: "php",
			Default: []domain.Dependency{},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=8.1.0 <8.2.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.0.2,<4.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "6.9.9|~6.9.0"},
						{Name: "curl", Version: "8.5.0|>=7.80.0"},
					},
				},
				{
					VersionRange: ">=7.1.0 <8.1.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.1,<1.3.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "6.9.9|~6.9.0"},
						{Name: "curl", Version: "7.88.1|>=7.80.0"},
					},
				},
				{
					VersionRange: ">=7.0.0 <7.1.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.0.1u|>=0.9.8,<1.2.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.88.1|>=7.80.0"},
					},
				},
				{
					VersionRange: ">=5.0.0 <7.0.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.0.1u|>=1.0.0,<1.1.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.88.1|>=7.80.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
				{
					VersionRange: ">=4.4.0 <5.0.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.0.1u|>=1.0.0,<1.1.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.88.1|>=7.80.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
				{
					VersionRange: ">=4.3.0 <4.4.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "0.9.8zh|>=0.9.8,<1.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.12.0|>=7.12.0,<7.13.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
				{
					VersionRange: ">=4.0.0 <4.3.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "0.9.8zh|>=0.9.8,<1.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "5.9.6|~5.9.0"},
						{Name: "curl", Version: "7.12.0|>=7.12.0,<7.13.0"},
						{Name: "flex", Version: "", Optional: true},
						{Name: "bison", Version: "", Optional: true},
					},
				},
			},
		},
		{
			Package: "openssl",
			Default: []domain.Dependency{
				{Name: "perl", Version: "5.38.2|>=5.32.0"},
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=3.0.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=1.1.0 <3.0.0",
					Dependencies: []domain.Dependency{
						{Name: "perl", Version: "5.38.2|>=5.32.0"},
						{Name: "m4", Version: "1.4.19|>=1.4.19"},
						{Name: "autoconf", Version: "2.71|>=2.71,<2.74"},
						{Name: "automake", Version: "1.16.5|>=1.16"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=1.0.0 <1.1.0",
					Dependencies: []domain.Dependency{
						{Name: "perl", Version: "5.38.2|>=5.32.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
				{
					VersionRange: ">=0.9.0 <1.0.0",
					Dependencies: []domain.Dependency{
						{Name: "perl", Version: "5.38.2|>=5.32.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.13"},
						{Name: "automake", Version: "1.4-p6"},
						{Name: "libtool", Version: "1.5.26"},
					},
				},
			},
		},
		{
			Package: "libxml2",
			Default: []domain.Dependency{
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=2.12.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=2.11.0 <2.12.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=2.9.0 <2.11.0",
					Dependencies: []domain.Dependency{
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
			},
		},
		{
			Package: "zlib",
			Default: []domain.Dependency{},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=1.3.0", Dependencies: []domain.Dependency{}},
				{VersionRange: ">=1.2.0 <1.3.0", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package: "oniguruma",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=6.9.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=5.9.0 <6.9.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
			},
		},
		{
			Package: "curl",
			Default: []domain.Dependency{
				{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
				{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=8.0.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.72"},
						{Name: "automake", Version: "1.17"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=7.80.0 <8.0.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.1,<4.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
						{Name: "automake", Version: "1.16.5"},
						{Name: "libtool", Version: "2.5.4"},
					},
				},
				{
					VersionRange: ">=7.20.0 <7.80.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.0,<3.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
						{Name: "automake", Version: "1.15.1"},
						{Name: "libtool", Version: "2.4.6"},
					},
				},
				{
					VersionRange: ">=7.12.0 <7.20.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.1.0,<3.0.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.59"},
						{Name: "automake", Version: "1.9.6"},
						{Name: "libtool", Version: "1.5.26"},
					},
				},
			},
		},
		{
			Package: "autoconf",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
			},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=2.69", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package: "automake",
			Default: []domain.Dependency{
				{Name: "autoconf", Version: "2.69"},
				{Name: "m4", Version: "1.4.19"},
			},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=1.15", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package: "libtool",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.71"},
			},
			Constraints: []domain.VersionConstraint{
				{
					VersionRange: ">=2.5.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.71"},
					},
				},
				{
					VersionRange: ">=1.5.0 <2.5.0",
					Dependencies: []domain.Dependency{
						{Name: "m4", Version: "1.4.19"},
						{Name: "autoconf", Version: "2.69"},
					},
				},
			},
		},
		{
			Package: "m4",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=1.4.19", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package:     "perl",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package: "flex",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
				{Name: "libtool", Version: "2.4.6"},
			},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package: "bison",
			Default: []domain.Dependency{
				{Name: "m4", Version: "1.4.19"},
				{Name: "autoconf", Version: "2.69"},
				{Name: "automake", Version: "1.15.1"},
			},
			Constraints: []domain.VersionConstraint{
				{VersionRange: ">=3.0", Dependencies: []domain.Dependency{}},
			},
		},
		{
			Package:     "icu",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
		{
			Package:     "zig",
			Default:     []domain.Dependency{},
			Constraints: []domain.VersionConstraint{},
		},
	}
}
