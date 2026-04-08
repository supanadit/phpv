package disk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
)

func (s *bundlerRepository) logInfo(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info(msg, args...)
	}
}

func (s *bundlerRepository) logWarn(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Warn(msg, args...)
	}
}

func (s *bundlerRepository) logError(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error(msg, args...)
	}
}

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
		zigVersion := "0.14.0"
		if v.Major < 7 {
			zigVersion = "0.13.0"
		}
		s.logInfo("Installing zig@%s (required for PHP %s)...", zigVersion, phpVersion)
		if err := s.installBuildTool("zig", zigVersion, phpVersion); err != nil {
			return "", nil, "", fmt.Errorf("[bundler] failed to install zig: %w", err)
		}
		zigBinary = utils.GetZigCompilerPath(s.silo.Root, phpVersion)
	} else {
		if err := s.siloRepo.IncrementBuildToolRef("zig", filepath.Base(filepath.Dir(zigBinary)), phpVersion); err != nil {
			s.logWarn("Warning: failed to increment zig ref: %v", err)
		}
	}

	return zigBinary + " cc", []string{"-fPIC", "-Wno-error"}, zigBinary + " c++", nil
}

func (s *bundlerRepository) installBuildTool(name, version, phpVersion string) error {
	pat, err := s.patternRegistry.MatchPatternByType(name, domain.SourceTypeBinary, utils.GetOS(), utils.GetArch(), utils.ParseVersion(version))
	if err != nil {
		return err
	}

	urls, err := pattern.BuildURLs(pat, utils.ParseVersion(version))
	if err != nil {
		return fmt.Errorf("[bundler] failed to build URL for %s@%s: %w", name, version, err)
	}

	installPath := filepath.Join(s.silo.Root, "build-tools", name, version)

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			return fmt.Errorf("[download] failed to download %s@%s: %w", name, version, err)
		}

		if err := os.MkdirAll(installPath, 0755); err != nil {
			return fmt.Errorf("[bundler] failed to create directory for %s@%s: %w", name, version, err)
		}

		if _, err := s.unloadSvc.Unpack(archive, installPath); err != nil {
			return fmt.Errorf("[unload] failed to extract %s@%s: %w", name, version, err)
		}
	}

	if name == "zig" {
		zigBinary := filepath.Join(installPath, "zig")
		if err := os.Chmod(zigBinary, 0755); err != nil {
			return fmt.Errorf("[bundler] failed to chmod zig binary: %w", err)
		}
	}

	if err := s.siloRepo.IncrementBuildToolRef(name, version, phpVersion); err != nil {
		return fmt.Errorf("[bundler] failed to increment build-tool ref: %w", err)
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
			s.logInfo("  ✓ %s@%s (system)%s", name, version, contextMsg)
		} else {
			s.logInfo("✓ %s@%s at %s%s", name, version, check.SystemPath, contextMsg)
		}
		return nil
	}

	if check.SystemAvailable && mustBuildFromSource(name, phpVersion) {
		s.logInfo("System %s available but PHP %s requires building from source", name, phpVersion)
	}

	switch check.Action {
	case "skip":
		s.logInfo("✓ %s@%s already installed%s", name, version, contextMsg)
		return nil
	case "download":
		s.logInfo("Installing %s@%s%s...", name, version, contextMsg)
		pat, err := s.patternRegistry.MatchPatternByType(name, check.SourceType, utils.GetOS(), utils.GetArch(), utils.ParseVersion(version))
		if err != nil {
			return err
		}
		urls, err := pattern.BuildURLs(pat, utils.ParseVersion(version))
		if err != nil {
			return fmt.Errorf("[bundler] failed to build URL for %s@%s: %w", name, version, err)
		}
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			s.logWarn("  Download failed, falling back to source build")
			return s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags, check.Suggestion, forceCompiler)
		}
		fallthrough
	case "extract":
		archive := s.findCachedArchive(name, version)
		if archive == "" {
			return fmt.Errorf("[bundler] no cached archive for %s@%s", name, version)
		}
		dest := utils.GetSourceDirPath(s.silo, name, version)
		if _, err := s.unloadSvc.Unpack(archive, dest); err != nil {
			return fmt.Errorf("[unload] failed to extract %s@%s: %w", name, version, err)
		}
		fallthrough
	case "build", "rebuild":
		err := s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, forceCompiler)
		if err != nil {
			s.logError("✗ Failed to build %s@%s: %v", name, version, err)
			if check.Suggestion != "" {
				s.logWarn("\n💡 Tip: Install system package to avoid building from source:\n   %s\n\n", check.Suggestion)
			}
			return err
		}
		s.logInfo("✓ Successfully installed %s@%s%s", name, version, contextMsg)
		return nil
	}
	return fmt.Errorf("[bundler] unknown action %q for %s@%s", check.Action, name, version)
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
		return fmt.Errorf("[bundler] failed to get sources for %s@%s: %w", name, version, err)
	}

	var lastErr error
	for _, src := range sources {
		archive := archivePathFromURL(s.silo.Root, name, version, src.URL)
		if _, err := s.downloadSvc.Download(src.URL, archive); err != nil {
			lastErr = err
			s.logWarn("Download failed for %s@%s from %s, trying next mirror...", name, version, src.URL)
			continue
		}

		sourceDir := utils.GetSourceDirPath(s.silo, name, version)
		if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
			lastErr = err
			s.logWarn("Extraction failed for %s@%s, trying next mirror...", name, version)
			continue
		}

		return s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, forceCompiler)
	}

	if lastErr != nil {
		return fmt.Errorf("[download] all mirrors failed for %s@%s: %w", name, version, lastErr)
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
		return fmt.Errorf("[download] download failed: %w, system check also failed: %v", err, checkErr)
	}

	if check.SystemAvailable {
		s.logInfo("Using system %s@%s at %s (build from source failed: %v)", name, version, check.SystemPath, err)
		return nil
	}

	if suggestion != "" {
		s.logWarn("\n💡 Tip: Install system package to avoid building from source:\n   %s\n\n", suggestion)
	}

	return err
}

func (s *bundlerRepository) compilePackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, forceCompiler string) error {
	installDir := utils.DependencyPath(s.silo, phpVersion, name, version)
	sourceDir := utils.GetSourceDirPath(s.silo, name, version)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("[forge] failed to create install directory: %w", err)
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

func (s *bundlerRepository) buildPackageWithInfo(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, contextMsg string, isBuildTool bool, forceCompiler string) (*domain.DependencyInfo, error) {
	check, err := s.advisorSvc.Check(name, version, phpVersion)
	if err != nil {
		return nil, err
	}

	depInfo := &domain.DependencyInfo{
		Name:            name,
		Version:         version,
		BuiltFromSource: true,
	}

	if check.SystemAvailable && !mustBuildFromSource(name, phpVersion) {
		depInfo.BuiltFromSource = false
		depInfo.SystemPath = check.SystemPath
		if isBuildTool {
			s.logInfo("  ✓ %s@%s (system)%s", name, version, contextMsg)
		} else {
			s.logInfo("✓ %s@%s at %s%s", name, version, check.SystemPath, contextMsg)
		}
		return depInfo, nil
	}

	if check.SystemAvailable && mustBuildFromSource(name, phpVersion) {
		s.logInfo("System %s available but PHP %s requires building from source", name, phpVersion)
	}

	switch check.Action {
	case "skip":
		s.logInfo("✓ %s@%s already installed%s", name, version, contextMsg)
		return depInfo, nil
	case "download":
		s.logInfo("Installing %s@%s%s...", name, version, contextMsg)
		pat, err := s.patternRegistry.MatchPatternByType(name, check.SourceType, utils.GetOS(), utils.GetArch(), utils.ParseVersion(version))
		if err != nil {
			return nil, err
		}
		urls, err := pattern.BuildURLs(pat, utils.ParseVersion(version))
		if err != nil {
			return nil, fmt.Errorf("[bundler] failed to build URL for %s@%s: %w", name, version, err)
		}
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			s.logWarn("  Download failed, falling back to source build")
			return nil, s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags, check.Suggestion, forceCompiler)
		}
		fallthrough
	case "extract":
		archive := s.findCachedArchive(name, version)
		if archive == "" {
			return nil, fmt.Errorf("[bundler] no cached archive for %s@%s", name, version)
		}
		dest := utils.GetSourceDirPath(s.silo, name, version)
		if _, err := s.unloadSvc.Unpack(archive, dest); err != nil {
			return nil, fmt.Errorf("[unload] failed to extract %s@%s: %w", name, version, err)
		}
		fallthrough
	case "build", "rebuild":
		err := s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, forceCompiler)
		if err != nil {
			s.logError("✗ Failed to build %s@%s: %v", name, version, err)
			if check.Suggestion != "" {
				s.logWarn("\n💡 Tip: Install system package to avoid building from source:\n   %s\n\n", check.Suggestion)
			}
			return nil, err
		}
		s.logInfo("✓ Successfully installed %s@%s%s", name, version, contextMsg)
		return depInfo, nil
	}
	return nil, fmt.Errorf("[bundler] unknown action %q for %s@%s", check.Action, name, version)
}
