package pecl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var peclNameVersionRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)-(\d+\.\d+\.\d+(?:[a-zA-Z0-9._-]*)?)`)

func parseNameVersion(archivePath string) (name, version string, err error) {
	base := filepath.Base(archivePath)
	base = strings.TrimSuffix(base, ".tar.gz")
	base = strings.TrimSuffix(base, ".tar.bz2")
	base = strings.TrimSuffix(base, ".tgz")

	matches := peclNameVersionRegex.FindStringSubmatch(base)
	if len(matches) >= 3 {
		return matches[1], matches[2], nil
	}

	parts := strings.Split(base, "-")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}

	return base, "unknown", nil
}

func extractArchive(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create extract dir: %w", err)
	}

	cmd := exec.Command("tar", "-xzf", archivePath, "-C", destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract %s: %w\n%s", archivePath, err, out)
	}
	return nil
}

func findSourceDir(baseDir, extName string) string {
	hasConfigM4 := func(dir string) bool {
		_, err := os.Stat(filepath.Join(dir, "config.m4"))
		return err == nil
	}

	if hasConfigM4(baseDir) {
		return baseDir
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() && (entry.Name() == extName || strings.Contains(entry.Name(), extName)) {
			sub := filepath.Join(baseDir, entry.Name())
			if hasConfigM4(sub) {
				return sub
			}
			if nested := findSourceDir(sub, extName); nested != "" {
				return nested
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			sub := filepath.Join(baseDir, entry.Name())
			if hasConfigM4(sub) {
				return sub
			}
		}
	}

	return ""
}
