package args

import (
	"os"
	"path/filepath"
)

// resolveRoot returns the phpv storage root, mirroring the repository helper
// so the delivery channel does not depend on the repository layer.
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
