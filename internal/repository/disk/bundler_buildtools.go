package disk

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
)

func (s *bundlerRepository) ensureBuildTools() error {
	buildTools := []struct {
		pkg string
		ver string
	}{
		{"m4", "1.4.19"},
		{"autoconf", "2.69"},
		{"autoconf", "2.71"},
		{"autoconf", "2.72"},
		{"automake", "1.16.5"},
		{"automake", "1.17"},
		{"libtool", "2.4.7"},
		{"libtool", "2.5.4"},
		{"perl", "5.38.2"},
		{"bison", "1.35"},
		{"flex", "2.5.39"},
	}

	for _, tool := range buildTools {
		toolPath := s.silo.BuildToolBinPath(tool.pkg, tool.ver)

		if exists, _ := afero.DirExists(s.fs, toolPath); exists {
			continue
		}

		if err := s.installBuildTool(tool.pkg, tool.ver, false); err != nil {
			return fmt.Errorf("failed to install build tool %s@%s: %w", tool.pkg, tool.ver, err)
		}
	}

	return nil
}

func (s *bundlerRepository) installBuildTool(pkg, version string, forceSource bool) error {
	if !forceSource {
		check, err := s.advisorSvc.Check(pkg, version)
		if err != nil {
			return err
		}

		switch check.Action {
		case "skip":
			return nil
		case "download":
			url, err := s.patternRegistry.BuildURLByType(pkg, version, check.SourceType)
			if err != nil {
				return err
			}
			dest := s.silo.GetArchivePath(pkg, version)
			if _, err := s.downloadSvc.Download(url, dest); err != nil {
				return err
			}
			fallthrough
		case "extract":
			archive := s.silo.GetArchivePath(pkg, version)
			sourceDir := s.silo.GetSourceDirPath(pkg, version)
			if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
				return err
			}
			fallthrough
		case "build", "rebuild":
			installDir := s.silo.BuildToolPath(pkg, version)
			config := domain.ForgeConfig{
				Name:    pkg,
				Version: version,
				Prefix:  installDir,
				Jobs:    s.jobs,
			}
			_, err := s.forgeSvc.Build(config)
			return err
		}
		return fmt.Errorf("unknown action %q for build tool %s@%s", check.Action, pkg, version)
	}

	sourceType := domain.SourceTypeSource
	url, err := s.patternRegistry.BuildURLByType(pkg, version, sourceType)
	if err != nil {
		return err
	}
	archive := archivePathFromURL(s.silo.Root, pkg, version, url)
	if _, err := s.downloadSvc.Download(url, archive); err != nil {
		return err
	}

	sourceDir := s.silo.GetSourceDirPath(pkg, version)
	if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
		return err
	}

	installDir := s.silo.BuildToolPath(pkg, version)
	config := domain.ForgeConfig{
		Name:    pkg,
		Version: version,
		Prefix:  installDir,
		Jobs:    s.jobs,
	}
	_, err = s.forgeSvc.Build(config)
	return err
}
