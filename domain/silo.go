package domain

type Silo struct {
	Root string
}

type InstallState int

const (
	StateNone InstallState = iota
	StateInProgress
	StateInstalled
	StateFailed
)

type DependencyInfo struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	BuiltFromSource bool   `json:"built_from_source"`
	SystemPath      string `json:"system_path,omitempty"`
}

type PHPInstallation struct {
	PHPVersion   string           `json:"php"`
	Dependencies []DependencyInfo `json:"dependencies"`
	BuildTools   []string         `json:"build_tools"`
	State        InstallState     `json:"state"`
}
