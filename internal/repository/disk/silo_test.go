package disk

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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

	if _, err := repo.Download(server.URL+"/package.tar.gz", "", ""); err != nil {
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
	if _, err := repo.Download(server.URL+"/file.tar.gz", "sha256", want); err != nil {
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

	_, err := repo.Download(server.URL+"/bad.tar.gz", "sha256", "deadbeef")
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

	if _, err := repo.Download(server.URL+"/missing.tar.gz", "", ""); err == nil {
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

	if _, err := repo.Download(server.URL+"/nested.tar.gz", "", ""); err != nil {
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

	if _, err := repo.Download(server.URL+"/file.tar.gz", "md5", "abc"); err == nil {
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

func TestSiloRepository_Download_SkipExisting(t *testing.T) {
	content := "already downloaded"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	// Pre-create the file with content.
	target := filepath.Join(dir, "package.tar.gz")
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		t.Fatalf("pre-create file: %v", err)
	}

	// Download should skip — server should not be hit.
	// Use a server that returns a different body to prove skip.
	server2 := newTestServer("different content")
	defer server2.Close()

	if _, err := repo.Download(server2.URL+"/package.tar.gz", "", ""); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	// File should still have original content.
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Fatalf("file overwritten, content = %q, want %q", got, content)
	}
}

func TestSiloRepository_Download_SkipEmptyFileRedownloads(t *testing.T) {
	content := "fresh download"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	// Pre-create a zero-byte file — should NOT be skipped.
	target := filepath.Join(dir, "package.tar.gz")
	if err := os.WriteFile(target, []byte{}, 0644); err != nil {
		t.Fatalf("pre-create empty file: %v", err)
	}

	if _, err := repo.Download(server.URL+"/package.tar.gz", "", ""); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Fatalf("empty file not re-downloaded, content = %q, want %q", got, content)
	}
}

func TestSiloRepository_Download_StalePartFileRemoved(t *testing.T) {
	content := "clean download"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	// Simulate a stale .part file from a previous interrupted run.
	partPath := filepath.Join(dir, "package.tar.gz.part")
	if err := os.WriteFile(partPath, []byte("incomplete"), 0644); err != nil {
		t.Fatalf("create stale .part: %v", err)
	}

	if _, err := repo.Download(server.URL+"/package.tar.gz", "", ""); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	// Stale .part should be gone.
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Fatalf("stale .part file should be removed, statErr = %v", err)
	}

	// Final file should have correct content.
	got, err := os.ReadFile(filepath.Join(dir, "package.tar.gz"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Fatalf("content = %q, want %q", got, content)
	}
}

func TestSiloRepository_Download_NoPartFileOnSuccess(t *testing.T) {
	content := "success"
	server := newTestServer(content)
	defer server.Close()

	dir := t.TempDir()
	repo := NewSiloRepository()
	repo.baseDir = dir

	if _, err := repo.Download(server.URL+"/package.tar.gz", "", ""); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	partPath := filepath.Join(dir, "package.tar.gz.part")
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Fatalf(".part file should not exist after successful download")
	}
}

func TestSiloRepository_Extract_PreservesMtime(t *testing.T) {
	// Author a .tar.gz that mirrors a PHP release tarball: a shipped generated
	// source (.c) listed before its generator (.re), both with an identical
	// mtime. If extraction preserved mtimes, the files stay equal so make does
	// not regenerate the .c from the .re.
	archivePath := filepath.Join(t.TempDir(), "pkg.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	mtime := time.Unix(1700000000, 0)
	for _, name := range []string{"src/something.c", "src/something.re"} {
		hdr := &tar.Header{
			Name:    name,
			Mode:    0o644,
			Size:    5,
			ModTime: mtime,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(t.TempDir(), "pkg")
	repo := NewSiloRepository()
	if _, err := repo.Extract(archivePath, destDir); err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, name := range []string{"src/something.c", "src/something.re"} {
		fi, err := os.Stat(filepath.Join(destDir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if !fi.ModTime().Equal(mtime) {
			t.Fatalf("%s mtime = %v, want %v (mtime not preserved from archive)", name, fi.ModTime(), mtime)
		}
	}
}
