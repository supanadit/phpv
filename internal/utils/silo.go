package utils

import (
	"path/filepath"

	"github.com/supanadit/phpv/domain"
)

func RootPath(silo *domain.Silo) string {
	return silo.Root
}

func CachePath(silo *domain.Silo) string {
	return filepath.Join(silo.Root, "cache")
}

func SourcePath(silo *domain.Silo) string {
	return filepath.Join(silo.Root, "sources")
}

func VersionPath(silo *domain.Silo) string {
	return filepath.Join(silo.Root, "versions")
}

func BinPath(silo *domain.Silo) string {
	return filepath.Join(silo.Root, "bin")
}

func ArchiveKey(pkg, ver string) string {
	return filepath.Join("cache", pkg, ver, "archive")
}

func SourceKey(pkg, ver string) string {
	return filepath.Join("sources", pkg, ver)
}

func VersionKey(pkg, ver string) string {
	return filepath.Join("versions", pkg, ver)
}

func SourceDirKey(pkg, ver string) string {
	return filepath.Join("sources", pkg, ver, "src")
}

func GetArchivePath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, ArchiveKey(pkg, ver))
}

func GetSourcePath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, SourceKey(pkg, ver))
}

func GetVersionPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, VersionKey(pkg, ver))
}

func GetSourceDirPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(silo.Root, SourceDirKey(pkg, ver))
}

func PHPVersionPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(silo.Root, "versions", phpVersion)
}

func PHPOutputPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(PHPVersionPath(silo, phpVersion), "output")
}

func DependencyPath(silo *domain.Silo, phpVersion, pkg, ver string) string {
	return filepath.Join(PHPVersionPath(silo, phpVersion), "dependency", pkg, ver)
}

func DependencyRootPath(silo *domain.Silo, phpVersion string) string {
	return filepath.Join(PHPVersionPath(silo, phpVersion), "dependency")
}

func BuildToolsPath(silo *domain.Silo) string {
	return filepath.Join(silo.Root, "build-tools")
}

func BuildToolPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(BuildToolsPath(silo), pkg, ver)
}

func BuildToolBinPath(silo *domain.Silo, pkg, ver string) string {
	return filepath.Join(BuildToolPath(silo, pkg, ver), "bin")
}
