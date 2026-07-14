package disk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/supanadit/phpv/update"
)

type UpdateRepository struct {
	httpClient *http.Client
}

func NewUpdateRepository() *UpdateRepository {
	return &UpdateRepository{httpClient: http.DefaultClient}
}

func (r *UpdateRepository) FetchLatestRelease() (update.Release, error) {
	resp, err := r.httpClient.Get("https://api.github.com/repos/supanadit/phpv/releases/latest")
	if err != nil {
		return update.Release{}, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return update.Release{}, fmt.Errorf("GitHub API: unexpected status %s", resp.Status)
	}

	var ghRel struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghRel); err != nil {
		return update.Release{}, fmt.Errorf("parse release: %w", err)
	}

	rel := update.Release{TagName: ghRel.TagName}
	for _, a := range ghRel.Assets {
		rel.Assets = append(rel.Assets, update.Asset{
			Name:        a.Name,
			DownloadURL: a.BrowserDownloadURL,
			Size:        a.Size,
		})
	}
	return rel, nil
}

func (r *UpdateRepository) DownloadFile(url, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download: unexpected status %s", resp.Status)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (r *UpdateRepository) FetchChecksums(url string) (map[string]string, error) {
	resp, err := r.httpClient.Get(url)
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

	cs := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			cs[parts[1]] = parts[0]
		}
	}
	return cs, nil
}

func (r *UpdateRepository) ExecutablePath() (string, error) {
	return os.Executable()
}

func (r *UpdateRepository) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (r *UpdateRepository) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *UpdateRepository) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (r *UpdateRepository) Remove(path string) error {
	return os.Remove(path)
}

func (r *UpdateRepository) Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

func (r *UpdateRepository) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (r *UpdateRepository) Getenv(key string) string {
	return os.Getenv(key)
}

func (r *UpdateRepository) VerifyChecksum(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash file: %w", err)
	}

	gotHash := fmt.Sprintf("%x", h.Sum(nil))
	if gotHash != expectedHash {
		return fmt.Errorf("checksum mismatch: got %s, want %s", gotHash, expectedHash)
	}
	return nil
}
