package patcher

// Patch describes a single source modification applied to extracted source code
// before building. Patches are needed when upstream packages cannot build on
// modern toolchains (e.g., GCC 15's stricter C23 pointer-type checking).
type Patch struct {
	// Name identifies the patch for logging.
	Name string
	// Package is the package name this patch applies to.
	Package string
	// VersionRange is an optional constraint (e.g., ">=6.9.0, <6.10.0").
	// Empty means "any version".
	VersionRange string
	// Apply mutates the extracted source tree in place.
	Apply func(sourceDir string) error
	// ExtraCFlags, if non-nil, are additional CFLAGS injected into the
	// package's build environment (e.g., to relax strict warnings).
	ExtraCFlags []string
}

// PatcherRepository resolves the list of patches to apply for a given package.
type PatcherRepository interface {
	PatchesFor(name string, version string) []Patch
}
