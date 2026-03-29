package disk

import (
	"path/filepath"
	"strings"
)

func extractVersion(fullVersion string) string {
	if idx := strings.Index(fullVersion, "|"); idx != -1 {
		return fullVersion[:idx]
	}
	return fullVersion
}

func archivePathFromURL(root, pkg, ver, url string) string {
	filename := filepath.Base(url)
	return filepath.Join(root, "cache", pkg, ver, filename)
}
