package memory

import (
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestRegistryRepository_List_PHP(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.List("php", false, "")
	if err != nil {
		t.Fatalf("List(php, false, \"\") returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("List(php, false, \"\") returned empty result")
	}

	first := got[0]
	if first.Name != "php" {
		t.Fatalf("first entry.Name = %q, want php", first.Name)
	}
	if first.Type != "source_code" {
		t.Fatalf("first entry.Type = %q, want source_code", first.Type)
	}
	if !strings.Contains(first.URL, "https://www.php.net/distributions/") {
		t.Fatalf("first entry.URL = %q, missing php.net domain", first.URL)
	}
	if first.OS != "all" {
		t.Fatalf("first entry.OS = %q, want %q", first.OS, "all")
	}
}

func TestRegistryRepository_List_PHP_ChecksumFilter(t *testing.T) {
	repo := NewRegistryRepository()

	// checksum=false returns all entries (including those without checksums)
	all, err := repo.List("php", false, "")
	if err != nil {
		t.Fatalf("List(php, false, \"\") returned error: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("List(php, false, \"\") returned empty result")
	}

	// checksum=true returns only entries that have checksums
	onlyWithChecksum, err := repo.List("php", true, "")
	if err != nil {
		t.Fatalf("List(php, true, \"\") returned error: %v", err)
	}
	if len(onlyWithChecksum) != 1 {
		t.Fatalf("List(php, true, \"\") returned %d entries, want 1", len(onlyWithChecksum))
	}
	if onlyWithChecksum[0].Version != "8.5.8" {
		t.Fatalf("List(php, true, \"\") version = %q, want 8.5.8", onlyWithChecksum[0].Version)
	}
	if onlyWithChecksum[0].ChecksumType != "sha256" {
		t.Fatalf("8.5.8 ChecksumType = %q, want sha256", onlyWithChecksum[0].ChecksumType)
	}
	want := "6ebc55e52af4396385e689f7af0f28944fbbf966843433b573e9dc1dc03df539"
	if onlyWithChecksum[0].ChecksumValue != want {
		t.Fatalf("8.5.8 ChecksumValue = %q, want %q", onlyWithChecksum[0].ChecksumValue, want)
	}

	// checksum=true for a package with no checksums returns empty
	none, err := repo.List("cmake", true, "")
	if err != nil {
		t.Fatalf("List(cmake, true, \"\") returned error: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("List(cmake, true, \"\") returned %d entries, want 0", len(none))
	}
}

func TestRegistryRepository_List_CMake(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.List("cmake", false, "")
	if err != nil {
		t.Fatalf("List(cmake) returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("List(cmake) returned empty result")
	}

	first := got[0]
	if first.Name != "cmake" {
		t.Fatalf("first entry.Name = %q, want cmake", first.Name)
	}
	if first.Type != "binary" {
		t.Fatalf("first entry.Type = %q, want binary", first.Type)
	}
	if !strings.Contains(first.URL, "github.com/Kitware/CMake") {
		t.Fatalf("first entry.URL = %q, missing github.com/Kitware/CMake", first.URL)
	}
	if first.OS != "linux" {
		t.Fatalf("first entry.OS = %q, want linux", first.OS)
	}
}

func TestRegistryRepository_List_CMake_OSSilter(t *testing.T) {
	repo := NewRegistryRepository()

	// Requesting linux returns cmake entries
	linux, err := repo.List("cmake", false, "linux")
	if err != nil {
		t.Fatalf("List(cmake, false, linux) returned error: %v", err)
	}
	if len(linux) == 0 {
		t.Fatal("List(cmake, false, linux) returned empty result")
	}
	for _, r := range linux {
		if r.OS != "linux" {
			t.Fatalf("entry.OS = %q, want linux", r.OS)
		}
	}

	// Requesting darwin returns no cmake entries (cmake is linux-only)
	darwin, err := repo.List("cmake", false, "darwin")
	if err != nil {
		t.Fatalf("List(cmake, false, darwin) returned error: %v", err)
	}
	if len(darwin) != 0 {
		t.Fatalf("List(cmake, false, darwin) returned %d entries, want 0", len(darwin))
	}
}

func TestRegistryRepository_List_PHP_OSFilter_AllOSIncluded(t *testing.T) {
	repo := NewRegistryRepository()

	// PHP is OS-agnostic, so requesting linux should still return all entries
	linux, err := repo.List("php", false, "linux")
	if err != nil {
		t.Fatalf("List(php, false, linux) returned error: %v", err)
	}
	all, err := repo.List("php", false, "")
	if err != nil {
		t.Fatalf("List(php, false, \"\") returned error: %v", err)
	}
	if len(linux) != len(all) {
		t.Fatalf("List(php, false, linux) returned %d entries, want %d (same as no OS filter)", len(linux), len(all))
	}
}

func TestRegistryRepository_List_Perl(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.List("perl", false, "")
	if err != nil {
		t.Fatalf("List(perl) returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("List(perl) returned empty result")
	}

	for _, r := range got {
		if r.Name != "perl" {
			t.Fatalf("entry.Name = %q, want perl", r.Name)
		}
		if r.Type != "source_code" {
			t.Fatalf("entry.Type = %q, want source_code", r.Type)
		}
		if r.OS != "all" {
			t.Fatalf("entry.OS = %q, want %q", r.OS, "all")
		}
		if !strings.Contains(r.URL, "https://www.cpan.org/src/5.0/") {
			t.Fatalf("entry.URL = %q, missing cpan.org domain", r.URL)
		}
	}

	wantExt := map[string]string{
		"5.18.4": "tar.bz2",
		"5.20.0": "tar.gz",
		"5.42.1": "tar.gz",
	}
	for _, r := range got {
		if expected, ok := wantExt[r.Version]; ok {
			if !strings.HasSuffix(r.URL, "."+expected) {
				t.Fatalf("perl %s URL = %q, want extension .%s", r.Version, r.URL, expected)
			}
		}
	}
}

func TestRegistryRepository_List_Unknown(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.List("unknown", false, "")
	if err == nil {
		t.Fatal("List(unknown) expected error, got nil")
	}
	if got != nil {
		t.Fatalf("List(unknown) result = %v, want nil", got)
	}
}

func TestRegistryRepository_List_ConsistentShape(t *testing.T) {
	repo := NewRegistryRepository()
	packages := []string{"php", "cmake", "perl"}
	for _, name := range packages {
		entries, err := repo.List(name, false, "")
		if err != nil {
			t.Fatalf("List(%q) returned error: %v", name, err)
		}
		for i, e := range entries {
			if e.Name == "" || e.Type == "" || e.URL == "" || e.Version == "" || e.OS == "" {
				t.Fatalf("List(%q)[%d] has empty field: %+v", name, i, e)
			}
		}
	}
}

func TestRegistryRepository_List_PHP_SortedByVersion(t *testing.T) {
	repo := NewRegistryRepository()
	entries, err := repo.List("php", false, "")
	if err != nil {
		t.Fatalf("List(php) returned error: %v", err)
	}
	// BuildRanges lists major versions in the order they are provided (8, 7, 5, 4).
	if entries[0].Version != "8.0.0" {
		t.Fatalf("first php version = %q, want 8.0.0", entries[0].Version)
	}

	found400 := false
	for _, e := range entries {
		if e.Version == "4.0.0" {
			found400 = true
			break
		}
	}
	if !found400 {
		t.Fatal("expected version 4.0.0 somewhere in php list")
	}
}

func TestRegistryRepository_Get_PHP(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.Get("php", "8.0.0", false, "")
	if err != nil {
		t.Fatalf("Get(php, 8.0.0) returned error: %v", err)
	}
	if got.Name != "php" {
		t.Fatalf("Get(php, 8.0.0).Name = %q, want php", got.Name)
	}
	if got.Version != "8.0.0" {
		t.Fatalf("Get(php, 8.0.0).Version = %q, want 8.0.0", got.Version)
	}
	if !strings.Contains(got.URL, "https://www.php.net/distributions/php-8.0.0.tar.gz") {
		t.Fatalf("Get(php, 8.0.0).URL = %q, want php.net URL", got.URL)
	}
}

func TestRegistryRepository_Get_PHP_WithChecksum(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.Get("php", "8.5.8", true, "")
	if err != nil {
		t.Fatalf("Get(php, 8.5.8, true) returned error: %v", err)
	}
	if got.Version != "8.5.8" {
		t.Fatalf("Get(php, 8.5.8, true).Version = %q, want 8.5.8", got.Version)
	}
	if got.ChecksumType != "sha256" {
		t.Fatalf("Get(php, 8.5.8, true).ChecksumType = %q, want sha256", got.ChecksumType)
	}
	want := "6ebc55e52af4396385e689f7af0f28944fbbf966843433b573e9dc1dc03df539"
	if got.ChecksumValue != want {
		t.Fatalf("Get(php, 8.5.8, true).ChecksumValue = %q, want %q", got.ChecksumValue, want)
	}
}

func TestRegistryRepository_Get_PHP_VersionNotFound(t *testing.T) {
	repo := NewRegistryRepository()
	_, err := repo.Get("php", "99.99.99", false, "")
	if err == nil {
		t.Fatal("Get(php, 99.99.99) expected error, got nil")
	}
}

func TestRegistryRepository_Get_UnknownPackage(t *testing.T) {
	repo := NewRegistryRepository()
	_, err := repo.Get("unknown", "1.0.0", false, "")
	if err == nil {
		t.Fatal("Get(unknown, 1.0.0) expected error, got nil")
	}
}

func TestRegistryRepository_Get_CMake_OSSFilter(t *testing.T) {
	repo := NewRegistryRepository()

	// cmake is linux-only; requesting linux returns the entry
	got, err := repo.Get("cmake", "3.27.6", false, "linux")
	if err != nil {
		t.Fatalf("Get(cmake, 3.27.6, linux) returned error: %v", err)
	}
	if got.Version != "3.27.6" {
		t.Fatalf("Get(cmake, 3.27.6, linux).Version = %q, want 3.27.6", got.Version)
	}

	// Requesting darwin returns error (cmake is linux-only)
	_, err = repo.Get("cmake", "3.27.6", false, "darwin")
	if err == nil {
		t.Fatal("Get(cmake, 3.27.6, darwin) expected error, got nil")
	}
}

func TestRegistryRepository_Get_PHP_ChecksumFilter(t *testing.T) {
	repo := NewRegistryRepository()

	// PHP 8.5.8 has a checksum, so it is returned when checksum=true
	got, err := repo.Get("php", "8.5.8", true, "")
	if err != nil {
		t.Fatalf("Get(php, 8.5.8, true) returned error: %v", err)
	}
	if got.ChecksumType != "sha256" {
		t.Fatalf("Get(php, 8.5.8, true).ChecksumType = %q, want sha256", got.ChecksumType)
	}

	// PHP 8.0.0 has no checksum, so it is not found when checksum=true
	_, err = repo.Get("php", "8.0.0", true, "")
	if err == nil {
		t.Fatal("Get(php, 8.0.0, true) expected error, got nil")
	}
}

func TestRegistryRepository_Get_Perl(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.Get("perl", "5.42.1", false, "")
	if err != nil {
		t.Fatalf("Get(perl, 5.42.1) returned error: %v", err)
	}
	if got.Version != "5.42.1" {
		t.Fatalf("Get(perl, 5.42.1).Version = %q, want 5.42.1", got.Version)
	}
	if !strings.HasSuffix(got.URL, ".tar.gz") {
		t.Fatalf("Get(perl, 5.42.1).URL = %q, want .tar.gz suffix", got.URL)
	}
}

var _ domain.Registry = domain.Registry{}