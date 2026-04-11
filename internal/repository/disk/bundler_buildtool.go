package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/utils"
)

func (s *bundlerRepository) installBuildTool(name, version, phpVersion string) error {
	pat, err := s.patternSvc.MatchPatternByType(name, domain.SourceTypeBinary, utils.GetOS(), utils.GetArch(), utils.ParseVersion(version))
	if err != nil {
		return err
	}

	urls, err := s.patternSvc.BuildURLs(pat, utils.ParseVersion(version))
	if err != nil {
		return fmt.Errorf("[bundler] failed to build URL for %s@%s: %w", name, version, err)
	}

	installPath := filepath.Join(s.silo.Root, "build-tools", name, version)
	if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
		return fmt.Errorf("[bundler] failed to create build-tools directory: %w", err)
	}
	lockPath := installPath + ".lock"

	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			var lockGone bool
			for i := 0; i < 60; i++ {
				time.Sleep(500 * time.Millisecond)
				if s.isToolInstalled(name, installPath) {
					return s.siloRepo.IncrementBuildToolRef(name, version, phpVersion)
				}
				if _, err := os.Stat(lockPath); os.IsNotExist(err) {
					lockGone = true
					break
				}
			}
			if lockGone && !s.isToolInstalled(name, installPath) {
				os.RemoveAll(installPath)
				return s.installBuildTool(name, version, phpVersion)
			}
		}
		return fmt.Errorf("[bundler] failed to acquire lock for %s@%s: %w", name, version, err)
	}
	defer func() {
		lock.Close()
		os.Remove(lockPath)
	}()

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		s.logInfo("Downloading build tool %s@%s...", name, version)
		archive := archivePathFromURL(s.silo.Root, name, version, urls[0])
		if _, err := s.downloadSvc.DownloadWithFallbacks(urls, archive); err != nil {
			return fmt.Errorf("[download] failed to download %s@%s: %w", name, version, err)
		}

		s.logInfo("Extracting build tool %s@%s...", name, version)
		if err := os.MkdirAll(installPath, 0755); err != nil {
			return fmt.Errorf("[bundler] failed to create directory for %s@%s: %w", name, version, err)
		}

		if _, err := s.unloadSvc.Unpack(archive, installPath); err != nil {
			return fmt.Errorf("[unload] failed to extract %s@%s: %w", name, version, err)
		}

		s.logInfo("Installing build tool %s@%s", name, version)
	}

	if name == "zig" {
		zigBinary := s.findZigBinary(installPath)
		if zigBinary == "" {
			return fmt.Errorf("[bundler] zig binary not found in %s", installPath)
		}
		if err := os.Chmod(zigBinary, 0755); err != nil {
			return fmt.Errorf("[bundler] failed to chmod zig binary: %w", err)
		}
	}

	if err := s.siloRepo.IncrementBuildToolRef(name, version, phpVersion); err != nil {
		return fmt.Errorf("[bundler] failed to increment build-tool ref: %w", err)
	}

	return nil
}

func (s *bundlerRepository) findZigBinary(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subZig := filepath.Join(dir, entry.Name(), "zig")
			if _, err := os.Stat(subZig); err == nil {
				return subZig
			}
			subBin := filepath.Join(dir, entry.Name(), "bin", "zig")
			if _, err := os.Stat(subBin); err == nil {
				return subBin
			}
		}
	}
	directZig := filepath.Join(dir, "zig")
	if _, err := os.Stat(directZig); err == nil {
		return directZig
	}
	directBin := filepath.Join(dir, "bin", "zig")
	if _, err := os.Stat(directBin); err == nil {
		return directBin
	}
	return ""
}

func (s *bundlerRepository) isToolInstalled(name, installPath string) bool {
	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return false
	}

	switch name {
	case "zig":
		return s.findZigBinary(installPath) != ""
	default:
		entries, err := os.ReadDir(installPath)
		if err != nil {
			return false
		}
		return len(entries) > 0
	}
}
