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
