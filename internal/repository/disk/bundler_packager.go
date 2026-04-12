package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (s *bundlerRepository) buildPackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags, pkgConfigPaths []string, forceCompiler string) (*domain.DependencyInfo, error) {
	check, err := s.advisorSvc.Check(name, version, phpVersion)
	if err != nil {
		return nil, err
	}

	depInfo := &domain.DependencyInfo{
		Name:            name,
		Version:         version,
		BuiltFromSource: true,
	}

	if check.SystemAvailable && check.Action == "skip" {
		depInfo.BuiltFromSource = false
		depInfo.SystemPath = check.SystemPath
		return depInfo, nil
	}

	if check.SystemAvailable && check.Action != "skip" {
		s.logInfo("System %s@%s available but doesn't satisfy constraint %s for PHP %s", name, check.SystemVersion, check.Constraint, phpVersion)
	}

	switch check.Action {
	case "skip":
		return depInfo, nil
	case "download":
		s.logInfo("Downloading %s@%s...", name, version)
		pat, err := s.patternSvc.MatchPatternByType(name, check.SourceType, utils.GetOS(), utils.GetArch(), utils.ParseVersion(version))
		if err != nil {
			return nil, err
		}
		urls, err := s.patternSvc.BuildURLs(pat, utils.ParseVersion(version))
		if err != nil {
			return nil, fmt.Errorf("[bundler] failed to build URL for %s@%s: %w", name, version, err)
		}
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			if check.Action != "skip" {
				s.logWarn("  Download failed (PHP %s requires build from source), trying source build...", phpVersion)
			} else {
				s.logWarn("  Download failed, falling back to source build")
			}
			err := s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags, pkgConfigPaths, check.Suggestion, forceCompiler)
			if err != nil {
				return nil, err
			}
			return depInfo, nil
		}
		fallthrough
	case "extract":
		s.logInfo("Extracting %s@%s...", name, version)
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
		err := s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, pkgConfigPaths, forceCompiler)
		if err != nil {
			s.logError("✗ Failed to build %s@%s: %v", name, version, err)
			if check.Suggestion != "" {
				s.logWarn("\n💡 Tip: Install system package to avoid building from source:\n   %s\n\n", check.Suggestion)
			}
			return nil, err
		}
		s.logInfo("Installing %s@%s", name, version)
		return depInfo, nil
	}
	return nil, fmt.Errorf("[bundler] unknown action %q for %s@%s", check.Action, name, version)
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

func (s *bundlerRepository) buildFromSource(name, version, phpVersion string, ldPath, cppFlags, ldFlags, pkgConfigPaths []string, forceCompiler string) error {
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

		return s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, pkgConfigPaths, forceCompiler)
	}

	if lastErr != nil {
		return fmt.Errorf("[download] all mirrors failed for %s@%s: %w", name, version, lastErr)
	}
	return nil
}

func (s *bundlerRepository) buildFromSourceOrSystem(name, version, phpVersion string, ldPath, cppFlags, ldFlags, pkgConfigPaths []string, suggestion string, forceCompiler string) error {
	err := s.buildFromSource(name, version, phpVersion, ldPath, cppFlags, ldFlags, pkgConfigPaths, forceCompiler)
	if err == nil {
		return nil
	}

	check, checkErr := s.advisorSvc.Check(name, version, phpVersion)
	if checkErr != nil {
		return fmt.Errorf("[download] download failed: %w, system check also failed: %v", err, checkErr)
	}

	if check.SystemAvailable {
		if check.Action == "skip" {
			s.logInfo("Using system %s@%s at %s (build from source failed: %v)", name, version, check.SystemPath, err)
			return nil
		}
		if suggestion != "" {
			s.logWarn("\n💡 %s@%s required by PHP %s but build from source failed.\n   Install system package to avoid building:\n   %s\n\n", name, version, phpVersion, suggestion)
		}
		return fmt.Errorf("[bundler] %s@%s required by PHP %s but build from source failed", name, version, phpVersion)
	}

	if suggestion != "" {
		s.logWarn("\n💡 Tip: Install system package to avoid building from source:\n   %s\n\n", suggestion)
	}

	return err
}

func (s *bundlerRepository) logBuildFlags(installDir string, configureFlags, cppFlags, ldFlags, ldPath, cflags, pkgConfigPaths []string) {
	if len(configureFlags) > 0 {
		s.logInfo("  Flags: %s", strings.Join(configureFlags, " "))
	} else {
		s.logInfo("  Flags: (none)")
	}
	s.logInfo("  Path: %s", installDir)
	if len(cppFlags) > 0 {
		s.logInfo("  CPPFLAGS: %s", strings.Join(cppFlags, " "))
	} else {
		s.logInfo("  CPPFLAGS: (none)")
	}
	if len(ldFlags) > 0 {
		s.logInfo("  LDFLAGS: %s", strings.Join(ldFlags, " "))
	} else {
		s.logInfo("  LDFLAGS: (none)")
	}
	if len(ldPath) > 0 {
		s.logInfo("  LD_LIBRARY_PATH: %s", strings.Join(ldPath, ":"))
	} else {
		s.logInfo("  LD_LIBRARY_PATH: (none)")
	}
	if len(cflags) > 0 {
		s.logInfo("  CFLAGS: %s", strings.Join(cflags, " "))
	} else {
		s.logInfo("  CFLAGS: (none)")
	}
	if len(pkgConfigPaths) > 0 {
		s.logInfo("  PKG_CONFIG_PATH: %s", strings.Join(pkgConfigPaths, ":"))
	} else {
		s.logInfo("  PKG_CONFIG_PATH: (none)")
	}
}

func (s *bundlerRepository) compilePackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags, pkgConfigPaths []string, forceCompiler string) error {
	installDir := utils.DependencyPath(s.silo, phpVersion, name, version)
	sourceDir := utils.GetSourceDirPath(s.silo, name, version)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("[forge] failed to create install directory: %w", err)
	}

	cc, cflags, cxx, err := s.getCompilerForVersion(phpVersion, forceCompiler)
	if err != nil {
		return err
	}

	configureFlags := s.forgeSvc.GetConfigureFlags(name, version)

	s.logBuildFlags(installDir, configureFlags, cppFlags, ldFlags, ldPath, cflags, pkgConfigPaths)

	compilerName := "gcc"
	compilerPath := ""
	if strings.Contains(cc, "zig") {
		compilerName = "zig"
		compilerPath = strings.Split(cc, " ")[0]
	}
	atMsg := ""
	if compilerPath != "" {
		atMsg = " at " + compilerPath
	}
	s.logInfo("  Compiling %s@%s with %s%s", name, version, compilerName, atMsg)

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
		PkgConfigPaths:  pkgConfigPaths,
		Verbose:         s.verbose,
	}

	_, err = s.forgeSvc.Build(config, sourceDir)
	return err
}
