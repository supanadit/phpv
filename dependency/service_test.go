package dependency

import (
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestGetPHPConfigureFlags_PHP7(t *testing.T) {
	service := NewService("/tmp/test-phpv")
	version := domain.Version{Major: 7, Minor: 3, Patch: 33}

	flags := service.GetPHPConfigureFlags(version)

	// PHP 7.0-7.3 should use -dir suffixed flags
	expectedFlags := map[string]bool{
		"--with-libxml-dir":  false,
		"--with-openssl-dir": false,
		"--with-zlib-dir":    false,
		"--with-curl":        false,
		"--with-onig":        false,
	}

	for _, flag := range flags {
		for expectedPrefix := range expectedFlags {
			if strings.HasPrefix(flag, expectedPrefix) {
				expectedFlags[expectedPrefix] = true
			}
		}
	}

	for flag, found := range expectedFlags {
		if !found {
			t.Errorf("expected flag %s not found in PHP 7.0-7.3 configure flags", flag)
		}
	}

	// Ensure PHP 8.x flags are NOT present
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--with-libxml=") {
			t.Error("PHP 7.0-7.3 should not have --with-libxml flag (should be --with-libxml-dir)")
		}
		if strings.HasPrefix(flag, "--with-openssl=") {
			t.Error("PHP 7.0-7.3 should not have --with-openssl flag (should be --with-openssl-dir)")
		}
		if strings.HasPrefix(flag, "--with-zlib=") {
			t.Error("PHP 7.0-7.3 should not have --with-zlib flag (should be --with-zlib-dir)")
		}
	}
}

func TestGetPHPConfigureFlags_PHP8(t *testing.T) {
	service := NewService("/tmp/test-phpv")
	version := domain.Version{Major: 8, Minor: 3, Patch: 27}

	flags := service.GetPHPConfigureFlags(version)

	// PHP 8.x should NOT use -dir suffixed flags (except for some)
	expectedFlags := map[string]bool{
		"--with-libxml":  false,
		"--with-openssl": false,
		"--with-zlib":    false,
		"--with-curl":    false,
		"--with-onig":    false,
	}

	for _, flag := range flags {
		for expectedPrefix := range expectedFlags {
			if strings.HasPrefix(flag, expectedPrefix) {
				expectedFlags[expectedPrefix] = true
			}
		}
	}

	for flag, found := range expectedFlags {
		if !found {
			t.Errorf("expected flag %s not found in PHP 8.x configure flags", flag)
		}
	}

	// Ensure PHP 7.x -dir flags are NOT present (except curl)
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--with-libxml-dir=") {
			t.Error("PHP 8.x should not have --with-libxml-dir flag (should be --with-libxml)")
		}
		if strings.HasPrefix(flag, "--with-openssl-dir=") {
			t.Error("PHP 8.x should not have --with-openssl-dir flag (should be --with-openssl)")
		}
		if strings.HasPrefix(flag, "--with-zlib-dir=") {
			t.Error("PHP 8.x should not have --with-zlib-dir flag (should be --with-zlib)")
		}
	}
}

func TestGetPHPConfigureFlags_PHP74(t *testing.T) {
	service := NewService("/tmp/test-phpv")
	version := domain.Version{Major: 7, Minor: 4, Patch: 33}

	flags := service.GetPHPConfigureFlags(version)

	// PHP 7.4 uses the same flags as PHP 8.x (not -dir suffixed)
	expectedFlags := map[string]bool{
		"--with-libxml":  false,
		"--with-openssl": false,
		"--with-zlib":    false,
		"--with-curl":    false,
		"--with-onig":    false,
	}

	for _, flag := range flags {
		for expectedPrefix := range expectedFlags {
			if strings.HasPrefix(flag, expectedPrefix) {
				expectedFlags[expectedPrefix] = true
			}
		}
	}

	for flag, found := range expectedFlags {
		if !found {
			t.Errorf("expected flag %s not found in PHP 7.4 configure flags", flag)
		}
	}

	// Ensure old PHP 7.0-7.3 -dir flags are NOT present
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--with-libxml-dir=") {
			t.Error("PHP 7.4 should not have --with-libxml-dir flag (should be --with-libxml)")
		}
		if strings.HasPrefix(flag, "--with-openssl-dir=") {
			t.Error("PHP 7.4 should not have --with-openssl-dir flag (should be --with-openssl)")
		}
		if strings.HasPrefix(flag, "--with-zlib-dir=") {
			t.Error("PHP 7.4 should not have --with-zlib-dir flag (should be --with-zlib)")
		}
	}
}

func TestGetPHPEnvironment(t *testing.T) {
	service := NewService("/tmp/test-phpv")
	version := domain.Version{Major: 8, Minor: 3, Patch: 27}

	env := service.GetPHPEnvironment(version)

	// Check that CC and CXX are set to clang
	foundCC := false
	foundCXX := false

	for _, e := range env {
		if strings.HasPrefix(e, "CC=") {
			foundCC = true
			if !strings.Contains(e, "clang") {
				t.Error("CC should be set to clang")
			}
		}
		if strings.HasPrefix(e, "CXX=") {
			foundCXX = true
			if !strings.Contains(e, "clang") {
				t.Error("CXX should be set to clang++")
			}
		}
	}

	if !foundCC {
		t.Error("CC environment variable not set")
	}
	if !foundCXX {
		t.Error("CXX environment variable not set")
	}
}

func TestGetPHPEnvironmentWithVersionSpecificCFLAGS(t *testing.T) {
	service := NewService("/tmp/test-phpv")

	// Test PHP 7.2 which should have CFLAGS set
	version72 := domain.Version{Major: 7, Minor: 2, Patch: 0}
	env72 := service.GetPHPEnvironment(version72)

	foundCFLAGS := false
	for _, e := range env72 {
		if strings.HasPrefix(e, "CFLAGS=") {
			foundCFLAGS = true
			if !strings.Contains(e, "-D_GNU_SOURCE") {
				t.Error("CFLAGS should contain -D_GNU_SOURCE for PHP 7.2")
			}
			if !strings.Contains(e, "-D_DEFAULT_SOURCE") {
				t.Error("CFLAGS should contain -D_DEFAULT_SOURCE for PHP 7.2")
			}
			if !strings.Contains(e, "-Wno-deprecated-declarations") {
				t.Error("CFLAGS should contain -Wno-deprecated-declarations for PHP 7.2")
			}
			if !strings.Contains(e, "-D_LARGEFILE_SOURCE") {
				t.Error("CFLAGS should contain -D_LARGEFILE_SOURCE for PHP 7.2")
			}
			if !strings.Contains(e, "-D_FILE_OFFSET_BITS=64") {
				t.Error("CFLAGS should contain -D_FILE_OFFSET_BITS=64 for PHP 7.2")
			}
			if !strings.Contains(e, "-D_POSIX_C_SOURCE=200809L") {
				t.Error("CFLAGS should contain -D_POSIX_C_SOURCE=200809L for PHP 7.2")
			}
			break
		}
	}
	if !foundCFLAGS {
		t.Error("CFLAGS environment variable should be set for PHP 7.2")
	}

	// Test PHP 8.3 which should not have CFLAGS set
	version83 := domain.Version{Major: 8, Minor: 3, Patch: 27}
	env83 := service.GetPHPEnvironment(version83)

	foundCFLAGS = false
	for _, e := range env83 {
		if strings.HasPrefix(e, "CFLAGS=") {
			foundCFLAGS = true
			break
		}
	}
	if foundCFLAGS {
		t.Error("CFLAGS environment variable should not be set for PHP 8.3")
	}
}

func TestGetCacheDir(t *testing.T) {
	service := NewService("/tmp/test-phpv")
	cacheDir := service.GetCacheDir()

	expected := "/tmp/test-phpv/cache/sources"
	if cacheDir != expected {
		t.Errorf("expected cache dir %s, got %s", expected, cacheDir)
	}
}

func TestGetCachedArchivePath(t *testing.T) {
	service := NewService("/tmp/test-phpv")

	tests := []struct {
		url      string
		expected string
	}{
		{
			url:      "https://github.com/madler/zlib/releases/download/v1.3.1/zlib-1.3.1.tar.gz",
			expected: "/tmp/test-phpv/cache/sources/zlib-1.3.1.tar.gz",
		},
		{
			url:      "https://download.gnome.org/sources/libxml2/2.12/libxml2-2.12.7.tar.xz",
			expected: "/tmp/test-phpv/cache/sources/libxml2-2.12.7.tar.xz",
		},
		{
			url:      "https://www.openssl.org/source/openssl-3.3.2.tar.gz",
			expected: "/tmp/test-phpv/cache/sources/openssl-3.3.2.tar.gz",
		},
	}

	for _, tc := range tests {
		result := service.getCachedArchivePath(tc.url)
		if result != tc.expected {
			t.Errorf("for URL %s, expected %s, got %s", tc.url, tc.expected, result)
		}
	}
}
