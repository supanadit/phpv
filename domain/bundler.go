package domain

type BundlerConfig struct {
	TargetOS   string
	TargetArch string
	Jobs       int
}

type InstallResult struct {
	Forge           Forge
	BuiltPackages   []string
	SkippedPackages []string
}

type VersionResolved struct {
	Package string
	Version string
	For     string
}
