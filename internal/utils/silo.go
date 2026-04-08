package utils

import (
	"os"
	"path/filepath"
	"runtime"

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

func GetSystemPkgConfigPaths() []string {
	var paths []string

	basePaths := []string{
		"/usr/lib/pkgconfig",
		"/usr/share/pkgconfig",
		"/usr/local/lib/pkgconfig",
		"/usr/local/share/pkgconfig",
		"/opt/homebrew/lib/pkgconfig",
	}

	archSuffix := runtime.GOARCH + "-linux-gnu"

	linuxGnuPaths := []string{
		filepath.Join("/usr/lib", archSuffix, "pkgconfig"),
		filepath.Join("/usr/lib64", archSuffix, "pkgconfig"),
	}

	if runtime.GOOS == "linux" {
		for _, p := range linuxGnuPaths {
			if _, err := os.Stat(p); err == nil {
				paths = append(paths, p)
			}
		}
		paths = append(basePaths, paths...)
	} else {
		paths = basePaths
	}

	return paths
}

func GetZigCompilerPath(siloRoot, phpVersion string) string {
	zigVersion := "0.14.0"
	if v := ParseVersion(phpVersion); v.Major < 7 {
		zigVersion = "0.13.0"
	}
	return filepath.Join(siloRoot, "build-tools", "zig", zigVersion, "zig")
}

func GetOS() string {
	return runtime.GOOS
}

func GetArch() string {
	a := runtime.GOARCH
	switch a {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	}
	return a
}
