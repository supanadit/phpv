package disk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/config"
	"github.com/supanadit/phpv/internal/utils"
)

func TestSiloRepository_GetSilo(t *testing.T) {
	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	silo, err := repo.GetSilo()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if silo == nil {
		t.Fatal("expected silo to not be nil")
	}

	if silo.Root == "" {
		t.Error("expected silo root to not be empty")
	}
}

func TestSiloRepository_EnsurePaths(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	err = repo.EnsurePaths()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPaths := []string{
		filepath.Join(tmpDir, "cache"),
		filepath.Join(tmpDir, "sources"),
		filepath.Join(tmpDir, "versions"),
		filepath.Join(tmpDir, "bin"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected path %s to exist", path)
		}
	}
}

func TestSiloRepository_ArchiveOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	pkg := "php"
	ver := "8.3.0"

	if repo.ArchiveExists(pkg, ver) {
		t.Error("expected archive to not exist initially")
	}

	archivePath := repo.GetArchivePath(pkg, ver)
	expectedPath := filepath.Join(tmpDir, "cache", pkg, ver, "archive")
	if archivePath != expectedPath {
		t.Errorf("expected archive path %s, got %s", expectedPath, archivePath)
	}

	content := "test archive content"
	err = repo.StoreArchive(pkg, ver, strings.NewReader(content))
	if err != nil {
		t.Fatalf("failed to store archive: %v", err)
	}

	if !repo.ArchiveExists(pkg, ver) {
		t.Error("expected archive to exist after storing")
	}

	rc, err := repo.RetrieveArchive(pkg, ver)
	if err != nil {
		t.Fatalf("failed to retrieve archive: %v", err)
	}
	defer rc.Close()

	data := make([]byte, len(content))
	n, err := rc.Read(data)
	if err != nil {
		t.Fatalf("failed to read archive: %v", err)
	}
	if string(data[:n]) != content {
		t.Errorf("expected content %q, got %q", content, string(data[:n]))
	}

	err = repo.RemoveArchive(pkg, ver)
	if err != nil {
		t.Fatalf("failed to remove archive: %v", err)
	}

	if repo.ArchiveExists(pkg, ver) {
		t.Error("expected archive to not exist after removing")
	}
}

func TestSiloRepository_SourceOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	pkg := "php"
	ver := "8.3.0"

	if repo.SourceExists(pkg, ver) {
		t.Error("expected source to not exist initially")
	}

	sourcePath := repo.GetSourcePath(pkg, ver)
	expectedPath := filepath.Join(tmpDir, "sources", pkg, ver)
	if sourcePath != expectedPath {
		t.Errorf("expected source path %s, got %s", expectedPath, sourcePath)
	}

	content := "test source content"
	err = repo.StoreSource(pkg, ver, strings.NewReader(content))
	if err != nil {
		t.Fatalf("failed to store source: %v", err)
	}

	if !repo.SourceExists(pkg, ver) {
		t.Error("expected source to exist after storing")
	}

	rc, err := repo.RetrieveSource(pkg, ver)
	if err != nil {
		t.Fatalf("failed to retrieve source: %v", err)
	}
	defer rc.Close()

	data := make([]byte, len(content))
	n, err := rc.Read(data)
	if err != nil {
		t.Fatalf("failed to read source: %v", err)
	}
	if string(data[:n]) != content {
		t.Errorf("expected content %q, got %q", content, string(data[:n]))
	}

	err = repo.RemoveSource(pkg, ver)
	if err != nil {
		t.Fatalf("failed to remove source: %v", err)
	}

	if repo.SourceExists(pkg, ver) {
		t.Error("expected source to not exist after removing")
	}
}

func TestSiloRepository_VersionOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	pkg := "php"
	ver := "8.3.0"

	if repo.VersionExists(pkg, ver) {
		t.Error("expected version to not exist initially")
	}

	versionPath := repo.GetVersionPath(pkg, ver)
	expectedPath := filepath.Join(tmpDir, "versions", pkg, ver)
	if versionPath != expectedPath {
		t.Errorf("expected version path %s, got %s", expectedPath, versionPath)
	}

	content := "test version content"
	err = repo.StoreVersion(pkg, ver, strings.NewReader(content))
	if err != nil {
		t.Fatalf("failed to store version: %v", err)
	}

	if !repo.VersionExists(pkg, ver) {
		t.Error("expected version to exist after storing")
	}

	rc, err := repo.RetrieveVersion(pkg, ver)
	if err != nil {
		t.Fatalf("failed to retrieve version: %v", err)
	}
	defer rc.Close()

	data := make([]byte, len(content))
	n, err := rc.Read(data)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}
	if string(data[:n]) != content {
		t.Errorf("expected content %q, got %q", content, string(data[:n]))
	}

	err = repo.RemoveVersion(pkg, ver)
	if err != nil {
		t.Fatalf("failed to remove version: %v", err)
	}

	if repo.VersionExists(pkg, ver) {
		t.Error("expected version to not exist after removing")
	}
}

func TestSiloRepository_FullClean(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	pkg := "php"
	ver := "8.3.0"

	err = repo.StoreArchive(pkg, ver, strings.NewReader("archive"))
	if err != nil {
		t.Fatalf("failed to store archive: %v", err)
	}
	err = repo.StoreSource(pkg, ver, strings.NewReader("source"))
	if err != nil {
		t.Fatalf("failed to store source: %v", err)
	}
	err = repo.StoreVersion(pkg, ver, strings.NewReader("version"))
	if err != nil {
		t.Fatalf("failed to store version: %v", err)
	}

	err = repo.FullClean(pkg, ver)
	if err != nil {
		t.Fatalf("failed to full clean: %v", err)
	}

	if repo.ArchiveExists(pkg, ver) {
		t.Error("expected archive to not exist after full clean")
	}
	if repo.SourceExists(pkg, ver) {
		t.Error("expected source to not exist after full clean")
	}
	if repo.VersionExists(pkg, ver) {
		t.Error("expected version to not exist after full clean")
	}
}

func TestSiloRepository_CleanAll(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	err = repo.EnsurePaths()
	if err != nil {
		t.Fatalf("failed to ensure paths: %v", err)
	}

	cachePath := filepath.Join(tmpDir, "cache", "php", "8.3.0")
	sourcePath := filepath.Join(tmpDir, "sources", "php", "8.3.0")
	versionPath := filepath.Join(tmpDir, "versions", "php", "8.3.0")

	os.MkdirAll(cachePath, 0o755)
	os.MkdirAll(sourcePath, 0o755)
	os.MkdirAll(versionPath, 0o755)

	err = repo.CleanAll()
	if err != nil {
		t.Fatalf("failed to clean all: %v", err)
	}

	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("expected cache path to not exist after clean all")
	}
	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Error("expected source path to not exist after clean all")
	}
	if _, err := os.Stat(versionPath); !os.IsNotExist(err) {
		t.Error("expected version path to not exist after clean all")
	}
}

func TestSiloRepository_ValidateInput(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	err = repo.StoreArchive("php", "8.3.0", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to store archive: %v", err)
	}

	tests := []struct {
		pkg  string
		ver  string
		want bool
	}{
		{"php", "8.3.0", true},
		{"", "8.3.0", false},
		{"php", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		if repo.ArchiveExists(tt.pkg, tt.ver) != tt.want {
			t.Errorf("ArchiveExists(%q, %q) = %v, want %v", tt.pkg, tt.ver, !tt.want, tt.want)
		}
	}
}

func TestSiloRepository_ListArchives(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	archives := repo.ListArchives()
	if len(archives) != 0 {
		t.Errorf("expected 0 archives initially, got %d", len(archives))
	}

	err = repo.StoreArchive("php", "8.3.0", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to store archive: %v", err)
	}
	err = repo.StoreArchive("openssl", "1.1.1", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to store archive: %v", err)
	}

	archives = repo.ListArchives()
	if len(archives) != 2 {
		t.Errorf("expected 2 archives, got %d: %v", len(archives), archives)
	}
}

func TestSiloRepository_ListSources(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	sources := repo.ListSources()
	if len(sources) != 0 {
		t.Errorf("expected 0 sources initially, got %d", len(sources))
	}

	err = repo.StoreSource("php", "8.3.0", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to store source: %v", err)
	}

	sources = repo.ListSources()
	if len(sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(sources))
	}
}

func TestSiloRepository_ListVersions(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetForTesting(tmpDir)
	defer config.ResetForTesting()

	repo, err := NewSiloRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	versions := repo.ListVersions()
	if len(versions) != 0 {
		t.Errorf("expected 0 versions initially, got %d", len(versions))
	}

	err = repo.StoreVersion("php", "8.3.0", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to store version: %v", err)
	}

	versions = repo.ListVersions()
	if len(versions) != 0 {
		t.Errorf("expected 0 versions (no binary), got %d", len(versions))
	}

	outputBinDir := filepath.Join(tmpDir, "versions", "8.3.0", "output", "bin")
	if err := os.MkdirAll(outputBinDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	phpBinary := filepath.Join(outputBinDir, "php")
	if err := os.WriteFile(phpBinary, []byte("fake"), 0755); err != nil {
		t.Fatalf("failed to create fake php binary: %v", err)
	}

	versions = repo.ListVersions()
	if len(versions) != 1 {
		t.Errorf("expected 1 version (with binary), got %d", len(versions))
	}
}

func TestDomain_SiloPathMethods(t *testing.T) {
	silo := domain.Silo{Root: "/home/user/.phpv"}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"RootPath", utils.RootPath(&silo), "/home/user/.phpv"},
		{"CachePath", utils.CachePath(&silo), "/home/user/.phpv/cache"},
		{"SourcePath", utils.SourcePath(&silo), "/home/user/.phpv/sources"},
		{"VersionPath", utils.VersionPath(&silo), "/home/user/.phpv/versions"},
		{"BinPath", utils.BinPath(&silo), "/home/user/.phpv/bin"},
		{"ArchiveKey", utils.ArchiveKey("php", "8.3.0"), "cache/php/8.3.0/archive"},
		{"SourceKey", utils.SourceKey("php", "8.3.0"), "sources/php/8.3.0"},
		{"VersionKey", utils.VersionKey("php", "8.3.0"), "versions/php/8.3.0"},
		{"SourceDirKey", utils.SourceDirKey("php", "8.3.0"), "sources/php/8.3.0/src"},
		{"GetArchivePath", utils.GetArchivePath(&silo, "php", "8.3.0"), "/home/user/.phpv/cache/php/8.3.0/archive"},
		{"GetSourcePath", utils.GetSourcePath(&silo, "php", "8.3.0"), "/home/user/.phpv/sources/php/8.3.0"},
		{"GetVersionPath", utils.GetVersionPath(&silo, "php", "8.3.0"), "/home/user/.phpv/versions/php/8.3.0"},
		{"GetSourceDirPath", utils.GetSourceDirPath(&silo, "php", "8.3.0"), "/home/user/.phpv/sources/php/8.3.0/src"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}
