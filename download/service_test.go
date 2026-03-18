package download

import (
	"errors"
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockDownloadRepository struct {
	fileInfo *domain.FileInfo
	download *domain.Download
	err      error
}

func (m *mockDownloadRepository) Exists(url string) (*domain.FileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.fileInfo, nil
}

func (m *mockDownloadRepository) Download(url, destination string) (*domain.Download, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.download, nil
}

func TestNewDownloadService(t *testing.T) {
	repo := &mockDownloadRepository{}
	svc := NewService(repo)

	if svc == nil {
		t.Error("expected service to not be nil")
	}

	if svc.downloadRepository != repo {
		t.Error("expected downloadRepository to be set")
	}
}

func TestService_Exists_Success(t *testing.T) {
	expectedFileInfo := &domain.FileInfo{
		URL:         "https://www.php.net/distributions/php-8.2.0.tar.gz",
		Size:        12345678,
		ContentType: "application/gzip",
		Exists:      true,
	}

	repo := &mockDownloadRepository{
		fileInfo: expectedFileInfo,
	}

	svc := NewService(repo)
	fileInfo, err := svc.Exists("https://www.php.net/distributions/php-8.2.0.tar.gz")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !fileInfo.Exists {
		t.Error("expected Exists to be true")
	}

	if fileInfo.Size != 12345678 {
		t.Errorf("expected Size to be 12345678, got %d", fileInfo.Size)
	}
}

func TestService_Exists_Error(t *testing.T) {
	expectedErr := errors.New("failed to check existence")

	repo := &mockDownloadRepository{
		err: expectedErr,
	}

	svc := NewService(repo)
	fileInfo, err := svc.Exists("https://example.com/file.tar.gz")

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}

	if fileInfo != nil {
		t.Error("expected fileInfo to be nil on error")
	}
}

func TestService_Download_Success(t *testing.T) {
	expectedDownload := &domain.Download{
		ID:          "abc123",
		URL:         "https://www.php.net/distributions/php-8.2.0.tar.gz",
		Destination: "/tmp/php-8.2.0.tar.gz",
		FilePath:    "/tmp/php-8.2.0.tar.gz",
		Status:      domain.DownloadStatusCompleted,
		Size:        12345678,
	}

	repo := &mockDownloadRepository{
		download: expectedDownload,
	}

	svc := NewService(repo)
	download, err := svc.Download("https://www.php.net/distributions/php-8.2.0.tar.gz", "/tmp/php-8.2.0.tar.gz")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download.Status != domain.DownloadStatusCompleted {
		t.Errorf("expected status to be '%s', got '%s'", domain.DownloadStatusCompleted, download.Status)
	}

	if download.ID != "abc123" {
		t.Errorf("expected ID to be 'abc123', got '%s'", download.ID)
	}
}

func TestService_Download_Error(t *testing.T) {
	expectedErr := errors.New("failed to download")

	repo := &mockDownloadRepository{
		err: expectedErr,
	}

	svc := NewService(repo)
	download, err := svc.Download("https://example.com/file.tar.gz", "/tmp/file.tar.gz")

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}

	if download != nil {
		t.Error("expected download to be nil on error")
	}
}
