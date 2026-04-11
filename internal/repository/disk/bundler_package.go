package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
	"github.com/supanadit/phpv/pattern"
)

func (s *bundlerRepository) logInfo(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info(msg, args...)
	}
}

func getZigTarget() string {
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "aarch64"
	}

	goos := runtime.GOOS
	abi := "-gnu"
	if goos == "darwin" {
		abi = "-macos"
	}

	return goarch + "-" + goos + abi
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
			target := getZigTarget()
			return zigPath + " cc -target " + target, []string{"-fPIC", "-Wno-error", "-fno-sanitize=undefined"}, zigPath + " c++ -target " + target, nil
		}
	}

	zigBinary := utils.GetZigCompilerPath(s.silo.Root, phpVersion)

	if _, err := os.Stat(zigBinary); os.IsNotExist(err) {
		v := utils.ParseVersion(phpVersion)
		zigVersion := "0.14.0"
		if v.Major < 7 {
			zigVersion = "0.13.0"
		}
		if err := s.installBuildTool("zig", zigVersion, phpVersion); err != nil {
			return "", nil, "", fmt.Errorf("[bundler] failed to install zig: %w", err)
		}
		zigBinary = utils.GetZigCompilerPath(s.silo.Root, phpVersion)
	} else {
		if err := s.siloRepo.IncrementBuildToolRef("zig", filepath.Base(filepath.Dir(zigBinary)), phpVersion); err != nil {
			s.logWarn("Warning: failed to increment zig ref: %v", err)
		}
	}

	target := getZigTarget()
	return zigBinary + " cc -target " + target, []string{"-fPIC", "-Wno-error", "-fno-sanitize=undefined"}, zigBinary + " c++ -target " + target, nil
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
	if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
		return fmt.Errorf("[bundler] failed to create build-tools directory: %w", err)
	}
	lockPath := installPath + ".lock"

	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			var lockGone bool
			for i := 0; i < 60; i++ {
				time.Sleep(500 * time.Millisecond)
				if s.isToolInstalled(name, installPath) {
					return s.siloRepo.IncrementBuildToolRef(name, version, phpVersion)
				}
				if _, err := os.Stat(lockPath); os.IsNotExist(err) {
					lockGone = true
					break
				}
			}
			if lockGone && !s.isToolInstalled(name, installPath) {
				os.RemoveAll(installPath)
				return s.installBuildTool(name, version, phpVersion)
			}
		}
		return fmt.Errorf("[bundler] failed to acquire lock for %s@%s: %w", name, version, err)
	}
	defer func() {
		lock.Close()
		os.Remove(lockPath)
	}()

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		s.logInfo("Downloading build tool %s@%s...", name, version)
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			return fmt.Errorf("[download] failed to download %s@%s: %w", name, version, err)
		}

		s.logInfo("Extracting build tool %s@%s...", name, version)
		if err := os.MkdirAll(installPath, 0755); err != nil {
			return fmt.Errorf("[bundler] failed to create directory for %s@%s: %w", name, version, err)
		}

		if _, err := s.unloadSvc.Unpack(archive, installPath); err != nil {
			return fmt.Errorf("[unload] failed to extract %s@%s: %w", name, version, err)
		}

		s.logInfo("Installing build tool %s@%s", name, version)
	}

	if name == "zig" {
		zigBinary := s.findZigBinary(installPath)
		if zigBinary == "" {
			return fmt.Errorf("[bundler] zig binary not found in %s", installPath)
		}
		if err := os.Chmod(zigBinary, 0755); err != nil {
			return fmt.Errorf("[bundler] failed to chmod zig binary: %w", err)
		}
	}

	if err := s.siloRepo.IncrementBuildToolRef(name, version, phpVersion); err != nil {
		return fmt.Errorf("[bundler] failed to increment build-tool ref: %w", err)
	}

	return nil
}

func (s *bundlerRepository) findZigBinary(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subZig := filepath.Join(dir, entry.Name(), "zig")
			if _, err := os.Stat(subZig); err == nil {
				return subZig
			}
			subBin := filepath.Join(dir, entry.Name(), "bin", "zig")
			if _, err := os.Stat(subBin); err == nil {
				return subBin
			}
		}
	}
	directZig := filepath.Join(dir, "zig")
	if _, err := os.Stat(directZig); err == nil {
		return directZig
	}
	directBin := filepath.Join(dir, "bin", "zig")
	if _, err := os.Stat(directBin); err == nil {
		return directBin
	}
	return ""
}

func (s *bundlerRepository) isToolInstalled(name, installPath string) bool {
	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return false
	}

	switch name {
	case "zig":
		return s.findZigBinary(installPath) != ""
	default:
		entries, err := os.ReadDir(installPath)
		if err != nil {
			return false
		}
		return len(entries) > 0
	}
}

func (s *bundlerRepository) buildPackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string, contextMsg string, isBuildTool bool, forceCompiler string) error {
	check, err := s.advisorSvc.Check(name, version, phpVersion)
	if err != nil {
		return err
	}

	if check.SystemAvailable && check.Action == "skip" {
		return nil
	}

	if check.SystemAvailable && check.Action != "skip" {
		s.logInfo("System %s available but PHP %s requires building from source", name, phpVersion)
	}

	switch check.Action {
	case "skip":
		return nil
	case "download":
		s.logInfo("Downloading %s@%s...", name, version)
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
		s.logInfo("Extracting %s@%s...", name, version)
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
		s.logInfo("Installing %s@%s", name, version)
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

	configureFlags := s.forgeSvc.GetConfigureFlags(name, version)

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

	if check.SystemAvailable && check.Action == "skip" {
		depInfo.BuiltFromSource = false
		depInfo.SystemPath = check.SystemPath
		return depInfo, nil
	}

	if check.SystemAvailable && check.Action != "skip" {
		s.logInfo("System %s available but PHP %s requires building from source", name, phpVersion)
	}

	switch check.Action {
	case "skip":
		return depInfo, nil
	case "download":
		s.logInfo("Downloading %s@%s...", name, version)
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
			if check.Action != "skip" {
				s.logWarn("  Download failed (PHP %s requires build from source), trying source build...", phpVersion)
			} else {
				s.logWarn("  Download failed, falling back to source build")
			}
			err := s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags, check.Suggestion, forceCompiler)
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
		err := s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags, forceCompiler)
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
