package disk

import (
	"os"
	"path/filepath"
)

// resolveRoot returns $PHPV_ROOT or falls back to $HOME/.phpv.
func resolveRoot() string {
	root := os.Getenv("PHPV_ROOT")
	if root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".phpv")
	}
	return filepath.Join(home, ".phpv")
}

// RootPath returns the phpv storage root.
func RootPath() string {
	return resolveRoot()
}

// CachePath returns the download cache directory.
func CachePath() string {
	return filepath.Join(resolveRoot(), "caches")
}

// SourcesPath returns the extracted source code directory.
func SourcesPath() string {
	return filepath.Join(resolveRoot(), "sources")
}

// SourcePath returns the extracted source directory for a specific package.
func SourcePath(pkg, version string) string {
	return filepath.Join(resolveRoot(), "sources", pkg, version)
}

// VersionPath returns the root directory for a specific PHP version.
func VersionPath(phpVersion string) string {
	return filepath.Join(resolveRoot(), "versions", phpVersion)
}

// PHPOutputPath returns the install prefix for a specific PHP version.
func PHPOutputPath(phpVersion string) string {
	return filepath.Join(resolveRoot(), "versions", phpVersion, "output")
}

// PackagePrefix returns the install prefix for any package.
func PackagePrefix(name, version string) string {
	return filepath.Join(resolveRoot(), "packages", name, version)
}

// BinPath returns the shim directory.
func BinPath() string {
	return filepath.Join(resolveRoot(), "bin")
}

// StatePath returns the state file path for a PHP version.
func StatePath(phpVersion string) string {
	return filepath.Join(resolveRoot(), "versions", phpVersion, ".state")
}

// DefaultPath returns the default version file path.
func DefaultPath() string {
	return filepath.Join(resolveRoot(), "default")
}
