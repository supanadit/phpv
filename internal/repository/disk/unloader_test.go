package disk

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestUnloadRepository_NewUnloadRepository(t *testing.T) {
	repo := NewUnloadRepository()

	if repo == nil {
		t.Error("expected repository to not be nil")
	}
}

func TestUnloadRepository_Unpack_Zip(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")
	extractDir := filepath.Join(tmpDir, "output")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, _ := zw.Create("subdir/")
	w.Write([]byte(""))
	w, _ = zw.Create("subdir/file1.txt")
	w.Write([]byte("test content for file1"))
	w, _ = zw.Create("subdir/file2.txt")
	w.Write([]byte("test content for file2"))
	zw.Close()

	repo := NewUnloadRepository()
	unload, err := repo.Unpack(archivePath, extractDir)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if unload.Source != archivePath {
		t.Errorf("expected Source to be %s, got %s", archivePath, unload.Source)
	}

	if unload.Destination != extractDir {
		t.Errorf("expected Destination to be %s, got %s", extractDir, unload.Destination)
	}

	if unload.Extracted != 2 {
		t.Errorf("expected Extracted to be 2, got %d", unload.Extracted)
	}

	path := filepath.Join(extractDir, "file1.txt")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file file1.txt to exist at %s", path)
	}
}

func TestUnloadRepository_Unpack_UnknownFormat(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(archivePath, []byte("not an archive"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	repo := NewUnloadRepository()
	_, err := repo.Unpack(archivePath, tmpDir)
	if err != ErrUnknownFormat {
		t.Errorf("expected ErrUnknownFormat, got %v", err)
	}
}

func TestUnloadRepository_Unpack_CreatesDestination(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")
	extractDir := filepath.Join(tmpDir, "newdir", "nested")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, _ := zw.Create("subdir/")
	w.Write([]byte(""))
	w, _ = zw.Create("subdir/test.txt")
	w.Write([]byte("content"))
	zw.Close()

	repo := NewUnloadRepository()
	unload, err := repo.Unpack(archivePath, extractDir)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if unload.Extracted != 1 {
		t.Errorf("expected Extracted to be 1, got %d", unload.Extracted)
	}

	path := filepath.Join(extractDir, "test.txt")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist at %s", path)
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		source string
		format string
	}{
		{"file.tar.gz", domain.UnloadFormatTarGz},
		{"file.tgz", domain.UnloadFormatTarGz},
		{"file.tar.xz", domain.UnloadFormatTarXz},
		{"file.zip", domain.UnloadFormatZip},
		{"file.TAR.GZ", domain.UnloadFormatTarGz},
		{"file.txt", ""},
		{"file.tar", ""},
	}

	for _, tt := range tests {
		format := detectFormat(tt.source)
		if format != tt.format {
			t.Errorf("detectFormat(%s) = %s, want %s", tt.source, format, tt.format)
		}
	}
}

func createTarGz(t *testing.T, path string, files map[string]string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if strings.HasSuffix(name, "/") {
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0o755
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if !strings.HasSuffix(name, "/") {
			tw.Write([]byte(content))
		}
	}
}

func TestUnloadRepository_Unpack_TarGz_NoTrailingSlash_StripsPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "mydir.tar.gz")
	extractDir := filepath.Join(tmpDir, "output")

	createTarGz(t, archivePath, map[string]string{
		"mydir/":             "",
		"mydir/file1.txt":    "content1",
		"mydir/file2.txt":    "content2",
		"mydir/subdir/":      "",
		"mydir/subdir/a.txt": "content3",
	})

	repo := NewUnloadRepository()
	unload, err := repo.Unpack(archivePath, extractDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if unload.Extracted != 3 {
		t.Errorf("expected 3 files extracted, got %d", unload.Extracted)
	}

	if _, err := os.Stat(filepath.Join(extractDir, "mydir")); !os.IsNotExist(err) {
		t.Error("expected mydir to NOT exist (should be stripped)")
	}
	if _, err := os.Stat(filepath.Join(extractDir, "file1.txt")); err != nil {
		t.Error("expected file1.txt to exist at root of extractDir")
	}
}

func TestUnloadRepository_Unpack_TarGz_TrailingSlash_NoStrip(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "mydir.tar.gz")
	extractDir := filepath.Join(tmpDir, "output")

	createTarGz(t, archivePath, map[string]string{
		"mydir/":          "",
		"mydir/file1.txt": "content1",
		"mydir/file2.txt": "content2",
	})

	repo := NewUnloadRepository()
	unload, err := repo.Unpack(archivePath+"/", extractDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if unload.Extracted != 2 {
		t.Errorf("expected 2 files extracted, got %d", unload.Extracted)
	}

	if _, err := os.Stat(filepath.Join(extractDir, "mydir")); err != nil {
		t.Error("expected mydir to exist (no strip)")
	}
	if _, err := os.Stat(filepath.Join(extractDir, "mydir", "file1.txt")); err != nil {
		t.Error("expected file1.txt inside mydir")
	}
}

func TestUnloadRepository_Unpack_Zip_TrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "mydir.zip")
	extractDir := filepath.Join(tmpDir, "output")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	zw.Create("mydir/")
	w, _ := zw.Create("mydir/file1.txt")
	w.Write([]byte("content"))
	zw.Close()

	repo := NewUnloadRepository()
	unload, err := repo.Unpack(archivePath+"/", extractDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if unload.Extracted != 1 {
		t.Errorf("expected 1 file extracted, got %d", unload.Extracted)
	}

	if _, err := os.Stat(filepath.Join(extractDir, "mydir", "file1.txt")); err != nil {
		t.Error("expected file1.txt inside mydir (no strip)")
	}
}
