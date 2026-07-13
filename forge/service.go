package forge

// ForgeRepository handles building and installing packages from source.
// Build and Install are separate so you can compile once and install later,
// or re-install without recompiling.
type ForgeRepository interface {
	Build(name string, version string, sourceDir string, extraEnv []string) (buildDir string, env map[string]string, err error)
	Install(name string, version string, buildDir string, prefix string) error
}

// Service wraps a ForgeRepository and provides the public build/install API.
type Service struct {
	forgeRep ForgeRepository
}

func NewService(fr ForgeRepository) *Service {
	return &Service{forgeRep: fr}
}

// Build compiles the package from sourceDir. extraEnv provides additional
// environment variables (e.g., PATH with build tool bins).
func (s *Service) Build(name string, version string, sourceDir string, extraEnv []string) (buildDir string, env map[string]string, err error) {
	return s.forgeRep.Build(name, version, sourceDir, extraEnv)
}

// Install installs a previously built package into prefix.
func (s *Service) Install(name string, version string, buildDir string, prefix string) error {
	return s.forgeRep.Install(name, version, buildDir, prefix)
}
