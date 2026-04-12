package domain

type PackageState int

const (
	StateUnknown PackageState = iota
	StateSourceMissing
	StateSourceDownloaded
	StateSourceExtracted
	StateSourceMissingBuilt
	StateBuilt
)

type AdvisorCheck struct {
	Name            string
	Version         string
	PHPVersion      string
	State           PackageState
	Action          string
	SystemAvailable bool
	SystemPath      string
	SystemVersion   string
	Constraint      string
	Message         string
	URL             string
	SourceType      string
	Suggestion      string
}
