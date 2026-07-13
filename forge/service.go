package forge

// ForgeRepository handles building and installing packages from source.
// Build and Install are separate so you can compile once and install later,
// or re-install without recompiling.
type ForgeRepository interface {
	// Build compiles the package from sourceDir.
	// extraEnv provides additional environment variables (e.g., CFLAGS, PATH).
	// extraConfigureFlags are appended to the package's ./configure command.
	// installPrefix is the absolute path where the package should be installed.
	// Returns the build directory and environment variables needed for installation.
	Build(name string, version string, sourceDir string, extraEnv []string, extraConfigureFlags []string, installPrefix string) (buildDir string, env map[string]string, err error)

	// Install installs a previously built package into prefix.
	Install(name string, version string, buildDir string, prefix string) error
}

type Service struct {
	repo ForgeRepository
}

func NewService(r ForgeRepository) *Service {
	return &Service{repo: r}
}

func (s *Service) Build(name string, version string, sourceDir string, extraEnv []string, extraConfigureFlags []string, installPrefix string) (string, map[string]string, error) {
	return s.repo.Build(name, version, sourceDir, extraEnv, extraConfigureFlags, installPrefix)
}

func (s *Service) Install(name string, version string, buildDir string, prefix string) error {
	return s.repo.Install(name, version, buildDir, prefix)
}
