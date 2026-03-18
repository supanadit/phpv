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

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &domain.Download{
			ID:          generateID(),
			URL:         url,
			Destination: destination,
			Status:      domain.DownloadStatusFailed,
		}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	file, err := os.Create(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	stat, err := os.Stat(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	now := time.Now()
	return &domain.Download{
		ID:          generateID(),
		URL:         url,
		Destination: destination,
		FilePath:    destination,
		Status:      domain.DownloadStatusCompleted,
		Size:        stat.Size(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}
