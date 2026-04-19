package http

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
)

type DownloadRepository struct {
	client *http.Client
	fs     afero.Fs
}

func NewDownloadRepository() *DownloadRepository {
	return &DownloadRepository{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		fs: afero.NewOsFs(),
	}
}

func NewDownloadRepositoryWithTimeout(timeout time.Duration) *DownloadRepository {
	return &DownloadRepository{
		client: &http.Client{
			Timeout: timeout,
		},
		fs: afero.NewOsFs(),
	}
}

func NewDownloadRepositoryWithFs(fs afero.Fs) *DownloadRepository {
	return &DownloadRepository{
		client: &http.Client{
			Timeout: 120 * time.Second,
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
		return fmt.Errorf("[download] failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("[download] failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[download] file not found or inaccessible: %s", url)
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
		return nil, fmt.Errorf("[download] failed to create destination directory: %w", err)
	}

	supportResume := r.checkResumeSupport(url)

	var downloadedSize int64
	var file afero.File

	stat, err := r.fs.Stat(destination)
	if err == nil && stat.Size() > 0 && supportResume {
		downloadedSize = stat.Size()
		file, err = r.fs.OpenFile(destination, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
		if err != nil {
			return nil, fmt.Errorf("[download] failed to open file: %w", err)
		}
	} else {
		if err == nil && stat.Size() > 0 {
			if err := r.fs.Remove(destination); err != nil {
				return nil, fmt.Errorf("[download] failed to remove incomplete file: %w", err)
			}
		}
		file, err = r.fs.Create(destination)
		if err != nil {
			return nil, fmt.Errorf("[download] failed to create file: %w", err)
		}
		downloadedSize = 0
	}
	defer file.Close()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[download] failed to create request: %w", err)
	}

	if downloadedSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", downloadedSize))
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[download] failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if downloadedSize > 0 {
			if err := file.Truncate(0); err != nil {
				return nil, fmt.Errorf("[download] failed to truncate file: %w", err)
			}
			if _, err := file.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("[download] failed to seek file: %w", err)
			}
			// Ensure position is at start after truncate+seek
			if _, err := file.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("[download] failed to reposition file: %w", err)
			}
		}
	case http.StatusPartialContent:
		if downloadedSize == 0 {
			return nil, fmt.Errorf("[download] server returned partial content but no existing file found")
		}
	case http.StatusRequestedRangeNotSatisfiable:
		file.Close()
		r.fs.Remove(destination)
		return r.Download(url, destination)
	default:
		return nil, fmt.Errorf("[download] unexpected status code: %d", resp.StatusCode)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return nil, fmt.Errorf("[download] failed to write file: %w", err)
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

func (r *DownloadRepository) DownloadWithFallbacks(urls []string, destination string, options ...download.DownloadOption) (*domain.Download, error) {
	opts := &download.DownloadOptions{
		MaxRetries: 3,
		RetryDelay: 1000, // 1 second base delay
	}

	for _, opt := range options {
		opt(opts)
	}

	for _, url := range urls {
		var lastErr error

		for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
			if attempt > 0 {
				delay := time.Duration(opts.RetryDelay*(1<<(attempt-1))) * time.Millisecond
				time.Sleep(delay)
			}

			err := r.exists(url)
			if err != nil {
				lastErr = fmt.Errorf("[attempt %d/%d] HEAD check failed: %w", attempt+1, opts.MaxRetries+1, err)
				continue
			}

			req, err := http.NewRequest(http.MethodHead, url, nil)
			if err != nil {
				lastErr = fmt.Errorf("[attempt %d/%d] failed to create request: %w", attempt+1, opts.MaxRetries+1, err)
				continue
			}

			resp, err := r.client.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("[attempt %d/%d] request failed: %w", attempt+1, opts.MaxRetries+1, err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("[attempt %d/%d] file not found or inaccessible: %s", attempt+1, opts.MaxRetries+1, url)
				continue
			}

			if !r.isValidContentType(resp) {
				lastErr = fmt.Errorf("[attempt %d/%d] server returned HTML instead of archive: %s", attempt+1, opts.MaxRetries+1, url)
				continue
			}

			download, err := r.DownloadWithRetry(url, destination, opts.MaxRetries, opts.RetryDelay)
			if err != nil {
				lastErr = fmt.Errorf("[attempt %d/%d] download failed: %w", attempt+1, opts.MaxRetries+1, err)
				continue
			}

			if opts.Checksum != "" {
				if err := verifyChecksum(destination, opts.Checksum); err != nil {
					r.fs.Remove(destination)
					lastErr = fmt.Errorf("[attempt %d/%d] checksum verification failed: %w", attempt+1, opts.MaxRetries+1, err)
					continue
				}
			}

			return download, nil
		}

		if lastErr != nil {
			continue
		}
	}

	return nil, fmt.Errorf("[download] all download attempts failed")
}

func (r *DownloadRepository) DownloadWithRetry(url, destination string, maxRetries, retryDelay int) (*domain.Download, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(retryDelay*(1<<(attempt-1))) * time.Millisecond
			time.Sleep(delay)
		}

		download, err := r.Download(url, destination)
		if err == nil {
			return download, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("[download] download failed after %d attempts: %w", maxRetries+1, lastErr)
}

func verifyChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("[download] failed to open file for checksum verification: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("[download] failed to compute checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("[download] checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}
