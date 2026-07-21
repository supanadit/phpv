package repository

import (
	"os"
	"path/filepath"
)

// ResolveCacheDir returns the cache directory path. It reads PHPV_ROOT from
// the environment; when unset it falls back to $HOME/.phpv. The "caches"
// subdirectory is always appended.
func ResolveCacheDir() string {
	root := os.Getenv("PHPV_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		root = filepath.Join(home, ".phpv")
	}
	return filepath.Join(root, "caches")
}
