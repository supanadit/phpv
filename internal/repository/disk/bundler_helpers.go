package disk

import (
	"path/filepath"
	"strings"
)

func extractVersion(fullVersion string) string {
	if before, _, found := strings.Cut(fullVersion, "|"); found {
		return before
	}
	return fullVersion
}

func archivePathFromURL(root, pkg, ver, url string) string {
	filename := filepath.Base(url)
	return filepath.Join(root, "cache", pkg, ver, filename)
}
