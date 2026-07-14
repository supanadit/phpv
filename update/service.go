package update

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type checksums struct {
	entries map[string]string
}

func fetchChecksums(checksumsURL string) (*checksums, error) {
	resp, err := http.Get(checksumsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("checksums: unexpected status %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}

	cs := &checksums{entries: make(map[string]string)}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			cs.entries[parts[1]] = parts[0]
		}
	}
	return cs, nil
}

func (cs *checksums) verify(filePath, assetName string) error {
	expectedHash, ok := cs.entries[assetName]
	if !ok {
		return fmt.Errorf("no checksum found for %s", assetName)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash file: %w", err)
	}

	gotHash := fmt.Sprintf("%x", h.Sum(nil))
	if gotHash != expectedHash {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", assetName, gotHash, expectedHash)
	}
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

func CheckForUpdate(currentVersion string) (latest string, hasUpdate bool, err error) {
	resp, err := http.Get("https://api.github.com/repos/supanadit/phpv/releases/latest")
	if err != nil {
		return "", false, fmt.Errorf("check for update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", false, fmt.Errorf("GitHub API: unexpected status %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", false, fmt.Errorf("parse release: %w", err)
	}

	latest = rel.TagName
	if currentVersion == "dev" || currentVersion == "" {
		return latest, true, nil
	}

	// Simple string comparison (assumes semver tags like v0.1.0, v0.2.0, etc.)
	if currentVersion == latest {
		return latest, false, nil
	}

	return latest, true, nil
}

func SelfUpdate(currentVersion string) error {
	latest, hasUpdate, err := CheckForUpdate(currentVersion)
	if err != nil {
		return err
	}
	if !hasUpdate {
		fmt.Printf("Already up to date (%s)\n", currentVersion)
		return nil
	}

	fmt.Printf("Updating from %s to %s...\n", currentVersion, latest)

	resp, err := http.Get("https://api.github.com/repos/supanadit/phpv/releases/latest")
	if err != nil {
		return fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return fmt.Errorf("parse release: %w", err)
	}

	myAsset := assetName()
	var downloadURL string
	var assetSize int64
	for _, a := range rel.Assets {
		if a.Name == myAsset {
			downloadURL = a.BrowserDownloadURL
			assetSize = a.Size
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s (available: %s)", myAsset, listAssetNames(rel.Assets))
	}

	// Find checksums file
	var checksumsURL string
	for _, a := range rel.Assets {
		if a.Name == "checksums.txt" || a.Name == "sha256sums.txt" {
			checksumsURL = a.BrowserDownloadURL
			break
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return fmt.Errorf("ensure exec dir: %w", err)
	}

	// Check writability
	testFile := filepath.Join(execDir, ".phpv_update_test")
	if err := os.WriteFile(testFile, []byte{}, 0o644); err != nil {
		return fmt.Errorf("cannot write to %s (try running with sudo): %w", execDir, err)
	}
	os.Remove(testFile)

	// Download to temp file
	tmpFile := filepath.Join(execDir, ".phpv_update_download")
	fmt.Printf("Downloading %s (%d bytes)...\n", downloadURL, assetSize)

	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	dlResp, err := http.Get(downloadURL)
	if err != nil {
		out.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("download: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		out.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("download: unexpected status %s", dlResp.Status)
	}

	if _, err := io.Copy(out, dlResp.Body); err != nil {
		out.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("download body: %w", err)
	}
	out.Close()

	// Verify checksum if available
	if checksumsURL != "" {
		fmt.Println("Verifying checksum...")
		cs, err := fetchChecksums(checksumsURL)
		if err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("fetch checksums: %w", err)
		}
		if err := cs.verify(tmpFile, myAsset); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("✓ Checksum verified")
	}

	// Make executable
	if err := os.Chmod(tmpFile, 0o755); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("chmod temp file: %w", err)
	}

	// Replace binary
	backupFile := execPath + ".bak"
	os.Remove(backupFile) // remove any stale backup

	if err := os.Rename(execPath, backupFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("backup current binary: %w", err)
	}

	if err := os.Rename(tmpFile, execPath); err != nil {
		// Try to restore backup
		os.Rename(backupFile, execPath)
		os.Remove(tmpFile)
		return fmt.Errorf("replace binary: %w", err)
	}

	os.Remove(backupFile)
	fmt.Printf("✓ Updated to %s\n", latest)
	return nil
}

func listAssetNames(assets []asset) string {
	var names []string
	for _, a := range assets {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}
