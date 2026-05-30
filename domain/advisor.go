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

// CompilerType represents a C compiler that can build PHP.
type CompilerType string

const (
	CompilerTypeGCC CompilerType = "gcc"
	CompilerTypeZig CompilerType = "zig"
)

// CompilerInfo contains information about a compiler for a given PHP version.
type CompilerInfo struct {
	Type         CompilerType
	Path         string
	Name         string
	Version      string
	Available    bool
	AutoDownload bool
}

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
