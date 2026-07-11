package memory

import (
	"fmt"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/repository"
)

type RegistryRepository struct{}

func NewRegistryRepository() *RegistryRepository {
	return &RegistryRepository{}
}

func (reg *RegistryRepository) List(name string, checksum bool, os string) (result []domain.Registry, err error) {
	switch name {
	case "php":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "php",
			Type: "source_code",
			OS:   "all",
			Ranges: repository.BuildRanges(
				repository.BuildMinorRanges(8, []repository.MinorRange{
					{Minor: 0, PatchEnd: 30},
					{Minor: 1, PatchEnd: 33},
					{Minor: 2, PatchEnd: 29},
					{Minor: 3, PatchEnd: 27},
					{Minor: 4, PatchEnd: 19},
					{Minor: 5, PatchEnd: 8},
				}),
				repository.BuildMinorRanges(7, []repository.MinorRange{
					{Minor: 0, PatchEnd: 33},
					{Minor: 1, PatchEnd: 33},
					{Minor: 2, PatchEnd: 34},
					{Minor: 3, PatchEnd: 33},
					{Minor: 4, PatchEnd: 33},
				}),
				repository.BuildMinorRanges(5, []repository.MinorRange{
					{Minor: 0, PatchEnd: 5},
					{Minor: 1, PatchEnd: 6},
					{Minor: 2, PatchEnd: 17},
					{Minor: 3, PatchEnd: 29},
					{Minor: 4, PatchEnd: 45},
					{Minor: 5, PatchEnd: 38},
					{Minor: 6, PatchEnd: 40},
				}),
				repository.BuildMinorRanges(4, []repository.MinorRange{
					{Minor: 0, PatchEnd: 6},
					{Minor: 1, PatchEnd: 2},
					{Minor: 2, PatchEnd: 3},
					{Minor: 3, PatchEnd: 11},
					{Minor: 4, PatchEnd: 9},
				}),
			),
			URLTemplate: "https://www.php.net/distributions/php-{version}.tar.gz",
			Checksums: []repository.Checksum{
				{
					Version: "8.5.8",
					Type:    "sha256",
					Value:   "6ebc55e52af4396385e689f7af0f28944fbbf966843433b573e9dc1dc03df539",
				},
			},
		})

	case "cmake":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "cmake",
			Type: "binary",
			OS:   "linux",
			Ranges: repository.BuildRanges(
				repository.BuildMinorRanges(3, []repository.MinorRange{
					{Minor: 21, PatchEnd: 4},
					{Minor: 22, PatchEnd: 1},
					{Minor: 23, PatchEnd: 3},
					{Minor: 24, PatchEnd: 2},
					{Minor: 25, PatchEnd: 3},
					{Minor: 26, PatchEnd: 5},
					{Minor: 27, PatchEnd: 6},
				}),
			),
			URLTemplate: "https://github.com/Kitware/CMake/releases/download/v{version}/cmake-{version}-linux-x86_64.tar.gz",
		})

	case "perl":
		result = repository.BuildRegistries(repository.PackageConfig{
			Name: "perl",
			Type: "source_code",
			OS:   "all",
			Versions: []string{
				"5.42.1", "5.40.3", "5.38.5", "5.36.3", "5.34.3",
				"5.32.1", "5.30.3", "5.28.3", "5.26.3", "5.24.4",
				"5.22.3", "5.20.0", "5.18.4", "5.16.3", "5.14.4",
				"5.12.5", "5.10.1", "5.8.9", "5.6.2", "5.5.30",
				"5.4.50",
			},
			URLTemplate: "https://www.cpan.org/src/5.0/perl-{version}.{ext}",
			Extension: repository.ExtensionConfig{
				Default: "tar.gz",
				Override: []repository.ExtOverride{
					{Before: "5.20.0", Ext: "tar.bz2"},
				},
			},
		})

	default:
		return nil, fmt.Errorf("unknown package: %s", name)
	}

	// Filter by OS when a specific OS is requested.
	// Entries with OS="all" are always included.
	if os != "" && os != "all" {
		filtered := make([]domain.Registry, 0, len(result))
		for _, r := range result {
			if r.OS == "all" || r.OS == os {
				filtered = append(filtered, r)
			}
		}
		result = filtered
	}

	if checksum {
		filtered := make([]domain.Registry, 0, len(result))
		for _, r := range result {
			if r.ChecksumType != "" {
				filtered = append(filtered, r)
			}
		}
		result = filtered
	}

	return result, err
}

// Get implements [registry.RegistryRepository].
func (reg *RegistryRepository) Get(name string, version string, checksum bool, os string) (r domain.Registry, err error) {
	registries, err := reg.List(name, checksum, os)
	if err != nil {
		return r, err
	}
	for _, registry := range registries {
		if registry.Version == version {
			return registry, nil
		}
	}
	return r, fmt.Errorf("registry %s version %s not found", name, version)
}
