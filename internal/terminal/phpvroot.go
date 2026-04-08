package terminal

import (
	"os"
	"path/filepath"
)

func GetPHPvRoot() string {
	if root := os.Getenv("PHPV_ROOT"); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(home, ".phpv")
	}
	return filepath.Join(home, ".phpv")
}

func GetBinPath(phpvRoot string) string {
	return filepath.Join(phpvRoot, "bin")
}

func GetVersionsPath(phpvRoot string) string {
	return filepath.Join(phpvRoot, "versions")
}

func GetDefaultFilePath(phpvRoot string) string {
	return filepath.Join(phpvRoot, "default")
}
