package disk

import (
	"fmt"
	"os"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
)

func (s *bundlerRepository) buildPHP(name, version string, ldPath, cppFlags, ldFlags []string, forceCompiler string) error {
	check, err := s.advisorSvc.Check(name, version, "")
	if err != nil {
		return err
	}

	if check.Action == "skip" {
		fmt.Printf("✓ PHP %s is already installed at %s\n", version, utils.PHPOutputPath(s.silo, version))
		return nil
	}

	fmt.Printf("Building PHP %s...\n", version)

	pat, err := s.patternRegistry.MatchPatternByType(name, check.SourceType, "linux", "x86_64", utils.ParseVersion(version))
	if err != nil {
		return err
	}

	urls, err := pattern.BuildURLs(pat, utils.ParseVersion(version))
	if err != nil {
		return fmt.Errorf("failed to build URL for PHP: %w", err)
	}

	archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
	if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
		return fmt.Errorf("failed to download PHP: %w", err)
	}

	sourceDir := utils.GetSourceDirPath(s.silo, name, version)
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return fmt.Errorf("failed to create source directory: %w", err)
	}

	if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
		return fmt.Errorf("failed to extract PHP source: %w", err)
	}

	installDir := utils.PHPOutputPath(s.silo, version)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	configureFlags := s.forgeSvc.GetPHPConfigureFlags(version, nil)

	cc, cflags, cxx, err := s.getCompilerForVersion(version, forceCompiler)
	if err != nil {
		return err
	}

	config := domain.ForgeConfig{
		Name:            name,
		Version:         version,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
		ConfigureFlags:  configureFlags,
		CC:              cc,
		CFLAGS:          cflags,
		CXX:             cxx,
		Verbose:         s.verbose,
	}

	_, err = s.forgeSvc.Build(config, sourceDir)
	if err != nil {
		fmt.Printf("✗ Failed to build PHP %s: %v\n", version, err)
		return err
	}

	fmt.Printf("✓ PHP %s installed successfully\n", version)
	return nil
}
