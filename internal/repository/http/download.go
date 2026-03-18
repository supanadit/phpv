package http

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/supanadit/phpv/domain"
)

func generateID() string {
	hash := md5.Sum([]byte(time.Now().String()))
	return fmt.Sprintf("%x", hash)
}

type DownloadRepository struct {
	client *http.Client
}

func NewDownloadRepository() *DownloadRepository {
	return &DownloadRepository{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *DownloadRepository) Exists(url string) (*domain.FileInfo, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return &domain.FileInfo{
			URL:    url,
			Exists: false,
		}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return &domain.FileInfo{
			URL:    url,
			Exists: false,
		}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	fileInfo := &domain.FileInfo{
		URL:         url,
		ContentType: resp.Header.Get("Content-Type"),
		Exists:      false,
	}

	if resp.ContentLength > 0 {
		fileInfo.Size = resp.ContentLength
	}

	switch resp.StatusCode {
	case http.StatusOK:
		fileInfo.Exists = true
	case http.StatusUnauthorized, http.StatusForbidden:
		fileInfo.Exists = false
	case http.StatusNotFound:
		fileInfo.Exists = false
	default:
		fileInfo.Exists = false
	}

	return fileInfo, nil
}

func (r *DownloadRepository) Download(url, destination string) (*domain.Download, error) {
	fileInfo, err := r.Exists(url)
	if err != nil {
		return nil, fmt.Errorf("failed to check file existence: %w", err)
	}

	if !fileInfo.Exists {
		return &domain.Download{
			ID:          generateID(),
			URL:         url,
			Destination: destination,
			Status:      domain.DownloadStatusNotFound,
		}, fmt.Errorf("file not found or inaccessible: %s", url)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var downloadedSize int64
	var file *os.File

	stat, err := os.Stat(destination)
	if err == nil && stat.Size() > 0 {
		downloadedSize = stat.Size()
		file, err = os.OpenFile(destination, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for append: %w", err)
		}
	} else {
		file, err = os.Create(destination)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
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
			downloadedSize = 0
		}
	case http.StatusPartialContent:
		if downloadedSize == 0 {
			return nil, fmt.Errorf("server returned partial content but no existing file found")
		}
	default:
		return &domain.Download{
			ID:          generateID(),
			URL:         url,
			Destination: destination,
			Status:      domain.DownloadStatusFailed,
		}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		now := time.Now()
		return &domain.Download{
			ID:             generateID(),
			URL:            url,
			Destination:    destination,
			FilePath:       destination,
			Status:         domain.DownloadStatusPartial,
			Size:           fileInfo.Size,
			DownloadedSize: downloadedSize + written,
			CreatedAt:      now,
			UpdatedAt:      now,
		}, fmt.Errorf("failed to write file: %w", err)
	}

	downloadedSize += written

	_, err = os.Stat(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	now := time.Now()
	return &domain.Download{
		ID:             generateID(),
		URL:            url,
		Destination:    destination,
		FilePath:       destination,
		Status:         domain.DownloadStatusCompleted,
		Size:           fileInfo.Size,
		DownloadedSize: downloadedSize,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
