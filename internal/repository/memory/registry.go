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

func (reg *RegistryRepository) List(name string) (result []domain.Registry, err error) {
	// Define PHP version ranges. Gaps between ranges are intentional —
	// only versions within these ranges are generated.
	ranges := []repository.VersionRange{}
	ranges = append(ranges, repository.BuildMinorRanges(8, []repository.MinorRange{
		{Minor: 0, PatchEnd: 30},
		{Minor: 1, PatchEnd: 33},
		{Minor: 2, PatchEnd: 29},
		{Minor: 3, PatchEnd: 27},
		{Minor: 4, PatchEnd: 19},
	})...)
	ranges = append(ranges, repository.BuildMinorRanges(7, []repository.MinorRange{
		{Minor: 0, PatchEnd: 33},
		{Minor: 1, PatchEnd: 33},
		{Minor: 2, PatchEnd: 34},
		{Minor: 3, PatchEnd: 33},
		{Minor: 4, PatchEnd: 33},
	})...)
	ranges = append(ranges, repository.BuildMinorRanges(5, []repository.MinorRange{
		{Minor: 0, PatchEnd: 5},
		{Minor: 1, PatchEnd: 6},
		{Minor: 2, PatchEnd: 17},
		{Minor: 3, PatchEnd: 29},
		{Minor: 4, PatchEnd: 45},
		{Minor: 5, PatchEnd: 38},
		{Minor: 6, PatchEnd: 40},
	})...)
	ranges = append(ranges, repository.BuildMinorRanges(4, []repository.MinorRange{
		{Minor: 0, PatchEnd: 6},
		{Minor: 1, PatchEnd: 2},
		{Minor: 2, PatchEnd: 3},
		{Minor: 3, PatchEnd: 11},
		{Minor: 4, PatchEnd: 9},
	})...)

	// Specific versions to skip (gaps within ranges)
	skip := []string{}

	versions := repository.GenerateVersions(ranges, skip)
	for _, v := range versions {
		result = append(result, domain.Registry{
			Name:    "php",
			Source:  "official",
			URL:     fmt.Sprintf("https://www.php.net/distributions/php-%s.tar.gz", v),
			Version: v,
		})
	}
	return result, nil
}
