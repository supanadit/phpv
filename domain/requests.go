package domain

// BuildRequest carries the inputs for compiling a package from source.
type BuildRequest struct {
	Name, Version, SourceDir, InstallPrefix string
	ExtraEnv, ExtraConfigureFlags           []string
	Verbose                                 bool
	Jobs                                    int
}

// InstallRequest carries the inputs for installing a compiled package.
type InstallRequest struct {
	Name, Version, BuildDir, Prefix string
	Verbose                         bool
	Jobs                            int
}

// PECLInstallRequest carries the inputs for installing a PECL extension.
type PECLInstallRequest struct {
	Name, PHPVersion string
	Jobs             int
	ExtraINI         []string
}

// ExtensionInstallRequest carries the inputs for building a bundled PHP extension.
type ExtensionInstallRequest struct {
	PHPVersion, ExtName, PHPSourceDir, PHPPrefix string
	Jobs                                         int
}
