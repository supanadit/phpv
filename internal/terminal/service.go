package terminal

import (
	"github.com/supanadit/phpv/domain"
)

type TerminalService interface {
	Install(version string, compiler string, extensions []string, verbose bool, fresh bool) (domain.Forge, error)
	Use(version string) (*UseResult, error)
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
	GetInitCode(shell string) (string, error)
	GetPHPvRoot() string
	PECLInstall(archivePath string) (*PECLInstallResult, error)
	PECLList() ([]string, error)
	PECLUninstall(name string) error
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
	Version   string
	IsDefault bool
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

type DoctorResultV2 struct {
	Issues   []DoctorIssue
	Warnings []DoctorWarning
	Summary  string
}
