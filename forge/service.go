package forge

import (
	"context"
	"strings"
)

// ForgeRepository handles building and installing packages from source.
// Build and Install are separate so you can compile once and install later,
// or re-install without recompiling.
type ForgeRepository interface {
	// Build compiles the package from sourceDir.
	// extraEnv provides additional environment variables (e.g., CFLAGS, PATH).
	// extraConfigureFlags are appended to the package's ./configure command.
	// installPrefix is the absolute path where the package should be installed.
	// verbose enables live build output to terminal.
	// jobs controls make parallelism (0 = auto).
	// Returns the build directory and environment variables needed for installation.
	Build(ctx context.Context, name string, version string, sourceDir string, extraEnv []string, extraConfigureFlags []string, installPrefix string, verbose bool, jobs int) (buildDir string, env map[string]string, err error)

	// Install installs a previously built package into prefix.
	Install(ctx context.Context, name string, version string, buildDir string, prefix string, verbose bool, jobs int) error
}

// Service wraps ForgeRepository and adds value by resolving
// {{prefix}}, {{source}}, and {{dep:NAME}} placeholders in
// extraConfigureFlags before delegating to the repository.
type Service struct {
	repo ForgeRepository
}

// NewService creates a forge service backed by the given repository.
func NewService(r ForgeRepository) *Service {
	return &Service{repo: r}
}

// Build resolves placeholders in extraConfigureFlags before delegating.
func (s *Service) Build(ctx context.Context, name string, version string, sourceDir string, extraEnv []string, extraConfigureFlags []string, installPrefix string, verbose bool, jobs int) (string, map[string]string, error) {
	resolved := resolvePlaceholders(extraConfigureFlags, installPrefix, sourceDir)
	return s.repo.Build(ctx, name, version, sourceDir, extraEnv, resolved, installPrefix, verbose, jobs)
}

// Install delegates to the repository.
func (s *Service) Install(ctx context.Context, name string, version string, buildDir string, prefix string, verbose bool, jobs int) error {
	return s.repo.Install(ctx, name, version, buildDir, prefix, verbose, jobs)
}

// resolvePlaceholders replaces {{prefix}}, {{source}}, and {{dep:NAME}}
// in configure flags with their resolved values.
func resolvePlaceholders(flags []string, prefix, sourceDir string) []string {
	if len(flags) == 0 {
		return flags
	}
	resolved := make([]string, len(flags))
	for i, f := range flags {
		f = strings.ReplaceAll(f, "{{prefix}}", prefix)
		f = strings.ReplaceAll(f, "{{source}}", sourceDir)
		resolved[i] = f
	}
	return resolved
}
