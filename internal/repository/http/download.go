package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
