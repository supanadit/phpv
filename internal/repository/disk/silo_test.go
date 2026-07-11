package disk

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, content)
	}))
}

func sha256Hex(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func TestSiloRepository_Download_NoChecksum(t *testing.T) {
	server := newTestServer("hello world")
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	if err := repo.Download(server.URL+"/package.tar.gz", "", ""); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "package.tar.gz"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("downloaded content = %q, want %q", got, "hello world")
	}
}

func TestSiloRepository_Download_WithValidChecksum(t *testing.T) {
	content := "checksum-verified-content"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	want := sha256Hex(content)
	if err := repo.Download(server.URL+"/file.tar.gz", "sha256", want); err != nil {
		t.Fatalf("Download with valid checksum returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "file.tar.gz"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != content {
		t.Fatalf("downloaded content = %q, want %q", got, content)
	}
}

func TestSiloRepository_Download_WithInvalidChecksum(t *testing.T) {
	content := "some content"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	err := repo.Download(server.URL+"/bad.tar.gz", "sha256", "deadbeef")
	if err == nil {
		t.Fatal("Download with invalid checksum expected error, got nil")
	}

	// The temp .part file should have been cleaned up and the final file
	// should not exist.
	if _, statErr := os.Stat(filepath.Join(dir, "bad.tar.gz")); !os.IsNotExist(statErr) {
		t.Fatalf("final file should not exist after checksum mismatch, statErr = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "bad.tar.gz.part")); !os.IsNotExist(statErr) {
		t.Fatalf("temp .part file should not exist after checksum mismatch, statErr = %v", statErr)
	}
}

func TestSiloRepository_Download_HTTLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	if err := repo.Download(server.URL+"/missing.tar.gz", "", ""); err == nil {
		t.Fatal("Download with 404 status expected error, got nil")
	}
}

func TestSiloRepository_Download_CreatesBaseDir(t *testing.T) {
	content := "nested dir test"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	repo := NewSiloRepository()
	repo.baseDir = nested

	if err := repo.Download(server.URL+"/nested.tar.gz", "", ""); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("base directory was not created: %v", err)
	}
}

func TestSiloRepository_Download_UnsupportedChecksum(t *testing.T) {
	content := "content"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	if err := repo.Download(server.URL+"/file.tar.gz", "md5", "abc"); err == nil {
		t.Fatal("Download with unsupported checksum type expected error, got nil")
	}
}

func TestNewSiloRepository_UsesPHPVRoot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PHPV_ROOT", dir)

	repo := NewSiloRepository()
	want := filepath.Join(dir, "caches")
	if repo.baseDir != want {
		t.Fatalf("repo.baseDir = %q, want %q", repo.baseDir, want)
	}
}
