package system

type Distro struct {
	Name    string
	Version string
	PM      string
}

type Package struct {
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

type CheckResult struct {
	Distro    Distro
	Available []Package
	Missing   []Package
}
