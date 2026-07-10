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
	ranges := []repository.VersionRange{
		{From: "8.4.0", To: "8.4.19"},
		{From: "8.3.0", To: "8.3.27"},
		{From: "8.2.0", To: "8.2.29"},
		{From: "8.1.0", To: "8.1.33"},
		{From: "8.0.0", To: "8.0.30"},
		{From: "7.4.0", To: "7.4.33"},
		{From: "7.3.0", To: "7.3.33"},
		{From: "7.2.0", To: "7.2.34"},
		{From: "7.1.0", To: "7.1.33"},
		{From: "7.0.0", To: "7.0.33"},
		{From: "5.6.0", To: "5.6.40"},
		{From: "5.5.0", To: "5.5.38"},
		{From: "5.4.0", To: "5.4.45"},
		{From: "5.3.0", To: "5.3.29"},
		{From: "5.2.0", To: "5.2.17"},
		{From: "5.1.0", To: "5.1.6"},
		{From: "5.0.0", To: "5.0.5"},
		{From: "4.4.0", To: "4.4.9"},
		{From: "4.3.0", To: "4.3.11"},
		{From: "4.2.0", To: "4.2.3"},
		{From: "4.1.0", To: "4.1.2"},
		{From: "4.0.0", To: "4.0.6"},
	}

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
