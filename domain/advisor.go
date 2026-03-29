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

func (s PackageState) String() string {
	switch s {
	case StateSourceMissing:
		return "source_missing"
	case StateSourceDownloaded:
		return "source_downloaded"
	case StateSourceExtracted:
		return "source_extracted"
	case StateSourceMissingBuilt:
		return "source_missing_built"
	case StateBuilt:
		return "built"
	default:
		return "unknown"
	}
}

type AdvisorCheck struct {
	Name            string
	Version         string
	State           PackageState
	Action          string
	SystemAvailable bool
	SystemPath      string
	Message         string
	URL             string
	SourceType      string
}

func (c AdvisorCheck) String() string {
	return c.Name + "-" + c.Version + " [" + c.State.String() + "] " + c.Action
}
