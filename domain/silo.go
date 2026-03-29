package domain

import (
	"path/filepath"
)

type Silo struct {
	Root string
}

func (s Silo) RootPath() string {
	return s.Root
}

func (s Silo) CachePath() string {
	return filepath.Join(s.Root, "cache")
}

func (s Silo) SourcePath() string {
	return filepath.Join(s.Root, "sources")
}

func (s Silo) VersionPath() string {
	return filepath.Join(s.Root, "versions")
}

func (s Silo) BinPath() string {
	return filepath.Join(s.Root, "bin")
}

func (s Silo) ArchiveKey(pkg, ver string) string {
	return filepath.Join("cache", pkg, ver, "archive")
}

func (s Silo) SourceKey(pkg, ver string) string {
	return filepath.Join("sources", pkg, ver)
}

func (s Silo) VersionKey(pkg, ver string) string {
	return filepath.Join("versions", pkg, ver)
}

func (s Silo) SourceDirKey(pkg, ver string) string {
	return filepath.Join("sources", pkg, ver, "src")
}

func (s Silo) GetArchivePath(pkg, ver string) string {
	return filepath.Join(s.Root, s.ArchiveKey(pkg, ver))
}

func (s Silo) GetSourcePath(pkg, ver string) string {
	return filepath.Join(s.Root, s.SourceKey(pkg, ver))
}

func (s Silo) GetVersionPath(pkg, ver string) string {
	return filepath.Join(s.Root, s.VersionKey(pkg, ver))
}

func (s Silo) GetSourceDirPath(pkg, ver string) string {
	return filepath.Join(s.Root, s.SourceDirKey(pkg, ver))
}

func (s Silo) PHPVersionPath(phpVersion string) string {
	return filepath.Join(s.Root, "versions", phpVersion)
}

func (s Silo) PHPOutputPath(phpVersion string) string {
	return filepath.Join(s.PHPVersionPath(phpVersion), "output")
}

func (s Silo) DependencyPath(phpVersion, pkg, ver string) string {
	return filepath.Join(s.PHPVersionPath(phpVersion), "dependency", pkg, ver)
}

func (s Silo) DependencyRootPath(phpVersion string) string {
	return filepath.Join(s.PHPVersionPath(phpVersion), "dependency")
}

func (s Silo) BuildToolsPath() string {
	return filepath.Join(s.Root, "build-tools")
}

func (s Silo) BuildToolPath(pkg, ver string) string {
	return filepath.Join(s.BuildToolsPath(), pkg, ver)
}

func (s Silo) BuildToolBinPath(pkg, ver string) string {
	return filepath.Join(s.BuildToolPath(pkg, ver), "bin")
}
