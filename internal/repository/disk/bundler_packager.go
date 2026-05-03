package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/flagresolver"
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
			s.logWarn("  Archive corrupted or incomplete, removing and re-downloading...")
			if removeErr := s.fs.Remove(archive); removeErr != nil {
				return nil, fmt.Errorf("[bundler] failed to remove corrupt archive: %w", removeErr)
			}
			sources, srcErr := s.sourceSvc.GetSources(name, version)
			if srcErr != nil {
				return nil, fmt.Errorf("[bundler] failed to get sources: %w", srcErr)
			}
			var lastErr error
			for _, src := range sources {
				downloadArchive := archivePathFromURL(s.silo.Root, name, version, src.URL)
				var dlErr error
				if _, dlErr = s.downloadSvc.Download(src.URL, downloadArchive); dlErr != nil {
					lastErr = dlErr
					continue
				}
				var unpackErr error
				if _, unpackErr = s.unloadSvc.Unpack(downloadArchive, dest); unpackErr == nil {
					break
				}
				lastErr = unpackErr
				if removeErr := s.fs.Remove(downloadArchive); removeErr != nil {
					break
				}
			}
			if lastErr != nil {
				return nil, fmt.Errorf("[bundler] failed to re-download and extract %s@%s: %w", name, version, lastErr)
			}
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

func (s *bundlerRepository) logBuildFlags(installDir string, configureFlags, cppFlags, ldFlags, ldPath, cflags, pkgConfigPaths, cxxflags []string, cc, cxx string) {
	if len(configureFlags) > 0 {
		s.logInfo("  Flags: %s", strings.Join(configureFlags, " "))
	} else {
		s.logInfo("  Flags: (none)")
	}
	s.logInfo("  Path: %s", installDir)

	ar, ranlib, nm, ld := "(default)", "(default)", "(default)", "(default)"
	if strings.Contains(cc, "zig") {
		zigBinary := strings.Split(cc, " ")[0]
		wrapperDir := filepath.Join(filepath.Dir(zigBinary), "wrappers")
		ar = filepath.Join(wrapperDir, "ar")
		ranlib = filepath.Join(wrapperDir, "ranlib")
		nm = filepath.Join(wrapperDir, "nm")
		if _, err := os.Stat(filepath.Join(wrapperDir, "ld")); err == nil {
			ld = filepath.Join(wrapperDir, "ld")
		}
	}

	s.logInfo("  AR: %s", ar)
	s.logInfo("  RANLIB: %s", ranlib)
	s.logInfo("  NM: %s", nm)
	s.logInfo("  LD: %s", ld)

	if cc != "" {
		s.logInfo("  CC: %s", cc)
	} else {
		s.logInfo("  CC: (default)")
	}
	if cxx != "" {
		s.logInfo("  CXX: %s", cxx)
	} else {
		s.logInfo("  CXX: (default)")
	}
	if len(cxxflags) > 0 {
		s.logInfo("  CXXFLAGS: %s", strings.Join(cxxflags, " "))
	} else {
		s.logInfo("  CXXFLAGS: (none)")
	}
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
	sourceDir := utils.GetSourceDirPath(s.silo, name, version)

	installDir := s.getInstallDir(name, version, phpVersion)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("[forge] failed to create install directory: %w", err)
	}

	cc, cflags, cxx, zigLdFlags, err := s.getCompilerForVersion(phpVersion, forceCompiler)
	if err != nil {
		return err
	}

	var libs []string
	if len(zigLdFlags) > 0 {
		for _, f := range zigLdFlags {
			if strings.HasPrefix(f, "/") && strings.HasSuffix(f, ".a") {
				libs = append(libs, f)
			} else {
				ldFlags = append(ldFlags, f)
			}
		}
	}

	// Note: system include paths (-I/usr/include etc.) are NOT added to CPPFLAGS
	// because they can conflict with packages that have their own headers (e.g., ICU).
	// Zig cc uses its own bundled libc headers by default. When system headers are
	// needed (e.g., curl finding system OpenSSL), the configure flags like
	// --with-ssl=/usr already provide explicit include paths.

	configureFlags := s.forgeSvc.GetConfigureFlags(name, version)

	configureFlags = s.resolveDependencyFlags(name, phpVersion, configureFlags)

	// When using zig cc with system OpenSSL, zig doesn't search /usr/include
	// by default. We need to explicitly add system include/library paths.
	if strings.Contains(cc, "zig") && name == "curl" {
		for _, flag := range configureFlags {
			if strings.HasPrefix(flag, "--with-ssl=") || strings.HasPrefix(flag, "--with-openssl=") {
				val := flag[strings.Index(flag, "=")+1:]
				if val == "/usr" || val == "/usr/local" {
					cppFlags = append(cppFlags, fmt.Sprintf("-I%s/include", val))
					ldFlags = append(ldFlags, fmt.Sprintf("-L%s/lib", val))
				}
			}
		}
	}

	cxxflags := flagresolver.CXXFlagsFromCFlags(cflags, false)

	if name == "php" && len(ldFlags) > 0 {
		var ccPrefix []string
		var cxxPrefix []string
		for _, flag := range ldFlags {
			if strings.HasPrefix(flag, "-L") || strings.HasPrefix(flag, "-Wl,-rpath-link") {
				ccPrefix = append(ccPrefix, flag)
				cxxPrefix = append(cxxPrefix, flag)
			}
		}
		if len(ccPrefix) > 0 {
			cc = cc + " " + strings.Join(ccPrefix, " ")
			cxx = cxx + " " + strings.Join(cxxPrefix, " ")
		}
	}

	s.logBuildFlags(installDir, configureFlags, cppFlags, ldFlags, ldPath, cflags, pkgConfigPaths, cxxflags, cc, cxx)

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
		PHPVersion:      phpVersion,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
		ConfigureFlags:  configureFlags,
		CC:              cc,
		CFLAGS:          cflags,
		CXX:             cxx,
		CXXFLAGS:        cxxflags,
		PkgConfigPaths:  pkgConfigPaths,
		Verbose:         s.verbose,
		Libs:            libs,
	}

	_, err = s.forgeSvc.Build(config, sourceDir)
	return err
}

var buildToolsList = utils.BuildTools

var systemPrefixes = []string{"/usr", "/usr/local"}

func findSystemPrefix(headerRelPath string) string {
	for _, prefix := range systemPrefixes {
		if _, err := os.Stat(filepath.Join(prefix, headerRelPath)); err == nil {
			return prefix
		}
	}
	return ""
}

func (s *bundlerRepository) resolveDependencyFlags(name, phpVersion string, flags []string) []string {
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag == "--with-openssl" || flag == "--with-ssl" {
			resolved := false
			opensslFlag := "--with-openssl"
			if name != "php" {
				opensslFlag = "--with-ssl"
			}
			deps, err := s.assemblerSvc.GetDependencies("php", phpVersion)
			if err == nil {
				for _, dep := range deps {
					if dep.Name == "openssl" {
						ver := dep.Version
						if idx := strings.Index(ver, "|"); idx != -1 {
							ver = ver[:idx]
						}
						opensslPath := utils.DependencyPath(s.silo, phpVersion, "openssl", ver)
						if fi, err := os.Stat(opensslPath); err == nil && fi.IsDir() {
							includeDir := filepath.Join(opensslPath, "include", "openssl")
							if _, err := os.Stat(includeDir); err == nil {
								result = append(result, opensslFlag+"="+opensslPath)
								resolved = true
							}
						}
						break
					}
				}
			}
			if !resolved {
				depPath := utils.DependencyPath(s.silo, phpVersion, "openssl", "")
				if entries, err := os.ReadDir(depPath); err == nil && len(entries) > 0 {
					for _, entry := range entries {
						candidatePath := filepath.Join(depPath, entry.Name())
						includeDir := filepath.Join(candidatePath, "include", "openssl")
						if _, err := os.Stat(includeDir); err == nil {
							result = append(result, opensslFlag+"="+candidatePath)
							resolved = true
							break
						}
					}
				}
			}
			if !resolved {
				result = append(result, flag)
			}
		} else if flag == "--with-zlib" && (name == "libxml2" || name == "curl") {
			resolved := false
			deps, err := s.assemblerSvc.GetDependencies("php", phpVersion)
			if err == nil {
				for _, dep := range deps {
					if dep.Name == "zlib" {
						ver := dep.Version
						if idx := strings.Index(ver, "|"); idx != -1 {
							ver = ver[:idx]
						}
						zlibPath := utils.DependencyPath(s.silo, phpVersion, "zlib", ver)
						if fi, err := os.Stat(zlibPath); err == nil && fi.IsDir() {
							includeFile := filepath.Join(zlibPath, "include", "zlib.h")
							if _, err := os.Stat(includeFile); err == nil {
								result = append(result, "--with-zlib="+zlibPath)
								resolved = true
							}
						}
						break
					}
				}
			}
			if !resolved {
				depPath := utils.DependencyPath(s.silo, phpVersion, "zlib", "")
				if entries, err := os.ReadDir(depPath); err == nil && len(entries) > 0 {
					for _, entry := range entries {
						candidatePath := filepath.Join(depPath, entry.Name())
						includeFile := filepath.Join(candidatePath, "include", "zlib.h")
						if _, err := os.Stat(includeFile); err == nil {
							result = append(result, "--with-zlib="+candidatePath)
							resolved = true
							break
						}
					}
				}
			}
			if !resolved {
				result = append(result, flag)
			}
		} else if strings.HasPrefix(flag, "--with-libxml") {
			resolved := false
			deps, err := s.assemblerSvc.GetDependencies("php", phpVersion)
			if err == nil {
				for _, dep := range deps {
					if dep.Name == "libxml2" {
						ver := dep.Version
						if idx := strings.Index(ver, "|"); idx != -1 {
							ver = ver[:idx]
						}
						libxml2Path := utils.DependencyPath(s.silo, phpVersion, "libxml2", ver)
						if fi, err := os.Stat(libxml2Path); err == nil && fi.IsDir() {
							pkgConfigPath := filepath.Join(libxml2Path, "lib", "pkgconfig", "libxml-2.0.pc")
							if _, err := os.Stat(pkgConfigPath); err == nil {
								result = append(result, flag+"="+libxml2Path)
								resolved = true
							}
						}
						break
					}
				}
			}
			if !resolved {
				depPath := utils.DependencyPath(s.silo, phpVersion, "libxml2", "")
				if entries, err := os.ReadDir(depPath); err == nil && len(entries) > 0 {
					for _, entry := range entries {
						candidatePath := filepath.Join(depPath, entry.Name())
						pkgConfigPath := filepath.Join(candidatePath, "lib", "pkgconfig", "libxml-2.0.pc")
						if _, err := os.Stat(pkgConfigPath); err == nil {
							result = append(result, flag+"="+candidatePath)
							resolved = true
							break
						}
					}
				}
			}
			if !resolved {
				result = append(result, flag)
			}
		} else if strings.HasPrefix(flag, "--with-curl") || strings.HasPrefix(flag, "--with-curl=") {
			resolved := false
			deps, err := s.assemblerSvc.GetDependencies("php", phpVersion)
			if err == nil {
				for _, dep := range deps {
					if dep.Name == "curl" {
						ver := dep.Version
						if idx := strings.Index(ver, "|"); idx != -1 {
							ver = ver[:idx]
						}
						curlPath := utils.DependencyPath(s.silo, phpVersion, "curl", ver)
						if fi, err := os.Stat(curlPath); err == nil && fi.IsDir() {
							libCurlPath := filepath.Join(curlPath, "lib", "libcurl.so")
							if _, err := os.Stat(libCurlPath); err == nil {
								result = append(result, "--with-curl="+curlPath)
								resolved = true
							}
						}
						break
					}
				}
			}
			if !resolved {
				depPath := utils.DependencyPath(s.silo, phpVersion, "curl", "")
				if entries, err := os.ReadDir(depPath); err == nil && len(entries) > 0 {
					for _, entry := range entries {
						candidatePath := filepath.Join(depPath, entry.Name())
						libCurlPath := filepath.Join(candidatePath, "lib", "libcurl.so")
						if _, err := os.Stat(libCurlPath); err == nil {
							result = append(result, "--with-curl="+candidatePath)
							resolved = true
							break
						}
					}
				}
			}
			if !resolved {
				result = append(result, flag)
			}
		} else if strings.HasPrefix(flag, "--with-pdo-pgsql") {
			if flag == "--with-pdo-pgsql" || flag == "--with-pdo-pgsql=yes" {
				wrapperPath := filepath.Join(s.silo.Root, "versions", phpVersion, "wrapper")
				pgConfigPath := filepath.Join(wrapperPath, "bin", "pg_config")
				if fi, err := os.Stat(pgConfigPath); err == nil && !fi.IsDir() {
					result = append(result, "--with-pdo-pgsql="+wrapperPath)
				} else {
					result = append(result, flag)
				}
			} else {
				result = append(result, flag)
			}
		} else if strings.HasPrefix(flag, "--with-pgsql") {
			if flag == "--with-pgsql" || flag == "--with-pgsql=yes" {
				wrapperPath := filepath.Join(s.silo.Root, "versions", phpVersion, "wrapper")
				pgConfigPath := filepath.Join(wrapperPath, "bin", "pg_config")
				if fi, err := os.Stat(pgConfigPath); err == nil && !fi.IsDir() {
					result = append(result, "--with-pgsql="+wrapperPath)
				} else {
					result = append(result, flag)
				}
			} else {
				result = append(result, flag)
			}
		} else {
			result = append(result, flag)
		}
	}
	return result
}

func (s *bundlerRepository) getInstallDir(name, version, phpVersion string) string {
	if buildToolsList[name] {
		return utils.BuildToolPath(s.silo, name, version)
	}
	return utils.DependencyPath(s.silo, phpVersion, name, version)
}
