package disk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
)

func (s *bundlerRepository) needsAlternativeCC(phpVersion string, forceCompiler string) bool {
	if forceCompiler == "zig" {
		return true
	}
	if forceCompiler != "" && forceCompiler != "gcc" {
		return false
	}
	v := utils.ParseVersion(phpVersion)
	if v.Major < 8 {
		return true
	}
	if v.Major == 8 && v.Minor == 0 {
		return true
	}
	return false
}

func (s *bundlerRepository) getCompilerForVersion(phpVersion string, forceCompiler string) (cc string, cflags []string, cxx string, err error) {
	if !s.needsAlternativeCC(phpVersion, forceCompiler) {
		return "", []string{}, "", nil
	}

	if zigPath := os.Getenv("PHPV_ZIG_PATH"); zigPath != "" {
		if _, err := os.Stat(zigPath); err == nil {
			return zigPath + " cc", []string{"-fPIC", "-Wno-error"}, zigPath + " c++", nil
		}
	}

	zigBinary := utils.GetZigCompilerPath(s.silo.Root, phpVersion)

	if _, err := os.Stat(zigBinary); os.IsNotExist(err) {
		v := utils.ParseVersion(phpVersion)
		zigVersion := "0.15.2"
		if v.Major < 7 {
			zigVersion = "0.13.0"
		}
		fmt.Printf("Installing zig@%s (required for PHP %s)...\n", zigVersion, phpVersion)
		if err := s.installBuildTool("zig", zigVersion, phpVersion); err != nil {
			return "", nil, "", fmt.Errorf("failed to install zig: %w", err)
		}
		zigBinary = utils.GetZigCompilerPath(s.silo.Root, phpVersion)
	} else {
		if err := s.siloRepo.IncrementBuildToolRef("zig", filepath.Base(filepath.Dir(zigBinary)), phpVersion); err != nil {
			fmt.Printf("Warning: failed to increment zig ref: %v\n", err)
		}
	}

	return zigBinary + " cc", []string{"-fPIC", "-Wno-error"}, zigBinary + " c++", nil
}

func (s *bundlerRepository) installBuildTool(name, version, phpVersion string) error {
	pat, err := s.patternRegistry.MatchPatternByType(name, domain.SourceTypeBinary, "linux", "x86_64", utils.ParseVersion(version))
	if err != nil {
		return err
	}

	urls, err := pattern.BuildURLs(pat, utils.ParseVersion(version))
	if err != nil {
		return fmt.Errorf("failed to build URL for %s@%s: %w", name, version, err)
	}

	installPath := filepath.Join(s.silo.Root, "build-tools", name, version)

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			return fmt.Errorf("failed to download %s@%s: %w", name, version, err)
		}

		if err := os.MkdirAll(installPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s@%s: %w", name, version, err)
		}

		if _, err := s.unloadSvc.Unpack(archive, installPath); err != nil {
			return fmt.Errorf("failed to extract %s@%s: %w", name, version, err)
		}
	}

	if name == "zig" {
		zigBinary := filepath.Join(installPath, "zig")
		if err := os.Chmod(zigBinary, 0755); err != nil {
			return fmt.Errorf("failed to chmod zig binary: %w", err)
		}
	}

	if err := s.siloRepo.IncrementBuildToolRef(name, version, phpVersion); err != nil {
		return fmt.Errorf("failed to increment build-tool ref: %w", err)
	}

	return nil
}

func (s *bundlerRepository) buildPackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, contextMsg string, isBuildTool bool, forceCompiler string) error {
	check, err := s.advisorSvc.Check(name, version, phpVersion)
	if err != nil {
		return err
	}

	if check.SystemAvailable && !mustBuildFromSource(name, phpVersion) {
		if isBuildTool {
			fmt.Printf("  ✓ %s@%s (system)%s\n", name, version, contextMsg)
		} else {
			fmt.Printf("✓ %s@%s at %s%s\n", name, version, check.SystemPath, contextMsg)
		}
		return nil
	}

	if check.SystemAvailable && mustBuildFromSource(name, phpVersion) {
		fmt.Printf("System %s available but PHP %s requires building from source\n", name, phpVersion)
	}

	switch check.Action {
	case "skip":
		fmt.Printf("✓ %s@%s already installed%s\n", name, version, contextMsg)
		return nil
	case "download":
		fmt.Printf("Installing %s@%s%s...\n", name, version, contextMsg)
		pat, err := s.patternRegistry.MatchPatternByType(name, check.SourceType, "linux", "x86_64", utils.ParseVersion(version))
		if err != nil {
			return err
		}
		urls, err := pattern.BuildURLs(pat, utils.ParseVersion(version))
		if err != nil {
			return fmt.Errorf("failed to build URL for %s@%s: %w", name, version, err)
		}
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			fmt.Printf("  Download failed, falling back to source build\n")
			return s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags, check.Suggestion, forceCompiler)
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
		err := s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, forceCompiler)
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

func (s *bundlerRepository) buildFromSource(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, forceCompiler string) error {
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

		return s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, forceCompiler)
	}

	if lastErr != nil {
		return fmt.Errorf("all mirrors failed for %s@%s: %w", name, version, lastErr)
	}
	return nil
}

func (s *bundlerRepository) buildFromSourceOrSystem(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, suggestion string, forceCompiler string) error {
	err := s.buildFromSource(name, version, phpVersion, ldPath, cppFlags, ldFlags, forceCompiler)
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

func (s *bundlerRepository) compilePackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, forceCompiler string) error {
	installDir := utils.DependencyPath(s.silo, phpVersion, name, version)
	sourceDir := utils.GetSourceDirPath(s.silo, name, version)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	cc, cflags, cxx, err := s.getCompilerForVersion(phpVersion, forceCompiler)
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
		ConfigureFlags:  s.forgeSvc.GetConfigureFlags(name),
		CC:              cc,
		CFLAGS:          cflags,
		CXX:             cxx,
		Verbose:         s.verbose,
	}

	_, err = s.forgeSvc.Build(config, sourceDir)
	return err
}
