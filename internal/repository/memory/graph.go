package memory

import (
	"fmt"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository"
)

type configureFlagRule struct {
	MinVer string
	MaxVer string
	Flags  []string
	Needs  string
}

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
	r.registerExtensions()
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
	optional := make(map[string]bool)
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
			if dep.Optional {
				optional[dep.Name] = true
			}
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
					Name:     pkgName,
					Version:  pkgVersion,
					Optional: optional[pkgName],
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
	def, ok := r.extensions[name]
	if !ok {
		return false
	}
	if def.MinPHPVersion != "" && repository.CompareVersions(phpVersion, def.MinPHPVersion) < 0 {
		return false
	}
	if def.MaxPHPVersion != "" && repository.CompareVersions(phpVersion, def.MaxPHPVersion) > 0 {
		return false
	}
	return true
}

func (r *GraphRepository) GetConflictingExtensions(name string) []string {
	return r.conflicts[name]
}

func (r *GraphRepository) GetExtensionDependency(name string) (string, bool) {
	def, ok := r.extensions[name]
	if !ok {
		return "", false
	}
	if def.RequiresPackage == "" {
		return "", false
	}
	return def.RequiresPackage, true
}

func (r *GraphRepository) GetExtensionDependencyWithVersion(extName, phpVersion string) (string, string, bool) {
	def, ok := r.extensions[extName]
	if !ok || def.RequiresPackage == "" {
		return "", "", false
	}
	for _, v := range def.Versions {
		if repository.MatchVersionRange(v.VersionRange, phpVersion) {
			return def.RequiresPackage, v.Version, true
		}
	}
	return def.RequiresPackage, "", true
}

func (r *GraphRepository) ValidateExtensions(extensions []string, phpVersion string) ([]string, error) {
	var unknown []string
	for _, ext := range extensions {
		if _, ok := r.extensions[ext]; !ok {
			unknown = append(unknown, ext)
		}
	}
	return unknown, nil
}

func (r *GraphRepository) CheckExtensionConflicts(extensions []string) ([]string, [][]string) {
	var conflicts []string
	var groups [][]string
	for _, ext := range extensions {
		if c, ok := r.conflicts[ext]; ok {
			for _, other := range extensions {
				for _, conflict := range c {
					if other == conflict {
						conflicts = append(conflicts, ext)
						groups = append(groups, []string{ext, other})
					}
				}
			}
		}
	}
	return conflicts, groups
}

func (r *GraphRepository) ListExtensions() []domain.ExtensionInfo {
	var list []domain.ExtensionInfo
	for name, def := range r.extensions {
		list = append(list, domain.ExtensionInfo{
			Name:        name,
			Description: def.Description,
		})
	}
	return list
}

func (r *GraphRepository) ListExtensionsForPHP(phpVersion string) []domain.ExtensionInfo {
	var list []domain.ExtensionInfo
	for name, def := range r.extensions {
		if r.IsExtensionValidForPHPVersion(name, phpVersion) {
			list = append(list, domain.ExtensionInfo{
				Name:        name,
				Description: def.Description,
			})
		}
	}
	return list
}

func (r *GraphRepository) ExpandImplied(extensions []string) ([]string, []string) {
	seen := make(map[string]bool)
	var expanded []string
	var added []string
	var expand func(name string)
	expand = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		expanded = append(expanded, name)
		if implied, ok := r.implied[name]; ok {
			for _, imp := range implied {
				if !seen[imp] {
					added = append(added, imp)
					expand(imp)
				}
			}
		}
	}
	for _, ext := range extensions {
		expand(ext)
	}
	return expanded, added
}

var configureFlagRules = map[string][]configureFlagRule{
	"openssl": {
		{MaxVer: "1.0.2", Flags: []string{"shared", "no-ssl3"}},
		{MinVer: "1.1.0", Flags: []string{"shared", "no-ssl3", "no-tests"}},
	},
	"m4": {
		{Flags: []string{"--disable-maintainer-mode"}},
	},
	"libxml2": {
		{Flags: []string{"--disable-shared", "--enable-static", "--without-lzma", "--without-python", "--disable-dependency-tracking", "--with-zlib"}},
	},
	"curl": {
		{Flags: []string{"--with-ssl", "--without-brotli", "--disable-ldap", "--without-libpsl", "--without-libidn2", "--without-zstd", "--without-nghttp2", "--without-zlib"}},
	},
	"icu": {
		{Flags: []string{"--disable-extras", "--disable-samples"}},
	},
}

func (r *GraphRepository) GetConfigureFlags(name, version string) []string {
	rules, ok := configureFlagRules[name]
	if !ok {
		return nil
	}
	for _, rule := range rules {
		minOK := rule.MinVer == "" || repository.CompareVersions(version, rule.MinVer) >= 0
		maxOK := rule.MaxVer == "" || repository.CompareVersions(version, rule.MaxVer) <= 0
		if minOK && maxOK {
			return rule.Flags
		}
	}
	return nil
}

// defaultExtensionCandidates is the recommended default extension set for a
// typical PHP install. Extensions are filtered by MinPHPVersion/MaxPHPVersion
// at runtime via DefaultExtensions().
var defaultExtensionCandidates = []string{
	"bcmath", "curl", "dom", "fileinfo", "filter", "gd",
	"iconv", "intl", "json", "mbstring", "openssl", "opcache",
	"pdo", "pdo_mysql", "pdo_sqlite", "phar", "session",
	"simplexml", "sqlite3", "tokenizer", "xml", "xmlreader",
	"xmlwriter", "zip", "zlib",
}

// DefaultExtensions returns the recommended default extension set for a
// typical PHP install, filtered to those compatible with the given PHP version.
// Returns (included, skipped) where each skipped entry includes a reason.
//
// IMPORTANT — Built-in extensions (IsBuiltIn: true) are SKIPPED from the default
// set because they are already compiled into the PHP binary. They do not need
// a configure flag or a phpize build. Including them would cause "Module already
// loaded" warnings when php.ini has extension=<name>.so for a built-in module.
//
// Users can still explicitly request built-in extensions via --ext <name> at
// install time or `phpv extension add <version> <name>`. The InstallExtension
// function has a safety net that checks php -m and skips the build if the
// extension is already loaded.
func (r *GraphRepository) DefaultExtensions(phpVersion string) ([]string, []string) {
	var included []string
	var skipped []string
	for _, name := range defaultExtensionCandidates {
		def, ok := r.extensions[name]
		if !ok {
			skipped = append(skipped, name+" (not found in registry)")
			continue
		}
		if def.MinPHPVersion != "" && repository.CompareVersions(phpVersion, def.MinPHPVersion) < 0 {
			skipped = append(skipped, name+" (requires PHP "+def.MinPHPVersion+"+)")
			continue
		}
		if def.MaxPHPVersion != "" && repository.CompareVersions(phpVersion, def.MaxPHPVersion) > 0 {
			skipped = append(skipped, name+" (not available in PHP "+def.MaxPHPVersion+"+)")
			continue
		}
		// Skip built-in extensions: they are compiled into the PHP binary and
		// do not need a configure flag or a phpize build. Including them in the
		// default set would cause InstallExtension to add extension=<name>.so to
		// php.ini, which triggers "Module already loaded" warnings at runtime.
		if isBuiltInForVersion(def, phpVersion) {
			skipped = append(skipped, name+" (built-in for PHP "+phpVersion+")")
			continue
		}
		included = append(included, name)
	}
	return included, skipped
}

// isBuiltInForVersion returns true if the extension has a FlagVersions entry
// with IsBuiltIn: true that matches the given PHP version.
//
// This is the authoritative check for whether an extension is compiled into
// the PHP binary for a given version. It is used by:
//   - DefaultExtensions: to skip built-in extensions from the default set
//   - SharedOnlyExtensions: to exclude built-in extensions from phpize builds
//   - InstallExtension (safety net): as a secondary check via php -m
//
// IMPORTANT: Do NOT use Flag: "" alone to determine built-in status. Flag: ""
// is ambiguous — it can mean either "built-in" (IsBuiltIn: true) or "shared-only"
// (IsBuiltIn: false, needs phpize). Always check IsBuiltIn explicitly.
func isBuiltInForVersion(def domain.ExtensionDef, phpVersion string) bool {
	for _, fv := range def.FlagVersions {
		if fv.IsBuiltIn && repository.MatchVersionRange(fv.VersionRange, phpVersion) {
			return true
		}
	}
	return false
}

// SharedOnlyExtensions returns the subset of requested extensions that must
// be built as shared libraries (phpize) rather than compiled into the main
// PHP binary, for the given PHP version.
//
// An extension is "shared-only" when its registry entry has a FlagVersions
// entry with Flag: "" AND IsBuiltIn: false that matches the version range.
// This means PHP ships the extension as a .so in the source tree (ext/<name>/)
// but does NOT compile it into the binary. It must be built with phpize after
// the main PHP build.
//
// IMPORTANT: Extensions with IsBuiltIn: true are EXCLUDED from this list.
// They are already compiled into the PHP binary and do not need a phpize build.
// Including them would cause InstallExtension to build a duplicate .so and add
// extension=<name>.so to php.ini, triggering "Module already loaded" warnings.
//
// The distinction between built-in and shared-only is critical:
//   - IsBuiltIn: true  + Flag: ""  = built-in (no build needed)
//   - IsBuiltIn: false + Flag: ""  = shared-only (needs phpize build)
//   - IsBuiltIn: false + Flag: "X" = configure-time extension (needs --enable-X)
func (r *GraphRepository) SharedOnlyExtensions(phpVersion string, requested []string) []string {
	var result []string
	for _, name := range requested {
		def, ok := r.extensions[name]
		if !ok {
			continue
		}
		// Skip built-in extensions: they are already in the PHP binary.
		if isBuiltInForVersion(def, phpVersion) {
			continue
		}
		for _, fv := range def.FlagVersions {
			if fv.Flag == "" && repository.MatchVersionRange(fv.VersionRange, phpVersion) {
				result = append(result, name)
				break
			}
		}
	}
	return result
}

func (r *GraphRepository) GetPHPConfigureFlags(phpVersion string, extensions []string) []string {
	flags := []string{
		"--disable-all",
		"--enable-cli",
		// --disable-maintainer-mode prevents zend_vm_gen.php from being run
		// during the build. The generated files (zend_vm_execute.h,
		// zend_vm_opcodes.h, etc.) in the PHP source distribution are
		// pre-generated by the PHP team with valid ZEND_VM_KIND values.
		// Running zend_vm_gen.php requires PHP to be installed, which is
		// not available when building PHP from source. The original files
		// work correctly with the default VM dispatch (CALL).
		"--disable-maintainer-mode",
	}
	for _, ext := range extensions {
		extFlags := r.GetExtensionConfigureFlags(ext, phpVersion)
		flags = append(flags, extFlags...)
	}
	return flags
}

// GetExtensionConfigureFlags returns the configure flags for a single extension
// for the given PHP version. The flag is resolved in this order:
//  1. def.ConfigureFlags (if set, returned directly)
//  2. def.FlagVersions (if a matching VersionRange is found, use that flag)
//  3. def.Flag (fallback default)
//  4. If the resolved flag is empty, return nil (no configure flag needed)
//
// IMPORTANT: def.FlagVersions is checked BEFORE def.Flag. This allows extensions
// to define version-specific flags without setting a default Flag field.
// For example, iconv has no default Flag but has FlagVersions entries for
// different PHP versions. Without this ordering, extensions that only use
// FlagVersions would get nil (no flag) and be silently skipped.
func (r *GraphRepository) GetExtensionConfigureFlags(name string, phpVersion string) []string {
	def, ok := r.extensions[name]
	if !ok {
		return nil
	}
	if len(def.ConfigureFlags) > 0 {
		return def.ConfigureFlags
	}
	// Start with the default flag, then check FlagVersions for an override.
	flag := def.Flag
	for _, fv := range def.FlagVersions {
		if repository.MatchVersionRange(fv.VersionRange, phpVersion) {
			flag = fv.Flag
			break
		}
	}
	if flag == "" {
		return nil
	}
	return []string{flag}
}

func (r *GraphRepository) GetCompilerStdRule(phpVersion string) domain.CompilerRule {
	return getCompilerStdRule(phpVersion)
}

func (r *GraphRepository) GetCompilerFlags(compiler, phpVersion string) []string {
	return getCompilerFlags(compiler, phpVersion)
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

func (r *GraphRepository) registerExtensions() {
	for _, ext := range builtInExtensions() {
		r.extensions[ext.Name] = ext
	}
	for _, ext := range builtInExtensions() {
		for _, imp := range ext.Implied {
			r.implied[ext.Name] = append(r.implied[ext.Name], imp)
		}
		for _, conflict := range ext.Conflicts {
			r.conflicts[ext.Name] = append(r.conflicts[ext.Name], conflict)
		}
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
					VersionRange: ">=8.0.0 <8.1.0",
					Dependencies: []domain.Dependency{
						{Name: "openssl", Version: "1.1.1w|>=1.0.2,<4.0.0"},
						{Name: "libxml2", Version: "2.9.14|~2.9.0"},
						{Name: "zlib", Version: "1.2.13|>=1.2.0,<1.3.0"},
						{Name: "oniguruma", Version: "6.9.9|~6.9.0"},
						{Name: "curl", Version: "7.88.1|>=7.80.0"},
						{Name: "icu", Version: "63.1|>=63.1,<74"},
					},
				},
				{
					VersionRange: ">=7.1.0 <8.0.0",
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
						{Name: "icu", Version: "58.2|>=58.0,<60"},
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
						{Name: "icu", Version: "58.2|>=58.0,<60"},
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
						{Name: "icu", Version: "58.2|>=58.0,<60"},
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
						{Name: "icu", Version: "58.2|>=58.0,<60"},
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
						{Name: "icu", Version: "58.2|>=58.0,<60"},
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

func builtInExtensions() []domain.ExtensionDef {
	return []domain.ExtensionDef{
		{
			Name:          "bcmath",
			Description:   "BC Math arbitrary precision",
			Flag:          "--enable-bcmath",
			MinPHPVersion: "5.0",
		},
		{
			Name:            "bz2",
			Description:     "BZip2 compression",
			Flag:            "--with-bz2",
			MinPHPVersion:   "5.0",
			RequiresPackage: "bzip2",
		},
		{
			Name:          "calendar",
			Description:   "Calendar conversion support",
			Flag:          "--enable-calendar",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "ctype",
			Description:   "Character type checking",
			Flag:          "--enable-ctype",
			MinPHPVersion: "5.0",
		},
		{
			Name:            "curl",
			Description:     "cURL support",
			Flag:            "--with-curl",
			MinPHPVersion:   "5.0",
			RequiresPackage: "curl",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0.0", Version: "8.10.1|>=8.0.0"},
				{VersionRange: ">=7.0.0 <8.0.0", Version: "7.88.1|>=7.80.0"},
				{VersionRange: ">=5.6.0 <7.0.0", Version: "7.20.0|>=7.20.0,<7.21.0"},
				{VersionRange: ">=5.1.0 <5.6.0", Version: "7.12.1|>=7.12.0,<7.13.0"},
				{VersionRange: ">=5.0.0 <5.1.0", Version: "7.12.0|>=7.12.0,<7.13.0"},
			},
		},
		{
			Name:          "dba",
			Description:   "Database (dbm-style) abstraction layer",
			Flag:          "--enable-dba",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "dom",
			Description:   "DOM support",
			Flag:          "--enable-dom",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "enchant",
			Description:   "Enchant spelling library",
			Flag:          "--with-enchant",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "exif",
			Description:   "EXIF headers",
			Flag:          "--enable-exif",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "fileinfo",
			Description:   "File information",
			Flag:          "--enable-fileinfo",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "filter",
			Description:   "Data filtering",
			Flag:          "--enable-filter",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "ftp",
			Description:   "FTP support",
			Flag:          "--enable-ftp",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "gd",
			Description:   "GD image processing",
			Flag:          "--with-gd",
			MinPHPVersion: "5.0",
			FlagVersions: []domain.FlagVersionDef{
				{VersionRange: ">=7.4", Flag: "--enable-gd"},
			},
		},
		{
			Name:          "gettext",
			Description:   "Gettext support",
			Flag:          "--with-gettext",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "gmp",
			Description:   "GNU MP support",
			Flag:          "--with-gmp",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "hash",
			Description:   "HASH message digest",
			Flag:          "--enable-hash",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "iconv",
			Description:   "Iconv support",
			MinPHPVersion: "5.0",
			FlagVersions: []domain.FlagVersionDef{
				// PHP < 8.5: iconv is a configure-time extension (--with-iconv).
				{VersionRange: ">=5.0 <8.5.0", Flag: "--with-iconv"},
				// PHP >= 8.5: iconv is NOT compiled into the binary. It ships as a
				// shared .so in the PHP source tree (ext/iconv/) and must be built
				// with phpize after the main PHP build. Flag: "" means no configure
				// flag is needed; IsBuiltIn: false means it's shared-only.
				//
				// IMPORTANT: IsBuiltIn: false + Flag: "" = shared-only (needs phpize).
				// Do NOT set IsBuiltIn: true here unless iconv is actually compiled
				// into the PHP binary for this version.
				{VersionRange: ">=8.5.0", Flag: "", IsBuiltIn: false},
			},
		},
		{
			Name:          "imap",
			Description:   "IMAP support",
			Flag:          "--with-imap",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "interbase",
			Description:   "InterBase support",
			Flag:          "--with-interbase",
			MinPHPVersion: "5.0",
		},
		{
			Name:            "intl",
			Description:     "Internationalization support",
			Flag:            "--enable-intl",
			MinPHPVersion:   "5.0",
			RequiresPackage: "icu",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0", Version: "74.2|>=74.2"},
				{VersionRange: ">=7.4", Version: "63.1|>=63.1,<74"},
				{VersionRange: ">=5.0 <7.4", Version: "58.2|>=58.0,<60"},
			},
		},
		{
			Name:          "json",
			Description:   "JSON support",
			Flag:          "--enable-json",
			MinPHPVersion: "5.2",
		},
		{
			Name:          "ldap",
			Description:   "LDAP support",
			Flag:          "--with-ldap",
			MinPHPVersion: "5.0",
		},
		{
			Name:            "libxml",
			Description:     "LIBXML support",
			Flag:            "--enable-libxml",
			MinPHPVersion:   "5.0",
			RequiresPackage: "libxml2",
			FlagVersions: []domain.FlagVersionDef{
				{VersionRange: ">=7.4", Flag: "--with-libxml"},
			},
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.2.0", Version: "2.12.7|~2.12.0"},
				{VersionRange: ">=8.0.0 <8.2.0", Version: "2.11.7|~2.11.0"},
				{VersionRange: ">=5.0.0 <8.0.0", Version: "2.9.14|~2.9.0"},
			},
		},
		{
			Name:            "mbstring",
			Description:     "Multibyte String support",
			Flag:            "--enable-mbstring",
			MinPHPVersion:   "5.0",
			RequiresPackage: "oniguruma",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0.0", Version: "6.9.9|~6.9.0"},
				{VersionRange: ">=7.4.0 <8.0.0", Version: "6.9.8|~6.9.0"},
				{VersionRange: ">=5.0.0 <7.4.0", Version: "5.9.6|~5.9.0"},
			},
		},
		{
			Name:          "mysql",
			Description:   "MySQL support (deprecated)",
			Flag:          "--with-mysql",
			MinPHPVersion: "5.0",
			MaxPHPVersion: "5.6",
			Conflicts:     []string{"mysqli", "pdo_mysql"},
			Implied:       []string{"zlib"},
		},
		{
			Name:          "mysqli",
			Description:   "MySQL Improved Extension",
			Flag:          "--with-mysqli",
			MinPHPVersion: "5.0",
			Conflicts:     []string{"mysql", "pdo_mysql"},
			Implied:       []string{"zlib"},
		},
		{
			Name:          "odbc",
			Description:   "ODBC support",
			Flag:          "--with-odbc",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "opcache",
			Description:   "OPcache",
			Flag:          "--enable-opcache",
			MinPHPVersion: "7.0",
		},
		{
			Name:            "openssl",
			Description:     "OpenSSL support",
			Flag:            "--with-openssl",
			MinPHPVersion:   "5.0",
			RequiresPackage: "openssl",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.4.0", Version: "1.1.1w|>=1.1.1,<4.0.0"},
				{VersionRange: ">=8.1.0 <8.4.0", Version: "1.1.1w|>=1.0.2,<4.0.0"},
				{VersionRange: ">=7.1.0 <8.1.0", Version: "1.1.1w|>=1.1.1,<1.3.0"},
				{VersionRange: ">=7.0.0 <7.1.0", Version: "1.0.1u|>=0.9.8,<1.2.0"},
				{VersionRange: ">=5.0.0 <7.0.0", Version: "1.0.1u|>=1.0.0,<1.1.0"},
			},
		},
		{
			Name:          "pcntl",
			Description:   "PCNTL process control",
			Flag:          "--enable-pcntl",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pcre",
			Description:   "PCRE regex support",
			Flag:          "--with-pcre-regex",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pdo",
			Description:   "PHP Data Objects",
			Flag:          "--enable-pdo",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pdo_dblib",
			Description:   "Sybase driver for PDO",
			Flag:          "--with-pdo-dblib",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:          "pdo_firebird",
			Description:   "Firebird driver for PDO",
			Flag:          "--with-pdo-firebird",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:          "pdo_mysql",
			Description:   "MySQL driver for PDO",
			Flag:          "--with-pdo-mysql=mysqlnd",
			MinPHPVersion: "5.0",
			Conflicts:     []string{"mysql", "mysqli"},
			Implied:       []string{"pdo", "zlib"},
		},
		{
			Name:          "pdo_oci",
			Description:   "Oracle driver for PDO",
			Flag:          "--with-pdo-oci",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:          "pdo_odbc",
			Description:   "ODBC driver for PDO",
			Flag:          "--with-pdo-odbc",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:            "pdo_pgsql",
			Description:     "PostgreSQL driver for PDO",
			Flag:            "--with-pdo-pgsql",
			MinPHPVersion:   "5.0",
			RequiresPackage: "libpq",
			Implied:         []string{"pdo"},
		},
		{
			Name:          "pdo_sqlite",
			Description:   "SQLite driver for PDO",
			Flag:          "--with-pdo-sqlite",
			MinPHPVersion: "5.0",
			Implied:       []string{"pdo"},
		},
		{
			Name:            "pgsql",
			Description:     "PostgreSQL support",
			Flag:            "--with-pgsql",
			MinPHPVersion:   "5.0",
			RequiresPackage: "libpq",
		},
		{
			Name:          "phar",
			Description:   "Phar archive support",
			Flag:          "--enable-phar",
			MinPHPVersion: "5.0",
			Implied:       []string{"json", "hash"},
		},
		{
			Name:          "posix",
			Description:   "POSIX system calls",
			Flag:          "--enable-posix",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "pspell",
			Description:   "PSpell support",
			Flag:          "--with-pspell",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "random",
			Description:   "Random number generator",
			Flag:          "--with-random",
			MinPHPVersion: "7.0",
		},
		{
			Name:          "readline",
			Description:   "Readline support",
			Flag:          "--with-readline",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "recode",
			Description:   "Recode support",
			Flag:          "--with-recode",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "session",
			Description:   "Session support",
			Flag:          "--enable-session",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "shmop",
			Description:   "Shared memory operations",
			Flag:          "--enable-shmop",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "simplexml",
			Description:   "SimpleXML support",
			Flag:          "--enable-simplexml",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "snmp",
			Description:   "SNMP support",
			Flag:          "--with-snmp",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "soap",
			Description:   "SOAP support",
			Flag:          "--enable-soap",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "sockets",
			Description:   "Socket support",
			Flag:          "--enable-sockets",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sodium",
			Description:   "Sodium support",
			Flag:          "--with-sodium",
			MinPHPVersion: "7.2",
		},
		{
			Name:          "sqlite3",
			Description:   "SQLite3 support",
			Flag:          "--with-sqlite3",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "standard",
			Description:   "Standard PHP functions",
			Flag:          "--enable-standard",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sysvmsg",
			Description:   "System V message queues",
			Flag:          "--enable-sysvmsg",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sysvsem",
			Description:   "System V semaphores",
			Flag:          "--enable-sysvsem",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "sysvshm",
			Description:   "System V shared memory",
			Flag:          "--enable-sysvshm",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "tidy",
			Description:   "Tidy HTML cleaner",
			Flag:          "--with-tidy",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "tokenizer",
			Description:   "Tokenizer support",
			Flag:          "--enable-tokenizer",
			MinPHPVersion: "5.0",
		},
		{
			Name:          "tokenizer_all",
			Description:   "Tokenizer all tokens",
			Flag:          "--enable-tokenizer-all",
			MinPHPVersion: "7.0",
		},
		{
			Name:          "xml",
			Description:   "XML support",
			Flag:          "--enable-xml",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xmlreader",
			Description:   "XMLReader support",
			Flag:          "--enable-xmlreader",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xmlrpc",
			Description:   "XML-RPC support",
			Flag:          "--enable-xmlrpc",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xmlwriter",
			Description:   "XMLWriter support",
			Flag:          "--enable-xmlwriter",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "xsl",
			Description:   "XSL support",
			Flag:          "--with-xsl",
			MinPHPVersion: "5.0",
			Implied:       []string{"libxml"},
		},
		{
			Name:          "zend_test",
			Description:   "Zend test extension",
			Flag:          "--enable-zend-test",
			MinPHPVersion: "7.0",
		},
		{
			Name:          "zip",
			Description:   "ZIP archive support",
			Flag:          "--enable-zip",
			MinPHPVersion: "5.0",
			FlagVersions: []domain.FlagVersionDef{
				{VersionRange: ">=7.4", Flag: "--with-zip"},
			},
		},
		{
			Name:            "zlib",
			Description:     "Zlib compression",
			Flag:            "--with-zlib",
			MinPHPVersion:   "5.0",
			RequiresPackage: "zlib",
			Versions: []domain.VersionConstraintDef{
				{VersionRange: ">=8.0.0", Version: "1.3.1|>=1.3.0"},
				{VersionRange: ">=5.0.0 <8.0.0", Version: "1.2.13|>=1.2.0,<1.3.0"},
			},
		},
	}
}
