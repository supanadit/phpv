package terminal

import (
	"github.com/supanadit/phpv/domain"
)

type TerminalService interface {
	Install(version string, compiler string, verbose bool, fresh bool) (domain.Forge, error)
	Use(version string) (string, error)
	SetDefault(version string) error
	GetDefault() (string, error)
	ListInstalled() ([]string, error)
	ListAvailable() ([]domain.Source, error)
	Which() (string, error)
	Uninstall(version string) (*UninstallResult, error)
	CleanBuildTools(dryRun bool) (*CleanBuildToolsResult, error)
	Upgrade(constraint string) (*UpgradeResult, error)
	Doctor() (*DoctorResult, error)
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
