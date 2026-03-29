package disk

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
)

func (s *bundlerRepository) buildPackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	check, err := s.advisorSvc.Check(name, version)
	if err != nil {
		return err
	}

	if check.SystemAvailable {
		fmt.Printf("Using system %s@%s at %s\n", name, version, check.SystemPath)
		return nil
	}

	switch check.Action {
	case "skip":
		return nil
	case "download":
		url, err := s.patternRegistry.BuildURLByType(name, version, check.SourceType)
		if err != nil {
			return err
		}
		archive := archivePathFromURL(s.silo.Root, name, version, url)
		if _, err := s.downloadSvc.Download(url, archive); err != nil {
			fmt.Printf("Binary download failed for %s@%s, falling back to source build\n", name, version)
			return s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags)
		}
		fallthrough
	case "extract":
		url, err := s.patternRegistry.BuildURLByType(name, version, check.SourceType)
		if err != nil {
			return err
		}
		archive := archivePathFromURL(s.silo.Root, name, version, url)
		dest := s.silo.GetSourceDirPath(name, version)
		if _, err := s.unloadSvc.Unpack(archive, dest); err != nil {
			return err
		}
		fallthrough
	case "build", "rebuild":
		return s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags)
	}
	return fmt.Errorf("unknown action %q for %s@%s", check.Action, name, version)
}

func (s *bundlerRepository) buildFromSource(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	sources, err := s.sourceSvc.GetSources(name, version)
	if err != nil {
		return fmt.Errorf("failed to get sources for %s@%s: %w", name, version, err)
	}

	var lastErr error
	for _, src := range sources {
		archive := archivePathFromURL(s.silo.Root, name, version, src.URL)
		if _, err := s.downloadSvc.Download(src.URL, archive); err != nil {
			lastErr = err
			fmt.Printf("Download failed for %s@%s from %s, trying next mirror...\n", name, version, src.URL)
			continue
		}

		sourceDir := s.silo.GetSourceDirPath(name, version)
		if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
			lastErr = err
			fmt.Printf("Extraction failed for %s@%s, trying next mirror...\n", name, version)
			continue
		}

		return s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags)
	}

	if lastErr != nil {
		return fmt.Errorf("all mirrors failed for %s@%s: %w", name, version, lastErr)
	}
	return nil
}

func (s *bundlerRepository) buildFromSourceOrSystem(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	err := s.buildFromSource(name, version, phpVersion, ldPath, cppFlags, ldFlags)
	if err == nil {
		return nil
	}

	check, checkErr := s.advisorSvc.Check(name, version)
	if checkErr != nil {
		return fmt.Errorf("download failed: %w, system check also failed: %v", err, checkErr)
	}

	if check.SystemAvailable {
		fmt.Printf("Using system %s@%s at %s (build from source failed: %v)\n", name, version, check.SystemPath, err)
		return nil
	}

	return err
}

func (s *bundlerRepository) compilePackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	installDir := s.silo.DependencyPath(phpVersion, name, version)

	config := domain.ForgeConfig{
		Name:            name,
		Version:         version,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
		ConfigureFlags:  s.forgeSvc.GetConfigureFlags(name),
	}

	_, err := s.forgeSvc.Build(config)
	return err
}
