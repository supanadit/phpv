package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestDownloadRepository_NewDownloadRepository(t *testing.T) {
	repo := NewDownloadRepository()

	if repo == nil {
		t.Error("expected repository to not be nil")
	}

	if repo.client == nil {
		t.Error("expected http client to be set")
	}
}

func TestDownloadRepository_Download_Fresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/gzip")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download.Status != domain.DownloadStatusCompleted {
		t.Errorf("expected status to be %s, got %s", domain.DownloadStatusCompleted, download.Status)
	}

	if _, err := os.Stat(destination); err != nil {
		t.Error("expected file to be created")
	}
}

func TestDownloadRepository_Download_Resume_Success(t *testing.T) {
	var receivedRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRange = r.Header.Get("Range")

		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("full content"))
			return
		}

		if !strings.Contains(rangeHeader, "bytes=5-") {
			t.Errorf("expected Range header to contain bytes=5-, got %s", rangeHeader)
		}

		w.Header().Set("Content-Range", "bytes 5-13/14")
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("remaining"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	if err := os.WriteFile(destination, []byte("first"), 0o644); err != nil {
		t.Fatalf("failed to create partial file: %v", err)
	}

	repo := &DownloadRepository{
		client: server.Client(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download.Status != domain.DownloadStatusCompleted {
		t.Errorf("expected status to be %s, got %s", domain.DownloadStatusCompleted, download.Status)
	}

	if receivedRange == "" {
		t.Error("expected Range header to be sent")
	}

	if !strings.Contains(receivedRange, "bytes=5-") {
		t.Errorf("expected Range header to contain bytes=5-, got %s", receivedRange)
	}
}

func TestDownloadRepository_Download_Resume_Restart(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" && requestCount == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("full content - no range support"))
			return
		}

		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("full content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	if err := os.WriteFile(destination, []byte("partial"), 0o644); err != nil {
		t.Fatalf("failed to create partial file: %v", err)
	}

	repo := &DownloadRepository{
		client: server.Client(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download.Status != domain.DownloadStatusCompleted {
		t.Errorf("expected status to be %s, got %s", domain.DownloadStatusCompleted, download.Status)
	}

	content, _ := os.ReadFile(destination)
	if string(content) != "full content" {
		t.Errorf("expected file content to be 'full content', got %s", string(content))
	}
}

func TestDownloadRepository_Download_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
	}

	download, err := repo.Download(server.URL, destination)
	if err == nil {
		t.Error("expected error, got nil")
	}

	if download.Status != domain.DownloadStatusNotFound {
		t.Errorf("expected status to be %s, got %s", domain.DownloadStatusNotFound, download.Status)
	}
}

func TestDownloadRepository_Download_TracksProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		w.Write(make([]byte, 100))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download.Size != 100 {
		t.Errorf("expected Size to be 100, got %d", download.Size)
	}

	if download.DownloadedSize != 100 {
		t.Errorf("expected DownloadedSize to be 100, got %d", download.DownloadedSize)
	}
}

func TestDownloadRepository_Download_Resume_Progress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			w.Header().Set("Content-Range", "bytes 50-99/100")
			w.WriteHeader(http.StatusPartialContent)
			w.Write(make([]byte, 50))
			return
		}

		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		w.Write(make([]byte, 100))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	if err := os.WriteFile(destination, make([]byte, 50), 0o644); err != nil {
		t.Fatalf("failed to create partial file: %v", err)
	}

	repo := &DownloadRepository{
		client: server.Client(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download.Size != 100 {
		t.Errorf("expected Size to be 100, got %d", download.Size)
	}

	if download.DownloadedSize != 100 {
		t.Errorf("expected DownloadedSize to be 100, got %d", download.DownloadedSize)
	}
}

func TestDownloadRepository_Exists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected method HEAD, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Length", "12345")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := &DownloadRepository{
		client: server.Client(),
	}

	fileInfo, err := repo.Exists(server.URL)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !fileInfo.Exists {
		t.Error("expected Exists to be true")
	}

	if fileInfo.Size != 12345 {
		t.Errorf("expected Size to be 12345, got %d", fileInfo.Size)
	}

	if fileInfo.ContentType != "application/gzip" {
		t.Errorf("expected ContentType to be application/gzip, got %s", fileInfo.ContentType)
	}
}

func TestDownloadRepository_Exists_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	repo := &DownloadRepository{
		client: server.Client(),
	}

	fileInfo, err := repo.Exists(server.URL)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if fileInfo.Exists {
		t.Error("expected Exists to be false")
	}
}
