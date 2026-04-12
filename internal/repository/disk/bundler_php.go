package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
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

		configureFlags := s.forgeSvc.GetPHPConfigureFlags(version, extensions)

		cc, cflags, cxx, err := s.getCompilerForVersion(version, forceCompiler)
		if err != nil {
			return err
		}

		s.logBuildFlags(installDir, configureFlags, cppFlags, ldFlags, ldPath, cflags, pkgConfigPaths, cflags, cc, cxx)

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
			CXXFLAGS:        cflags,
			PkgConfigPaths:  pkgConfigPaths,
			Verbose:         s.verbose,
		}

		sourceDir := utils.GetSourceDirPath(s.silo, name, version)
		_, err = s.forgeSvc.Build(config, sourceDir)
		if err != nil {
			s.logError("✗ Failed to build PHP %s: %v", version, err)
			return err
		}

		s.logInfo("  installing php@%s", version)
	}

	return nil
}
