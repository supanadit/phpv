package disk

import (
	"fmt"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (s *bundlerRepository) buildPHP(name, version string, ldPath, cppFlags, ldFlags []string) error {
	check, err := s.advisorSvc.Check(name, version, "")
	if err != nil {
		return err
	}

	if check.Action == "skip" {
		fmt.Printf("✓ PHP %s is already installed at %s\n", version, utils.PHPOutputPath(s.silo, version))
		return nil
	}

	fmt.Printf("Building PHP %s...\n", version)

	installDir := utils.PHPOutputPath(s.silo, version)
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
		Verbose:         s.verbose,
	}

	_, err = s.forgeSvc.Build(config)
	if err != nil {
		fmt.Printf("✗ Failed to build PHP %s: %v\n", version, err)
		return err
	}

	fmt.Printf("✓ PHP %s installed successfully\n", version)
	return nil
}

func (s *bundlerRepository) siloPHPOutputPath(version string) string {
	return filepath.Join(utils.PHPOutputPath(s.silo, version), "lib")
}
