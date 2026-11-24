package domain

// Distro describes the host distribution and its package manager.
type Distro struct {
	Name    string
	Version string
	PM      string
}

// SystemPackage describes one OS-level package checked or installed by a build.
type SystemPackage struct {
	Name       string
	SystemName string
	Version    string
	Installed  bool
	// Required marks the package as a known build dependency whose installed
	// version is checked against the dependency constraint before use. An
	// advisory package (Required false) is a dev library the build may need but
	// which is not modelled as a buildable dependency, so its version is not
	// gated.
	Required bool
}

// SystemCheckResult groups the installed and missing packages for a distro.
type SystemCheckResult struct {
	Distro    Distro
	Available []SystemPackage
	Missing   []SystemPackage
}
