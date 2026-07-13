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
}

type CheckResult struct {
	Distro    Distro
	Available []Package
	Missing   []Package
}
