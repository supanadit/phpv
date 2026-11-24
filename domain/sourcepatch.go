package domain

// SourcePatch describes a source modification applied to extracted code
// before building (e.g., a compatibility fix for a newer toolchain).
type SourcePatch struct {
	Name           string
	Package        string
	VersionRange   string
	Apply          func(sourceDir string) error
	ExtraCFlags    []string
	ConfigureFlags []string
}

// PreparedPatches holds the result of preparing patches for a package.
type PreparedPatches struct {
	ExtraCFlags    []string
	ConfigureFlags []string
	Applied        []string
}

// PatchManifest is the optional flags.yaml beside a patch directory.
type PatchManifest struct {
	ExtraCFlags    []string `yaml:"extraCFlags"`
	ConfigureFlags []string `yaml:"configureFlags"`
}

// PatchEntry is one selected patch-tree directory.
type PatchEntry struct {
	DirPath    string
	Package    string
	RangeDir   string
	Source     string
	PatchFiles []string
	Manifest   *PatchManifest
}
