package terminal

import (
	"github.com/supanadit/phpv/domain"
)

type TerminalService interface {
	Install(version string, verbose bool) (domain.Forge, error)
	Use(version string) (string, error)
	SetDefault(version string) error
	GetDefault() (string, error)
	ListInstalled() ([]string, error)
	ListAvailable() ([]domain.Source, error)
	Which() (string, error)
}
