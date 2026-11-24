package domain

// Severity classifies a health-check finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Issue is one health-check finding.
type Issue struct {
	Severity Severity `json:"severity"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Fix      string   `json:"fix,omitempty"`
}

// PECLInstallResult describes an installed PECL extension.
type PECLInstallResult struct {
	Name, Version, InstallDir string
}

// ServerRunOptions configures one webserver run.
type ServerRunOptions struct {
	Server, ServerVersion, PHPVersion, SAPI, MPM string
	Host                                         string
	Port                                         int
	Dir, Config, FPMConf, PHPIni                 string
	PHPBin, FPMBin, ServerBin                    string
	Verbose, Detach, Reload                      bool
	Jobs                                         int
}
