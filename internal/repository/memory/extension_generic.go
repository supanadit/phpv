package memory

import (
	"github.com/supanadit/phpv/internal/repository"

	vsn "github.com/supanadit/phpv/pkg/version"

	"github.com/supanadit/phpv/domain"
)

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
	for _, name := range repository.DefaultExtensionCandidates {
		def, ok := r.extensions[name]
		if !ok {
			skipped = append(skipped, name+" (not found in registry)")
			continue
		}
		if def.MinPHPVersion != "" && vsn.CompareVersions(phpVersion, def.MinPHPVersion) < 0 {
			skipped = append(skipped, name+" (requires PHP "+def.MinPHPVersion+"+)")
			continue
		}
		if def.MaxPHPVersion != "" && vsn.CompareVersions(phpVersion, def.MaxPHPVersion) > 0 {
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
		if fv.IsBuiltIn && vsn.MatchVersionRange(fv.VersionRange, phpVersion) {
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
			if fv.Flag == "" && vsn.MatchVersionRange(fv.VersionRange, phpVersion) {
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
		if vsn.MatchVersionRange(fv.VersionRange, phpVersion) {
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
	return repository.GetCompilerStdRule(phpVersion)
}

func (r *GraphRepository) GetCompilerFlags(compiler, phpVersion string) []string {
	return repository.GetCompilerFlags(compiler, phpVersion)
}

// getDependencies returns the dependency list for a package at a specific version.
func (r *GraphRepository) getDependencies(name string, version string) ([]domain.Dependency, error) {
	pkg, ok := r.packages[name]
	if !ok {
		return nil, nil
	}

	for _, c := range pkg.Constraints {
		if vsn.MatchVersionRange(c.VersionRange, version) {
			return c.Dependencies, nil
		}
	}

	return pkg.Default, nil
}

func (r *GraphRepository) registerPackages() {
	for _, pkg := range BuiltInPackages() {
		r.packages[pkg.Package] = pkg
	}
}

func (r *GraphRepository) registerExtensions() {
	for _, ext := range BuiltInExtensions() {
		r.extensions[ext.Name] = ext
	}
	for _, ext := range BuiltInExtensions() {
		for _, imp := range ext.Implied {
			r.implied[ext.Name] = append(r.implied[ext.Name], imp)
		}
		for _, conflict := range ext.Conflicts {
			r.conflicts[ext.Name] = append(r.conflicts[ext.Name], conflict)
		}
	}
}
