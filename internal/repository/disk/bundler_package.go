package disk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (s *bundlerRepository) buildPackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, contextMsg string, isBuildTool bool) error {
	check, err := s.advisorSvc.Check(name, version, phpVersion)
	if err != nil {
		return err
	}

	if check.SystemAvailable {
		if isBuildTool {
			fmt.Printf("  ✓ %s@%s (system)%s\n", name, version, contextMsg)
		} else {
			fmt.Printf("✓ %s@%s at %s%s\n", name, version, check.SystemPath, contextMsg)
		}
		return nil
	}

	switch check.Action {
	case "skip":
		fmt.Printf("✓ %s@%s already installed%s\n", name, version, contextMsg)
		return nil
	case "download":
		fmt.Printf("Installing %s@%s%s...\n", name, version, contextMsg)
		url, err := s.patternRegistry.BuildURLByType(name, version, check.SourceType)
		if err != nil {
			return err
		}
		archive := archivePathFromURL(s.silo.Root, name, version, url)
		if _, err := s.downloadSvc.Download(url, archive); err != nil {
			fmt.Printf("  Binary download failed, falling back to source build\n")
			return s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags, check.Suggestion)
		}
		fallthrough
	case "extract":
		archive := s.findCachedArchive(name, version)
		if archive == "" {
			return fmt.Errorf("no cached archive for %s@%s", name, version)
		}
		dest := utils.GetSourceDirPath(s.silo, name, version)
		if _, err := s.unloadSvc.Unpack(archive, dest); err != nil {
			return err
		}
		fallthrough
	case "build", "rebuild":
		err := s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags)
		if err != nil {
			fmt.Printf("✗ Failed to build %s@%s: %v\n", name, version, err)
			if check.Suggestion != "" {
				fmt.Printf("\n💡 Tip: Install system package to avoid building from source:\n   %s\n\n", check.Suggestion)
			}
			return err
		}
		fmt.Printf("✓ Successfully installed %s@%s%s\n", name, version, contextMsg)
		return nil
	}
	return fmt.Errorf("unknown action %q for %s@%s", check.Action, name, version)
}

func (s *bundlerRepository) findCachedArchive(pkg, ver string) string {
	cacheDir := filepath.Join(s.silo.Root, "cache", pkg, ver)
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.Name() != "archive" && !entry.IsDir() {
			return filepath.Join(cacheDir, entry.Name())
		}
	}
	return filepath.Join(cacheDir, "archive")
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

		sourceDir := utils.GetSourceDirPath(s.silo, name, version)
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

func (s *bundlerRepository) buildFromSourceOrSystem(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, suggestion string) error {
	err := s.buildFromSource(name, version, phpVersion, ldPath, cppFlags, ldFlags)
	if err == nil {
		return nil
	}

	check, checkErr := s.advisorSvc.Check(name, version, phpVersion)
	if checkErr != nil {
		return fmt.Errorf("download failed: %w, system check also failed: %v", err, checkErr)
	}

	if check.SystemAvailable {
		fmt.Printf("Using system %s@%s at %s (build from source failed: %v)\n", name, version, check.SystemPath, err)
		return nil
	}

	if suggestion != "" {
		fmt.Printf("\n💡 Tip: Install system package to avoid building from source:\n   %s\n\n", suggestion)
	}

	return err
}

func (s *bundlerRepository) compilePackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	installDir := utils.DependencyPath(s.silo, phpVersion, name, version)

	config := domain.ForgeConfig{
		Name:            name,
		Version:         version,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
		ConfigureFlags:  s.forgeSvc.GetConfigureFlags(name),
		Verbose:         s.verbose,
	}

	_, err := s.forgeSvc.Build(config)
	return err
}
