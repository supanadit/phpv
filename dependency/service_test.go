package dependency

import (
	"strings"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestGetPHPConfigureFlags_PHP7(t *testing.T) {
	service := NewService("/tmp/test-phpv")
	version := domain.Version{Major: 7, Minor: 4, Patch: 33}

	flags := service.GetPHPConfigureFlags(version)

	// PHP 7.x should use -dir suffixed flags
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
			t.Errorf("expected flag %s not found in PHP 7.x configure flags", flag)
		}
	}

	// Ensure PHP 8.x flags are NOT present
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--with-libxml=") {
			t.Error("PHP 7.x should not have --with-libxml flag (should be --with-libxml-dir)")
		}
		if strings.HasPrefix(flag, "--with-openssl=") {
			t.Error("PHP 7.x should not have --with-openssl flag (should be --with-openssl-dir)")
		}
		if strings.HasPrefix(flag, "--with-zlib=") {
			t.Error("PHP 7.x should not have --with-zlib flag (should be --with-zlib-dir)")
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
