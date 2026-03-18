package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadRepository_NewDownloadRepository(t *testing.T) {
	repo := NewDownloadRepository()

	if repo == nil {
		t.Error("expected repository to not be nil")
	}

	if repo.client == nil {
		t.Error("expected http client to be set")
	}
}

func TestDownloadRepository_Download_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
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

	if download.URL != server.URL {
		t.Errorf("expected URL to be %s, got %s", server.URL, download.URL)
	}

	if download.Destination != destination {
		t.Errorf("expected destination to be %s, got %s", destination, download.Destination)
	}

	if _, err := os.Stat(destination); err != nil {
		t.Error("expected file to be created")
	}

	content, _ := os.ReadFile(destination)
	if string(content) != "test content" {
		t.Errorf("expected file content to be 'test content', got %s", string(content))
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

	if download == nil {
		t.Error("expected download to not be nil")
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

	if download == nil {
		t.Error("expected download to not be nil")
	}

	content, _ := os.ReadFile(destination)
	if string(content) != "full content" {
		t.Errorf("expected file content to be 'full content', got %s", string(content))
	}
}

func TestDownloadRepository_Download_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
	}

	_, err := repo.Download(server.URL, destination)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDownloadRepository_Download_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
	}

	_, err := repo.Download(server.URL, destination)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDownloadRepository_Download_PartialContentNoExistingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Range", "bytes 50-99/100")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(make([]byte, 50))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
	}

	_, err := repo.Download(server.URL, destination)
	if err == nil {
		t.Error("expected error for partial content without existing file")
	}
}
