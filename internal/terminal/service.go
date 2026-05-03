package terminal

import (
	"github.com/supanadit/phpv/domain"
)

type TerminalService interface {
	Install(version string, compiler string, extensions []string, verbose bool, fresh bool) (domain.Forge, error)
	Rebuild(version string, compiler string, extensions []string, verbose bool) (domain.Forge, error)
	Use(version string) (*UseResult, error)
	UseSystem() (*UseResult, error)
	ShellUse(version string) error
	AutoDetect() (string, error)
	AutoDetectResolve(constraint string) (string, error)
	SetDefault(version string) error
	GetDefault() (string, error)
	ListInstalled() ([]string, error)
	ListAvailable() ([]domain.Source, error)
	Which() (string, error)
	Uninstall(version string) (*UninstallResult, error)
	CleanBuildTools(dryRun bool) (*CleanBuildToolsResult, error)
	Upgrade(constraint string) (*UpgradeResult, error)
	Doctor() (*DoctorResult, error)
	DoctorV2(version string) (*DoctorResultV2, error)
	GetInitCode(shell string) (string, error)
	GetPHPvRoot() string
	PECLInstall(archivePath string) (*PECLInstallResult, error)
	PECLList() ([]string, error)
	PECLUninstall(name string) error
	PharInstall(name string, version string) (*domain.PharResult, error)
	PharUpdate(name string, version string) (*domain.PharResult, error)
	PharRemove(name string) error
	PharList() ([]string, error)
	PharWhich(name string) (string, error)
}

type PECLInstallResult struct {
	Name       string
	Version    string
	InstallDir string
}

type UninstallResult struct {
	Version      string
	RemovedTools []string
	WasDefault   bool
}

type CleanBuildToolsResult struct {
	Removed    []string
	WillRemove []string
	DryRun     bool
}

type UpgradeResult struct {
	FromVersion string
	ToVersion   string
	Forge       domain.Forge
}

type DoctorResult struct {
	Issues   []DoctorIssue
	Warnings []DoctorWarning
}

type DoctorIssue struct {
	Category string
	Message  string
}

type DoctorWarning struct {
	Category string
	Message  string
}

type DoctorCheckItem struct {
	Name       string
	Available  bool
	Version    string
	Category   string // "available", "autodownload", "buildable", "system"
	Suggestion string
}

type DoctorExtCheck struct {
	Extension   string
	Flag        string
	Package     string
	Status      string // "builtin", "system", "build", "mismatch", "missing"
	SystemVer   string
	ExpectedVer string
	Suggestion  string
}

type DoctorPHPInstall struct {
	Version     string
	Installed   bool
	BinaryPath  string
	ConfigFlags string
	EnabledExts []string
}

type DoctorCompilerForVersion struct {
	MajorVersion int
	Compiler     string // "gcc" or "zig"
	Available    bool
	AutoDownload bool // true if this compiler will be auto-downloaded by phpv
}

type DoctorResultV2 struct {
	BuildTools        []DoctorCheckItem
	LibChecks         []DoctorCheckItem
	Extensions        []DoctorExtCheck
	PHPInstall        *DoctorPHPInstall
	Verdict           string // "ready", "minor", "blocked"
	VerdictMsg        string
	HasGcc            bool // gcc available on system
	HasZig            bool // zig available on system
	CompilerByMajor   []DoctorCompilerForVersion
	EffectiveCompiler string // "gcc" or "zig" - the compiler that will actually be used
	QuickFix          string // consolidated install command for all missing deps
	BuildableInfo     string // info about packages that will be built by phpv
	Summary           string
}

type InstallResult struct {
	Version    string
	Forge      domain.Forge
	BinaryPath string
}

type VersionsResult struct {
	Versions   []VersionInfo
	DefaultVer string
}

type VersionInfo struct {
	Version    string
	IsDefault  bool
	IsSystem   bool
	SystemPath string
}

type ListResult struct {
	Versions []string
}

type UseResultV2 struct {
	ExactVersion string
	ShimPath     string
	OutputPath   string
	Message      string
}
