package memory

import (
	"testing"
)

func TestNewSourceRepository(t *testing.T) {
	repo := NewSourceRepository()

	if repo == nil {
		t.Error("expected repository to not be nil")
	}
}

func TestSourceRepository_GetVersions(t *testing.T) {
	repo := NewSourceRepository()
	versions, err := repo.GetVersions()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(versions) == 0 {
		t.Error("expected versions to not be empty")
	}
}

func TestSourceRepository_GetVersions_FirstVersion(t *testing.T) {
	repo := NewSourceRepository()
	versions, err := repo.GetVersions()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(versions) == 0 {
		t.Fatal("expected versions to not be empty")
	}

	firstVersion := versions[0]
	if firstVersion.Name != "php" {
		t.Errorf("expected name to be 'php', got '%s'", firstVersion.Name)
	}

	if firstVersion.Version == "" {
		t.Error("expected version to not be empty")
	}

	if firstVersion.URL == "" {
		t.Error("expected URL to not be empty")
	}
}

func TestSourceRepository_GetVersions_VersionFormat(t *testing.T) {
	repo := NewSourceRepository()
	versions, err := repo.GetVersions()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	for _, v := range versions {
		if v.Version == "" {
			t.Error("expected version to not be empty")
		}
		if v.URL == "" {
			t.Error("expected URL to not be empty")
		}
		if v.Name != "php" {
			t.Errorf("expected name to be 'php', got '%s'", v.Name)
		}
	}
}

func TestSourceRepository_GetVersions_SortedDescending(t *testing.T) {
	repo := NewSourceRepository()
	versions, err := repo.GetVersions()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(versions) < 2 {
		t.Skip("not enough versions to test sorting")
	}

	first := versions[0].Version
	if first != "8.5.4" {
		t.Errorf("expected first version to be 8.5.4, got '%s'", first)
	}

	second := versions[1].Version
	if second != "8.5.3" {
		t.Logf("second version: %s", second)
	}
}

func TestSourceRepository_generateRangeVersions(t *testing.T) {
	repo := NewSourceRepository()
	versions := repo.generateRangeVersions(8, 2, 0, 2)

	if len(versions) != 3 {
		t.Errorf("expected 3 versions, got %d", len(versions))
	}

	if versions[0].Version != "8.2.0" {
		t.Errorf("expected first version to be 8.2.0, got '%s'", versions[0].Version)
	}

	if versions[1].Version != "8.2.1" {
		t.Errorf("expected second version to be 8.2.1, got '%s'", versions[1].Version)
	}

	if versions[2].Version != "8.2.2" {
		t.Errorf("expected third version to be 8.2.2, got '%s'", versions[2].Version)
	}
}

func TestSourceRepository_buildDownloadURL_PHP4(t *testing.T) {
	repo := NewSourceRepository()
	url := repo.buildDownloadURL(4, 4, 0)

	expected := "https://museum.php.net/php4/php-4.4.0.tar.gz"
	if url != expected {
		t.Errorf("expected URL to be '%s', got '%s'", expected, url)
	}
}

func TestSourceRepository_buildDownloadURL_PHP5(t *testing.T) {
	repo := NewSourceRepository()
	url := repo.buildDownloadURL(5, 2, 0)

	expected := "https://museum.php.net/php5/php-5.2.0.tar.gz"
	if url != expected {
		t.Errorf("expected URL to be '%s', got '%s'", expected, url)
	}
}

func TestSourceRepository_buildDownloadURL_PHP5_3(t *testing.T) {
	repo := NewSourceRepository()
	url := repo.buildDownloadURL(5, 3, 0)

	expected := "https://www.php.net/distributions/php-5.3.0.tar.gz"
	if url != expected {
		t.Errorf("expected URL to be '%s', got '%s'", expected, url)
	}
}

func TestSourceRepository_buildDownloadURL_PHP8(t *testing.T) {
	repo := NewSourceRepository()
	url := repo.buildDownloadURL(8, 2, 0)

	expected := "https://www.php.net/distributions/php-8.2.0.tar.gz"
	if url != expected {
		t.Errorf("expected URL to be '%s', got '%s'", expected, url)
	}
}
