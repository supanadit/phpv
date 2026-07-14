package update

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

type Service struct {
	repo    Repository
	version string
}

func NewService(repo Repository, version string) *Service {
	return &Service{repo: repo, version: version}
}

func (s *Service) CheckForUpdate() (latest string, hasUpdate bool, err error) {
	rel, err := s.repo.FetchLatestRelease()
	if err != nil {
		return "", false, err
	}

	latest = rel.TagName
	if s.version == "dev" || s.version == "" {
		return latest, true, nil
	}
	if s.version == latest {
		return latest, false, nil
	}
	return latest, true, nil
}

func (s *Service) SelfUpdate() error {
	latest, hasUpdate, err := s.CheckForUpdate()
	if err != nil {
		return err
	}
	if !hasUpdate {
		fmt.Printf("Already up to date (%s)\n", s.version)
		return nil
	}

	fmt.Printf("Updating from %s to %s...\n", s.version, latest)

	rel, err := s.repo.FetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch release: %w", err)
	}

	myAsset := assetName()
	var downloadURL string
	var assetSize int64
	for _, a := range rel.Assets {
		if a.Name == myAsset {
			downloadURL = a.DownloadURL
			assetSize = a.Size
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s (available: %s)", myAsset, listAssetNames(rel.Assets))
	}

	var checksumsURL string
	for _, a := range rel.Assets {
		if a.Name == "checksums.txt" || a.Name == "sha256sums.txt" {
			checksumsURL = a.DownloadURL
			break
		}
	}

	execPath, err := s.repo.ExecutablePath()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)
	if err := s.repo.MkdirAll(execDir, 0o755); err != nil {
		return fmt.Errorf("ensure exec dir: %w", err)
	}

	testFile := filepath.Join(execDir, ".phpv_update_test")
	if err := s.repo.WriteFile(testFile, []byte{}, 0o644); err != nil {
		return fmt.Errorf("cannot write to %s (try running with sudo): %w", execDir, err)
	}
	s.repo.Remove(testFile)

	tmpFile := filepath.Join(execDir, ".phpv_update_download")
	fmt.Printf("Downloading %s (%d bytes)...\n", downloadURL, assetSize)

	if err := s.repo.DownloadFile(downloadURL, tmpFile); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	if checksumsURL != "" {
		fmt.Println("Verifying checksum...")
		cs, err := s.repo.FetchChecksums(checksumsURL)
		if err != nil {
			s.repo.Remove(tmpFile)
			return fmt.Errorf("fetch checksums: %w", err)
		}
		expectedHash, ok := cs[myAsset]
		if !ok {
			s.repo.Remove(tmpFile)
			return fmt.Errorf("no checksum found for %s", myAsset)
		}
		if err := s.repo.VerifyChecksum(tmpFile, expectedHash); err != nil {
			s.repo.Remove(tmpFile)
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("✓ Checksum verified")
	}

	if err := s.repo.Chmod(tmpFile, 0o755); err != nil {
		s.repo.Remove(tmpFile)
		return fmt.Errorf("chmod temp file: %w", err)
	}

	backupFile := execPath + ".bak"
	s.repo.Remove(backupFile)

	if err := s.repo.Rename(execPath, backupFile); err != nil {
		s.repo.Remove(tmpFile)
		return fmt.Errorf("backup current binary: %w", err)
	}

	if err := s.repo.Rename(tmpFile, execPath); err != nil {
		s.repo.Rename(backupFile, execPath)
		s.repo.Remove(tmpFile)
		return fmt.Errorf("replace binary: %w", err)
	}

	s.repo.Remove(backupFile)
	fmt.Printf("✓ Updated to %s\n", latest)
	return nil
}

func assetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	switch osName {
	case "darwin":
		osName = "macOS"
	case "linux":
		osName = "Linux"
	case "windows":
		osName = "Windows"
	}

	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	case "386":
		arch = "i386"
	}

	return fmt.Sprintf("phpv-%s-%s", osName, arch)
}

func listAssetNames(assets []Asset) string {
	var names []string
	for _, a := range assets {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}
