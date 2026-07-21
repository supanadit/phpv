package domain

// Silo represents the phpv storage root.
type Silo struct {
	Root string
}

// InstallState tracks the lifecycle of a PHP installation.
type InstallState string

const (
	StateNone        InstallState = ""
	StateInProgress  InstallState = "in_progress"
	StateInstalled   InstallState = "installed"
	StateFailed      InstallState = "failed"
	StateInterrupted InstallState = "interrupted"
)
