package assembler

import (
	"github.com/supanadit/phpv/domain"
)

// AssemblerRepository resolves the transitive dependency graph and
// orchestrates the full build pipeline (resolve → download → build → install).
type AssemblerRepository interface {
	GetOrderedDependencies(name string, version string) ([]domain.Dependency, error)
	Assemble(name string, version string, progress ProgressFunc) (*AssemblerResult, error)
}

// AssemblerResult holds the outcome of assembling a package.
type AssemblerResult struct {
	DownloadResults []DownloadResult
	PHPVersion      string
	Prefix          string
	Env             map[string]string
}

// DownloadResult holds the outcome of downloading + extracting a single package.
type DownloadResult struct {
	Name       string
	Version    string
	Downloaded bool
	Extracted  bool
	Err        error
}

// ProgressFunc receives human-readable status updates during assembly.
// Pass nil to disable progress reporting.
type ProgressFunc func(stage, message string)

// Service is a thin pass-through to AssemblerRepository.
type Service struct {
	repo AssemblerRepository
}

func NewService(r AssemblerRepository) *Service {
	return &Service{repo: r}
}

// Assemble runs the full pipeline for (name, version).
func (s *Service) Assemble(name string, version string, progress ProgressFunc) (*AssemblerResult, error) {
	return s.repo.Assemble(name, version, progress)
}

// DownloadFailed returns true if any result has an error.
func DownloadFailed(results []DownloadResult) bool {
	for _, r := range results {
		if r.Err != nil {
			return true
		}
	}
	return false
}
