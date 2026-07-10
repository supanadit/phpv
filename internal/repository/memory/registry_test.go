package memory

import (
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestRegistryRepository_List_PHP(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.List("php", false)
	if err != nil {
		t.Fatalf("List(php, false) returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("List(php, false) returned empty result")
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
}

func TestRegistryRepository_List_PHP_ChecksumFilter(t *testing.T) {
	repo := NewRegistryRepository()

	// checksum=false returns all entries (including those without checksums)
	all, err := repo.List("php", false)
	if err != nil {
		t.Fatalf("List(php, false) returned error: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("List(php, false) returned empty result")
	}

	// checksum=true returns only entries that have checksums
	onlyWithChecksum, err := repo.List("php", true)
	if err != nil {
		t.Fatalf("List(php, true) returned error: %v", err)
	}
	if len(onlyWithChecksum) != 1 {
		t.Fatalf("List(php, true) returned %d entries, want 1", len(onlyWithChecksum))
	}
	if onlyWithChecksum[0].Version != "8.5.8" {
		t.Fatalf("List(php, true) version = %q, want 8.5.8", onlyWithChecksum[0].Version)
	}
	if onlyWithChecksum[0].ChecksumType != "sha256" {
		t.Fatalf("8.5.8 ChecksumType = %q, want sha256", onlyWithChecksum[0].ChecksumType)
	}
	want := "6ebc55e52af4396385e689f7af0f28944fbbf966843433b573e9dc1dc03df539"
	if onlyWithChecksum[0].ChecksumValue != want {
		t.Fatalf("8.5.8 ChecksumValue = %q, want %q", onlyWithChecksum[0].ChecksumValue, want)
	}

	// checksum=true for a package with no checksums returns empty
	none, err := repo.List("cmake", true)
	if err != nil {
		t.Fatalf("List(cmake, true) returned error: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("List(cmake, true) returned %d entries, want 0", len(none))
	}
}

func TestRegistryRepository_List_CMake(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.List("cmake", false)
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
}

func TestRegistryRepository_List_Perl(t *testing.T) {
	repo := NewRegistryRepository()
	got, err := repo.List("perl", false)
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
	got, err := repo.List("unknown", false)
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
		entries, err := repo.List(name, false)
		if err != nil {
			t.Fatalf("List(%q) returned error: %v", name, err)
		}
		for i, e := range entries {
			if e.Name == "" || e.Type == "" || e.URL == "" || e.Version == "" {
				t.Fatalf("List(%q)[%d] has empty field: %+v", name, i, e)
			}
		}
	}
}

func TestRegistryRepository_List_PHP_SortedByVersion(t *testing.T) {
	repo := NewRegistryRepository()
	entries, err := repo.List("php", false)
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

var _ domain.Registry = domain.Registry{}
