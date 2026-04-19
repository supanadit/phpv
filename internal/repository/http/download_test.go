package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
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
		fs:     afero.NewOsFs(),
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

func TestDownloadRepository_Download_AlreadyComplete(t *testing.T) {
	var getContentCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "12")
			w.WriteHeader(http.StatusOK)
			return
		}
		getContentCount++
		w.Header().Set("Content-Length", "12")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	// Pre-create a complete file (12 bytes, matching Content-Length)
	if err := os.WriteFile(destination, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create complete file: %v", err)
	}

	repo := &DownloadRepository{
		client: server.Client(),
		fs:     afero.NewOsFs(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download == nil {
		t.Error("expected download to not be nil")
	}

	if getContentCount > 0 {
		t.Errorf("expected no GET request (file already complete), but got %d GET requests", getContentCount)
	}

	content, _ := os.ReadFile(destination)
	if string(content) != "test content" {
		t.Errorf("expected file content to remain 'test content', got %s", string(content))
	}
}

func TestDownloadRepository_Download_Resume_Success(t *testing.T) {
	var receivedRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
			return
		}

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
		fs:     afero.NewOsFs(),
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
	// When the server doesn't support resume (no Accept-Ranges),
	// a partial file should be discarded and the download restarted.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			// No Accept-Ranges header, no Content-Length
			w.WriteHeader(http.StatusOK)
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
		fs:     afero.NewOsFs(),
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

func TestDownloadRepository_Download_416_CompleteFile_UnknownLength(t *testing.T) {
	// Simulates a complete file where the server returns 416 Range Not Satisfiable
	// and doesn't provide Content-Length (common with GitHub codeload / php.net).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
			return
		}

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			// File is already complete, server rejects the range
			w.Header().Set("Content-Range", "bytes */14")
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("full content!!!"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	// Pre-create a "complete" file matching the server content
	if err := os.WriteFile(destination, []byte("full content!!!"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	repo := &DownloadRepository{
		client: server.Client(),
		fs:     afero.NewOsFs(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download == nil {
		t.Error("expected download to not be nil")
	}

	// File should remain unchanged
	content, _ := os.ReadFile(destination)
	if string(content) != "full content!!!" {
		t.Errorf("expected file content to remain 'full content!!!', got %s", string(content))
	}
}

func TestDownloadRepository_Download_416_CompleteFile_KnownLength(t *testing.T) {
	// When total size is known from Content-Range and file size matches.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", "14")
			w.WriteHeader(http.StatusOK)
			return
		}

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			w.Header().Set("Content-Range", "bytes */14")
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		w.Header().Set("Content-Length", "14")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("full content!!!"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	if err := os.WriteFile(destination, []byte("full content!!!"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	repo := &DownloadRepository{
		client: server.Client(),
		fs:     afero.NewOsFs(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download == nil {
		t.Error("expected download to not be nil")
	}

	content, _ := os.ReadFile(destination)
	if string(content) != "full content!!!" {
		t.Errorf("expected file content to remain unchanged, got %s", string(content))
	}
}

func TestDownloadRepository_Download_NoResumeSupport_FreshDownload(t *testing.T) {
	// Server with no Content-Length and no Accept-Ranges (like php.net / GitHub codeload).
	// Should always download fresh when no resume support.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			// No Content-Length, no Accept-Ranges - like php.net
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fresh download content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
		fs:     afero.NewOsFs(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download == nil {
		t.Error("expected download to not be nil")
	}

	content, _ := os.ReadFile(destination)
	if string(content) != "fresh download content" {
		t.Errorf("expected file content to be 'fresh download content', got %s", string(content))
	}
}

func TestDownloadRepository_Download_ValidArchive_SkipReDownload(t *testing.T) {
	// Simulates php.net/GitHub scenario: no Content-Length, no Accept-Ranges.
	// But if a valid tar.gz already exists, it should skip re-download.
	headRequestCount := 0
	getRequestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			headRequestCount++
			// No Content-Length, no Accept-Ranges
			w.WriteHeader(http.StatusOK)
			return
		}
		getRequestCount++
		// Create a minimal valid gzip file (gzip header + minimal data)
		// Gzip magic: 1f 8b 08 00 00 00 00 00 00 ff
		gzipHeader := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}
		w.WriteHeader(http.StatusOK)
		w.Write(gzipHeader)
		w.Write([]byte("dummy compressed data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	// Create a valid gzip file (starts with 1f 8b)
	gzipHeader := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}
	validArchive := append(gzipHeader, []byte("dummy compressed data")...)
	// Make it > 100KB to pass size check
	for len(validArchive) < 100*1024 {
		validArchive = append(validArchive, 0x00)
	}
	if err := os.WriteFile(destination, validArchive, 0o644); err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}

	repo := &DownloadRepository{
		client: server.Client(),
		fs:     afero.NewOsFs(),
	}

	download, err := repo.Download(server.URL, destination)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if download == nil {
		t.Error("expected download to not be nil")
	}

	if headRequestCount == 0 {
		t.Error("expected at least 1 HEAD request")
	}
	if getRequestCount > 0 {
		t.Errorf("expected no GET request (archive already valid), but got %d GET requests", getRequestCount)
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
		fs:     afero.NewOsFs(),
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
		fs:     afero.NewOsFs(),
	}

	_, err := repo.Download(server.URL, destination)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDownloadRepository_Download_PartialContentNoExistingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Range", "bytes 50-99/100")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(make([]byte, 50))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destination := filepath.Join(tmpDir, "test.tar.gz")

	repo := &DownloadRepository{
		client: server.Client(),
		fs:     afero.NewOsFs(),
	}

	_, err := repo.Download(server.URL, destination)
	if err == nil {
		t.Error("expected error for partial content without existing file")
	}
}

func TestParseTotalSizeFromContentRange(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"bytes */14", 14},
		{"bytes 0-13/14", 14},
		{"bytes 5-13/14", 14},
		{"bytes */0", 0},
		{"", 0},
		{"invalid", 0},
		{"bytes */abc", 0},
	}

	for _, tt := range tests {
		result := parseTotalSizeFromContentRange(tt.input)
		if result != tt.expected {
			t.Errorf("parseTotalSizeFromContentRange(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
