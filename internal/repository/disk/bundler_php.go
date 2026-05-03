package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/flagresolver"
	"github.com/supanadit/phpv/internal/utils"
)

func (s *bundlerRepository) buildPHP(name, version string, extensions []string, ldPath, cppFlags, ldFlags, pkgConfigPaths []string, forceCompiler string, forceRebuild bool) error {
	check, err := s.advisorSvc.Check(name, version, "")
	if err != nil {
		return err
	}

	outputPath := utils.PHPOutputPath(s.silo, version)
	phpBinary := filepath.Join(outputPath, "bin", "php")

	var archive string

	switch check.Action {
	case "skip":
		exists, _ := afero.Exists(s.fs, phpBinary)
		if exists && !forceRebuild {
			s.logInfo("✓ PHP %s is already installed at %s", version, outputPath)
			return nil
		}
		if !exists {
			s.logInfo("PHP binary not found, will rebuild from source %s", version)
		} else {
			s.logInfo("Forcing rebuild of PHP %s with new extension flags", version)
		}
		fallthrough

	case "download":
		if !forceRebuild {
			pat, err := s.patternSvc.MatchPatternByType(name, check.SourceType, utils.GetOS(), utils.GetArch(), utils.ParseVersion(version))
			if err != nil {
				if check.SourceType == domain.SourceTypeBinary && name == "php" {
					pat, err = s.patternSvc.MatchPatternByType(name, domain.SourceTypeSource, utils.GetOS(), utils.GetArch(), utils.ParseVersion(version))
				}
				if err != nil {
					return fmt.Errorf("failed to find URL pattern for %s@%s: %w", name, version, err)
				}
			}

			urls, err := s.patternSvc.BuildURLs(pat, utils.ParseVersion(version))
			if err != nil {
				return fmt.Errorf("failed to build URL for PHP: %w", err)
			}

			archive = archivePathFromURL(s.silo.Root, name, version, urls[0])

			s.logInfo("Downloading PHP %s...", version)
			if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
				return fmt.Errorf("failed to download PHP: %w", err)
			}
		}
		fallthrough

	case "extract":
		if !forceRebuild {
			s.logInfo("Extracting PHP %s...", version)

			if archive == "" {
				archive = s.findCachedArchive(name, version)
				if archive == "" {
					return fmt.Errorf("[bundler] no cached archive for php@%s", version)
				}
			}

			sourceDir := utils.GetSourceDirPath(s.silo, name, version)
			if err := os.MkdirAll(sourceDir, 0o755); err != nil {
				return fmt.Errorf("failed to create source directory: %w", err)
			}

			if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
				return fmt.Errorf("failed to extract PHP source: %w", err)
			}

			s.touchPHPGeneratedFiles(sourceDir)
		}
		fallthrough

	case "build", "rebuild":
		if len(extensions) > 0 {
			if err := s.flagResolverSvc.ValidateExtensions(extensions, version); err != nil {
				return fmt.Errorf("invalid extension: %w", err)
			}

			conflicts, conflictPairs, err := s.flagResolverSvc.CheckExtensionConflicts(extensions)
			if err != nil {
				s.logWarn("Warning: extension conflict detected between %s", conflicts)
				for _, pair := range conflictPairs {
					s.logWarn("  %s conflicts with %s", pair[0], pair[1])
				}
			}
		}

		installDir := utils.PHPOutputPath(s.silo, version)
		if err := os.MkdirAll(installDir, 0o755); err != nil {
			return fmt.Errorf("failed to create install directory: %w", err)
		}

		sourceDir := utils.GetSourceDirPath(s.silo, name, version)
		s.patchIntlConfigM4(sourceDir)
		s.patchScanfFunctionCasts(sourceDir, version)

		configureFlags := s.forgeSvc.GetPHPConfigureFlags(version, extensions)

		configureFlags = s.resolveDependencyFlags("php", version, configureFlags)

		cc, cflags, cxx, zigLdFlags, err := s.getCompilerForVersion(version, forceCompiler)
		if err != nil {
			return err
		}

		if len(zigLdFlags) > 0 {
			for _, f := range zigLdFlags {
				if strings.HasPrefix(f, "/") && strings.HasSuffix(f, ".a") {
					// static libs go to LIBS, not LDFLAGS
				} else {
					ldFlags = append(ldFlags, f)
				}
			}
		}

		// Get compiler standard flags based on PHP version
		stdRule := s.flagResolverSvc.GetCompilerStdRule(version)
		cxxflags := flagresolver.CXXFlagsFromCFlagsWithStd(cflags, true, stdRule)

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
		compileMsg := fmt.Sprintf("  Compiling php@%s with %s%s", version, compilerName, atMsg)
		if len(extensions) > 0 {
			compileMsg += fmt.Sprintf(" and extension %s", strings.Join(extensions, ","))
		}
		s.logInfo("%s", compileMsg)

		// ICU 74+ deprecated icu-config; PHP's configure may fail to
		// detect ICU via pkg-config. Add ICU .so files by full path to
		// LDFLAGS so the linker can't miss them.
		if slices.Contains(extensions, "intl") {
			icuPath := s.resolveICUPath(version)
			if icuPath != "" {
				icuLibDir := filepath.Join(icuPath, "lib")
				for _, lib := range []string{"libicui18n.so", "libicuuc.so", "libicudata.so", "libicuio.so"} {
					fullPath := filepath.Join(icuLibDir, lib)
					if _, err := os.Stat(fullPath); err == nil {
						ldFlags = append(ldFlags, fullPath)
					}
				}
			}
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
			CStd:            stdRule.CStd,
			CXX:             cxx,
			CXXFLAGS:        cxxflags,
			CXXStd:          stdRule.CXXStd,
			PkgConfigPaths:  pkgConfigPaths,
			Verbose:         s.verbose,
		}

		_, err = s.forgeSvc.Build(config, sourceDir)
		if err != nil {
			s.logError("✗ Failed to build PHP %s: %v", version, err)
			return err
		}

		s.logInfo("  installing php@%s", version)
	}

	return nil
}

func (s *bundlerRepository) touchPHPGeneratedFiles(sourceDir string) {
	now := time.Now()
	zendDir := filepath.Join(sourceDir, "Zend")

	generatedFiles := []string{
		filepath.Join(zendDir, "zend_vm_execute.h"),
		filepath.Join(zendDir, "zend_vm_opcodes.c"),
		filepath.Join(zendDir, "zend_vm_opcodes.h"),
		filepath.Join(zendDir, "zend_vm_handlers.h"),
		filepath.Join(zendDir, "zend_vm_trace_handlers.h"),
		filepath.Join(zendDir, "zend_vm_trace_lines.h"),
		filepath.Join(zendDir, "zend_vm_trace_map.h"),
	}

	for _, f := range generatedFiles {
		if _, err := os.Stat(f); err == nil {
			os.Chtimes(f, now, now)
		}
	}

	time.Sleep(time.Second)
	now = time.Now()
	generatorFiles := []string{
		filepath.Join(zendDir, "zend_vm_gen.php"),
		filepath.Join(zendDir, "zend_vm_def.h"),
		filepath.Join(zendDir, "zend_vm_execute.skl"),
	}
	for _, f := range generatorFiles {
		if _, err := os.Stat(f); err == nil {
			os.Chtimes(f, now.Add(-time.Second), now.Add(-time.Second))
		}
	}
}

// patchIntlConfigM4 patches ext/intl/config.m4 to request C++17 instead of C++11.
// Modern ICU (77+) headers require C++14+ features (std::enable_if_t, std::u16string_view,
// etc.). PHP 8.0's intl extension hardcodes PHP_CXX_COMPILE_STDCXX(11, ...), which
// adds -std=c++11 AFTER CXXFLAGS_CLEAN in the Makefile, overriding -std=gnu++17.
// This patch ensures the intl extension requests C++17 so the generated Makefile
// uses -std=c++17 instead, which is compatible with modern ICU.
func (s *bundlerRepository) patchIntlConfigM4(sourceDir string) {
	configPath := filepath.Join(sourceDir, "ext", "intl", "config.m4")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return // intl extension not present or already patched
	}

	content := string(data)
	// Replace PHP_CXX_COMPILE_STDCXX(11, with PHP_CXX_COMPILE_STDCXX(17,
	// This handles both the bare 11 and quoted [11] forms used in different PHP versions.
	if strings.Contains(content, "PHP_CXX_COMPILE_STDCXX(11,") {
		content = strings.ReplaceAll(content, "PHP_CXX_COMPILE_STDCXX(11,", "PHP_CXX_COMPILE_STDCXX(17,")
	} else if strings.Contains(content, "PHP_CXX_COMPILE_STDCXX([11],") {
		content = strings.ReplaceAll(content, "PHP_CXX_COMPILE_STDCXX([11],", "PHP_CXX_COMPILE_STDCXX([17],")
	} else {
		return // already patched or uses a different version
	}

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		s.logWarn("Warning: failed to patch ext/intl/config.m4: %v", err)
		return
	}
	s.logInfo("  Patched ext/intl/config.m4: C++11 -> C++17 for ICU compatibility")
}

// patchScanfFunctionCasts patches ext/standard/scanf.c to fix function pointer
// casts that GCC 15+ (C23) rejects. PHP 8.0's scanf.c declares:
//
//   zend_long (*fn)();
//
// and assigns:
//
//   fn = (zend_long (*)())ZEND_STRTOL_PTR;
//
// then calls:
//
//   (*fn)(buf, NULL, base);
//
// C23 changed () in function declarations from "unspecified params" to
// "no params" (matching C++), so GCC 15+ sees fn as 0-arg and hard-errors
// at the 3-arg call site. -fpermissive does not downgrade this error.
//
// This patch fixes both the declaration and the casts to use the correct
// 3-arg function pointer type: zend_long (*)(const char *, char **, int)
func (s *bundlerRepository) patchScanfFunctionCasts(sourceDir, phpVersion string) {
	// Only PHP 8.0.x is affected; PHP 8.1+ already fixed this upstream.
	v := utils.ParseVersion(phpVersion)
	if v.Major != 8 || v.Minor != 0 {
		return
	}

	scanfPath := filepath.Join(sourceDir, "ext", "standard", "scanf.c")
	data, err := os.ReadFile(scanfPath)
	if err != nil {
		return
	}

	content := string(data)

	// Fix the declaration: zend_long (*fn)() -> zend_long (*fn)(const char *, char **, int)
	// Also handle (void) variant in case some patchlevels differ.
	oldDecl := "zend_long (*fn)()"
	if strings.Contains(content, oldDecl) {
		content = strings.ReplaceAll(content, oldDecl,
			"zend_long (*fn)(const char *, char **, int)")
	} else {
		oldDecl = "zend_long (*fn)(void)"
		if strings.Contains(content, oldDecl) {
			content = strings.ReplaceAll(content, oldDecl,
				"zend_long (*fn)(const char *, char **, int)")
		}
	}

	// Fix the casts: (zend_long (*)()) -> (zend_long (*)(const char *, char **, int))
	// Also handle (void) variant.
	oldCast := "(zend_long (*)())"
	newCast := "(zend_long (*)(const char *, char **, int))"
	if strings.Contains(content, oldCast) {
		content = strings.ReplaceAll(content, oldCast, newCast)
	}
	oldCastVoid := "(zend_long (*)(void))"
	if strings.Contains(content, oldCastVoid) {
		content = strings.ReplaceAll(content, oldCastVoid, newCast)
	}

	// Don't rewrite if nothing changed (already patched).
	if string(data) == content {
		return
	}

	if err := os.WriteFile(scanfPath, []byte(content), 0o644); err != nil {
		s.logWarn("Warning: failed to patch ext/standard/scanf.c: %v", err)
		return
	}
	s.logInfo("  Patched ext/standard/scanf.c: function pointer types for C23/GCC 15+ compatibility")
}
