package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
)

type DownloadRepository struct {
	client *http.Client
	fs     afero.Fs
}

func NewDownloadRepository() *DownloadRepository {
	return &DownloadRepository{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		fs: afero.NewOsFs(),
	}
}

func NewDownloadRepositoryWithFs(fs afero.Fs) *DownloadRepository {
	return &DownloadRepository{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		fs: fs,
	}
}

func (r *DownloadRepository) ensureFs() {
	if r.fs == nil {
		r.fs = afero.NewOsFs()
	}
}

func (r *DownloadRepository) exists(url string) error {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("file not found or inaccessible: %s", url)
	}

	return nil
}

func (r *DownloadRepository) Download(url, destination string) (*domain.Download, error) {
	r.ensureFs()

	if err := r.exists(url); err != nil {
		return nil, err
	}

	dir := filepath.Dir(destination)
	if err := r.fs.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	supportResume := r.checkResumeSupport(url)

	var downloadedSize int64
	var file afero.File

	stat, err := r.fs.Stat(destination)
	if err == nil && stat.Size() > 0 && supportResume {
		downloadedSize = stat.Size()
		file, err = r.fs.OpenFile(destination, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for append: %w", err)
		}
	} else {
		if err == nil && stat.Size() > 0 {
			if err := r.fs.Remove(destination); err != nil {
				return nil, fmt.Errorf("failed to remove incomplete file: %w", err)
			}
		}
		file, err = r.fs.Create(destination)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
		downloadedSize = 0
	}
	defer file.Close()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if downloadedSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", downloadedSize))
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if downloadedSize > 0 {
			if err := file.Truncate(0); err != nil {
				return nil, fmt.Errorf("failed to truncate file: %w", err)
			}
			if _, err := file.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("failed to seek file: %w", err)
			}
		}
	case http.StatusPartialContent:
		if downloadedSize == 0 {
			return nil, fmt.Errorf("server returned partial content but no existing file found")
		}
	case http.StatusRequestedRangeNotSatisfiable:
		file.Close()
		r.fs.Remove(destination)
		return r.Download(url, destination)
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &domain.Download{
		URL:         url,
		Destination: destination,
	}, nil
}

func (r *DownloadRepository) checkResumeSupport(url string) bool {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	return resp.Header.Get("Accept-Ranges") == "bytes"
}

func (r *DownloadRepository) isValidContentType(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return true
	}
	blockedPrefixes := []string{
		"text/html",
		"application/xhtml",
	}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return false
		}
	}
	return true
}

func (r *DownloadRepository) DownloadWithFallbacks(urls []string, destination string) (*domain.Download, error) {
	var lastErr error
	for _, url := range urls {
		err := r.exists(url)
		if err != nil {
			lastErr = err
			continue
		}

		req, err := http.NewRequest(http.MethodHead, url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		resp, err := r.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("file not found or inaccessible: %s", url)
			continue
		}

		if !r.isValidContentType(resp) {
			lastErr = fmt.Errorf("server returned HTML instead of archive: %s", url)
			continue
		}

		download, err := r.Download(url, destination)
		if err != nil {
			lastErr = err
			continue
		}
		return download, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all download attempts failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no URLs available to download")
}
