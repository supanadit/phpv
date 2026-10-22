// Package source locates the actual buildable source directory inside an
// extracted package archive.
package source

import (
	"os"
	"path/filepath"
	"strings"
)

// Find returns the directory under extractDir that holds the build files for
// the package named name. It returns "" when no such directory exists.
//
// Tarballs may extract directly into extractDir, into a single top-level
// directory, or (for ICU) into icu/source.
func Find(extractDir, name string) string {
	if hasBuildFile(extractDir) {
		return extractDir
	}
	// ICU tarballs extract to icu/source/ (nested one level deeper).
	if name == "icu" {
		icuSource := filepath.Join(extractDir, "icu", "source")
		if hasBuildFile(icuSource) {
			return icuSource
		}
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return ""
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 1 {
		candidate := filepath.Join(extractDir, dirs[0])
		if hasBuildFile(candidate) {
			return candidate
		}
	}
	for _, d := range dirs {
		candidate := filepath.Join(extractDir, d)
		if hasBuildFile(candidate) {
			return candidate
		}
	}
	return ""
}

func hasBuildFile(dir string) bool {
	return fileExists(filepath.Join(dir, "configure")) ||
		fileExists(filepath.Join(dir, "Configure")) ||
		fileExists(filepath.Join(dir, "CMakeLists.txt")) ||
		fileExists(filepath.Join(dir, "Makefile")) ||
		fileExists(filepath.Join(dir, "makefile"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
