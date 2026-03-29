package disk

import (
	"path/filepath"

	"github.com/supanadit/phpv/domain"
)

func (s *bundlerRepository) buildPHP(name, version string, ldPath, cppFlags, ldFlags []string) error {
	installDir := s.silo.PHPOutputPath(version)
	configureFlags := s.forgeSvc.GetPHPConfigureFlags(version, nil)

	config := domain.ForgeConfig{
		Name:            name,
		Version:         version,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
		ConfigureFlags:  configureFlags,
	}

	_, err := s.forgeSvc.Build(config)
	return err
}

func (s *bundlerRepository) siloPHPOutputPath(version string) string {
	return filepath.Join(s.silo.PHPOutputPath(version), "lib")
}
