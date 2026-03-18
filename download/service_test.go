package download

import (
	"errors"
	"testing"

	"github.com/supanadit/phpv/domain"
)

type mockDownloadRepository struct {
	download *domain.Download
	err      error
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

func TestService_Download_Success(t *testing.T) {
	expectedDownload := &domain.Download{
		URL:         "https://www.php.net/distributions/php-8.2.0.tar.gz",
		Destination: "/tmp/php-8.2.0.tar.gz",
	}

	repo := &mockDownloadRepository{
		download: expectedDownload,
	}

	svc := NewService(repo)
	download, err := svc.Download("https://www.php.net/distributions/php-8.2.0.tar.gz", "/tmp/php-8.2.0.tar.gz")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download.URL != expectedDownload.URL {
		t.Errorf("expected URL to be '%s', got '%s'", expectedDownload.URL, download.URL)
	}

	if download.Destination != expectedDownload.Destination {
		t.Errorf("expected Destination to be '%s', got '%s'", expectedDownload.Destination, download.Destination)
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
