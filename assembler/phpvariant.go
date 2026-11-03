package assembler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/internal/source"
)

// preparePHPSource materializes a pristine PHP source tree for a SAPI variant
// build: extract the cached tarball into work/src and re-apply published
// patches. Reusing the CLI build's configured tree would carry stale
// .deps/.lo state that breaks make.
func preparePHPSource(deps *Service, phpVersion, work string) error {
	if source.Find(filepath.Join(work, "src"), "php") != "" {
		return nil // already extracted; resume
	}
	// A partial/interrupted extraction leaves a non-empty src dir that
	// Extract would skip (it only checks for content), so clean it first.
	_ = os.RemoveAll(work)
	reg, err := deps.reg.Get("php", phpVersion)
	if err != nil {
		return fmt.Errorf("resolve php %s archive: %w", phpVersion, err)
	}
	archivePath := filepath.Join(cacheDir(), filepath.Base(reg.URL))
	if _, err := os.Stat(archivePath); err != nil {
		if _, err := deps.silo.DownloadURL(reg.URL, reg.ChecksumType, reg.ChecksumValue); err != nil {
			return fmt.Errorf("download php %s for variant build: %w", phpVersion, err)
		}
	}
	if err := os.RemoveAll(work); err != nil {
		return fmt.Errorf("clean variant work dir: %w", err)
	}
	if _, err := deps.silo.Extract(archivePath, filepath.Join(work, "src")); err != nil {
		return fmt.Errorf("extract php source for variant build: %w", err)
	}
	srcPath := source.Find(filepath.Join(work, "src"), "php")
	if srcPath == "" {
		return fmt.Errorf("php %s source not found after extract in %s", phpVersion, work)
	}
	if _, err := deps.patcher.Prepare("php", phpVersion, srcPath); err != nil {
		return fmt.Errorf("patch php source for variant build: %w", err)
	}
	return nil
}

// variantSourcePath finds the extracted PHP source dir inside a work dir.
func variantSourcePath(work string) (string, error) {
	p := source.Find(filepath.Join(work, "src"), "php")
	if p == "" {
		return "", fmt.Errorf("php source not found under %s", work)
	}
	return p, nil
}
