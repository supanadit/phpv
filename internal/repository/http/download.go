package http

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
			Timeout: 10 * time.Minute,
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
			Timeout: 10 * time.Minute,
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

	// Check if file already exists and appears complete.
	stat, statErr := r.fs.Stat(destination)
	if statErr == nil && stat.Size() > 0 {
		// First check: Content-Length match
		contentLength := getContentLengthFromHead(url, r.client)
		if contentLength > 0 && stat.Size() == contentLength {
			return &domain.Download{
				URL:         url,
				Destination: destination,
			}, nil
		}

		// Second check: validate archive integrity (works even without Content-Length)
		if isValidArchive(destination) {
			return &domain.Download{
				URL:         url,
				Destination: destination,
			}, nil
		}
	}

	supportResume := checkResumeSupportFromHead(url, r.client)

	var downloadedSize int64
	var file afero.File

	stat, statErr = r.fs.Stat(destination)
	if statErr == nil && stat.Size() > 0 && supportResume {
		// Resume: open existing file for read+write without truncation.
		// Use O_RDWR (not O_APPEND) so we can seek to the correct position.
		downloadedSize = stat.Size()
		file, statErr = r.fs.OpenFile(destination, os.O_CREATE|os.O_RDWR, 0o644)
		if statErr != nil {
			return nil, fmt.Errorf("[download] failed to open file: %w", statErr)
		}
	} else {
		// Fresh download: remove any partial/corrupt file and create new.
		if statErr == nil && stat.Size() > 0 {
			if err := r.fs.Remove(destination); err != nil {
				return nil, fmt.Errorf("[download] failed to remove incomplete file: %w", err)
			}
		}
		file, statErr = r.fs.Create(destination)
		if statErr != nil {
			return nil, fmt.Errorf("[download] failed to create file: %w", statErr)
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
		// Server sent the full file (ignored our Range request or no partial file).
		// Truncate and write from the beginning.
		if err := file.Truncate(0); err != nil {
			return nil, fmt.Errorf("[download] failed to truncate file: %w", err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("[download] failed to seek file: %w", err)
		}
		downloadedSize = 0

	case http.StatusPartialContent:
		if downloadedSize == 0 {
			return nil, fmt.Errorf("[download] server returned partial content but no existing file found")
		}
		// Seek to end of existing data so new data appends correctly.
		if _, err := file.Seek(0, 2); err != nil {
			return nil, fmt.Errorf("[download] failed to seek file for resume: %w", err)
		}

	case http.StatusRequestedRangeNotSatisfiable:
		// The server says our range is not satisfiable. This usually means
		// the file is already complete. Try to parse total size from
		// Content-Range header (format: "bytes */TOTAL" or "bytes START-END/TOTAL").
		totalSize := parseTotalSizeFromContentRange(resp.Header.Get("Content-Range"))
		if totalSize > 0 && downloadedSize >= totalSize {
			// File is complete.
			return &domain.Download{
				URL:         url,
				Destination: destination,
			}, nil
		}
		if totalSize > 0 && downloadedSize > 0 && downloadedSize < totalSize {
			// File is genuinely partial but something went wrong with the range.
			// Fall through to restart the download from scratch.
		}
		// Either total size is unknown, or file might be complete.
		// If we have a file with data and can't confirm it's incomplete,
		// assume it's complete to avoid infinite re-download loops.
		if downloadedSize > 0 && totalSize <= 0 {
			return &domain.Download{
				URL:         url,
				Destination: destination,
			}, nil
		}
		// totalSize is known and different from downloadedSize, or totalSize is 0
		// and downloadedSize is 0. Restart from scratch.
		file.Close()
		if err := r.fs.Remove(destination); err != nil {
			return nil, fmt.Errorf("[download] failed to remove file for restart: %w", err)
		}
		return r.downloadFresh(url, destination)

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

// downloadFresh performs a fresh download without any resume logic.
func (r *DownloadRepository) downloadFresh(url, destination string) (*domain.Download, error) {
	r.ensureFs()

	dir := filepath.Dir(destination)
	if err := r.fs.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("[download] failed to create destination directory: %w", err)
	}

	file, err := r.fs.Create(destination)
	if err != nil {
		return nil, fmt.Errorf("[download] failed to create file: %w", err)
	}
	defer file.Close()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[download] failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[download] failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

func getContentLengthFromHead(url string, client *http.Client) int64 {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return 0
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	if resp.ContentLength > 0 {
		return resp.ContentLength
	}

	// Fallback: parse Content-Length header manually (some servers/Go versions
	// don't populate resp.ContentLength for HEAD responses).
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil && n > 0 {
			return n
		}
	}

	return 0
}

func checkResumeSupportFromHead(url string, client *http.Client) bool {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	return resp.Header.Get("Accept-Ranges") == "bytes"
}

// parseTotalSizeFromContentRange extracts the total size from a Content-Range header.
// Formats: "bytes */TOTAL", "bytes START-END/TOTAL", "bytes */TOTAL"
func parseTotalSizeFromContentRange(cr string) int64 {
	if cr == "" {
		return 0
	}

	// Content-Range: bytes */TOTAL  or  bytes START-END/TOTAL
	parts := strings.SplitN(cr, "/", 2)
	if len(parts) != 2 {
		return 0
	}

	total, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0
	}
	return total
}

// isValidArchive checks if a file is a valid tar archive (optionally compressed).
// This allows skipping re-download when Content-Length is unavailable.
func isValidArchive(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Check file size - archives should be at least 100KB
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	if stat.Size() < 100*1024 {
		return false
	}

	// Check magic bytes for gzip, bzip2, xz, or plain tar
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil || n < 2 {
		return false
	}

	// Gzip: 1f 8b
	if buf[0] == 0x1f && buf[1] == 0x8b {
		return true
	}

	// Bzip2: 42 5a (BZ)
	if buf[0] == 0x42 && buf[1] == 0x5a {
		return true
	}

	// XZ: fd 37 7a 58 5a 00 (xz magic)
	if n >= 6 && buf[0] == 0xfd && buf[1] == 0x37 && buf[2] == 0x7a && buf[3] == 0x58 && buf[4] == 0x5a && buf[5] == 0x00 {
		return true
	}

	// Plain tar: check if it looks like a tar header
	// Tar headers have the filename at offset 0 and ustar magic at offset 257
	// For simplicity, just check if first 100 bytes are printable ASCII (filename field)
	isPrintable := true
	for i := 0; i < 100 && i < n; i++ {
		b := buf[i]
		// Allow printable ASCII, space, tab, null
		if b != 0 && b != ' ' && b != '\t' && (b < 32 || b > 126) {
			isPrintable = false
			break
		}
	}
	if isPrintable {
		// Try to read ustar magic at offset 257
		_, err := file.Seek(257, 0)
		if err == nil {
			magic := make([]byte, 5)
			if _, err := file.Read(magic); err == nil {
				if string(magic[:5]) == "ustar" {
					return true
				}
			}
		}
	}

	return false
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