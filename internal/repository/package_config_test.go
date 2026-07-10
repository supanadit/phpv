package repository

import (
	"reflect"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestBuildRegistries_FromRanges(t *testing.T) {
	cfg := PackageConfig{
		Name:        "php",
		Type:        "source_code",
		Ranges:      BuildMinorRanges(8, []MinorRange{{Minor: 0, PatchEnd: 2}}),
		URLTemplate: "https://php.net/distributions/php-{version}.tar.gz",
	}
	got := BuildRegistries(cfg)
	want := []domain.Registry{
		{Name: "php", Type: "source_code", Version: "8.0.0", URL: "https://php.net/distributions/php-8.0.0.tar.gz"},
		{Name: "php", Type: "source_code", Version: "8.0.1", URL: "https://php.net/distributions/php-8.0.1.tar.gz"},
		{Name: "php", Type: "source_code", Version: "8.0.2", URL: "https://php.net/distributions/php-8.0.2.tar.gz"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildRegistries(ranges) = %v, want %v", got, want)
	}
}

func TestBuildRegistries_FromVersions(t *testing.T) {
	cfg := PackageConfig{
		Name:        "perl",
		Type:        "source_code",
		Versions:    []string{"5.22.3", "5.20.0"},
		URLTemplate: "https://cpan.org/src/5.0/perl-{version}.tar.gz",
	}
	got := BuildRegistries(cfg)
	want := []domain.Registry{
		{Name: "perl", Type: "source_code", Version: "5.22.3", URL: "https://cpan.org/src/5.0/perl-5.22.3.tar.gz"},
		{Name: "perl", Type: "source_code", Version: "5.20.0", URL: "https://cpan.org/src/5.0/perl-5.20.0.tar.gz"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildRegistries(versions) = %v, want %v", got, want)
	}
}

func TestBuildRegistries_ExtensionOverride(t *testing.T) {
	cfg := PackageConfig{
		Name:        "perl",
		Type:        "source_code",
		Versions:    []string{"5.22.3", "5.18.4"},
		URLTemplate: "https://cpan.org/src/5.0/perl-{version}.{ext}",
		Extension: ExtensionConfig{
			Default: "tar.gz",
			Override: []ExtOverride{
				{Before: "5.20.0", Ext: "tar.bz2"},
			},
		},
	}
	got := BuildRegistries(cfg)
	want := []domain.Registry{
		{Name: "perl", Type: "source_code", Version: "5.22.3", URL: "https://cpan.org/src/5.0/perl-5.22.3.tar.gz"},
		{Name: "perl", Type: "source_code", Version: "5.18.4", URL: "https://cpan.org/src/5.0/perl-5.18.4.tar.bz2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildRegistries(extension) = %v, want %v", got, want)
	}
}

func TestBuildRegistries_Checksums(t *testing.T) {
	cfg := PackageConfig{
		Name:        "php",
		Type:        "source_code",
		Versions:    []string{"8.5.8", "8.5.7"},
		URLTemplate: "https://php.net/distributions/php-{version}.tar.gz",
		Checksums: []Checksum{
			{Version: "8.5.8", Type: "sha256", Value: "abc123"},
		},
	}
	got := BuildRegistries(cfg)
	want := []domain.Registry{
		{
			Name:          "php",
			Type:          "source_code",
			Version:       "8.5.8",
			URL:           "https://php.net/distributions/php-8.5.8.tar.gz",
			ChecksumType:  "sha256",
			ChecksumValue: "abc123",
		},
		{
			Name:    "php",
			Type:    "source_code",
			Version: "8.5.7",
			URL:     "https://php.net/distributions/php-8.5.7.tar.gz",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildRegistries(checksums) = %v, want %v", got, want)
	}
}

func TestBuildRegistries_Skip(t *testing.T) {
	cfg := PackageConfig{
		Name:        "php",
		Type:        "source_code",
		Ranges:      BuildMinorRanges(8, []MinorRange{{Minor: 0, PatchEnd: 2}}),
		Skip:        []string{"8.0.1"},
		URLTemplate: "https://php.net/distributions/php-{version}.tar.gz",
	}
	got := BuildRegistries(cfg)
	want := []string{"8.0.0", "8.0.2"}
	if len(got) != len(want) {
		t.Fatalf("BuildRegistries(skip) count = %d, want %d", len(got), len(want))
	}
	for i, entry := range got {
		if entry.Version != want[i] {
			t.Fatalf("BuildRegistries(skip)[%d].Version = %q, want %q", i, entry.Version, want[i])
		}
	}
}

func TestResolveExtension_NoOverride(t *testing.T) {
	cfg := ExtensionConfig{Default: "tar.gz"}
	if got := resolveExtension(cfg, "5.18.4"); got != "tar.gz" {
		t.Fatalf("resolveExtension(no override) = %q, want %q", got, "tar.gz")
	}
}

func TestResolveExtension_Override(t *testing.T) {
	cfg := ExtensionConfig{
		Default: "tar.gz",
		Override: []ExtOverride{
			{Before: "5.20.0", Ext: "tar.bz2"},
		},
	}
	if got := resolveExtension(cfg, "5.18.4"); got != "tar.bz2" {
		t.Fatalf("resolveExtension(override) = %q, want %q", got, "tar.bz2")
	}
	if got := resolveExtension(cfg, "5.20.0"); got != "tar.gz" {
		t.Fatalf("resolveExtension(boundary) = %q, want %q", got, "tar.gz")
	}
}
