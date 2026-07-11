package assembler

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

// AssemblerRepository resolves the transitive dependency graph for a package
// and returns an ordered list of packages to download.
type AssemblerRepository interface {
	// GetOrderedDependencies returns all transitive dependencies for
	// (name, version) in dependency order — dependencies before dependents.
	// The returned list excludes the root package itself.
	GetOrderedDependencies(name string, version string) ([]domain.Dependency, error)
}

// Service resolves the dependency graph and downloads all packages in parallel.
// It owns the full download workflow: resolve deps → resolve exact versions
// via the registry → download each via the silo in parallel.
type Service struct {
	assemblerRep AssemblerRepository
	registryRep  registry.RegistryRepository
	siloRep      silo.SiloRepository
}

// NewService creates a new assembler service.
func NewService(ar AssemblerRepository, rr registry.RegistryRepository, sr silo.SiloRepository) *Service {
	return &Service{
		assemblerRep: ar,
		registryRep:  rr,
		siloRep:      sr,
	}
}

// DownloadResult holds the outcome of a single package download.
type DownloadResult struct {
	Name       string
	Version    string
	Downloaded bool // true = fetched from network, false = skipped (already existed)
	Err        error
}

// Download resolves the transitive dependency graph for (name, version),
// resolves exact download URLs via the registry, then downloads all
// packages in parallel using goroutines.
// The root package itself is included in the download set.
func (s *Service) Download(name string, version string) ([]DownloadResult, error) {
	// Get all transitive dependencies (ordered, excluding root).
	deps, err := s.assemblerRep.GetOrderedDependencies(name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies for %s@%s: %w", name, version, err)
	}

	// Build the full download list: deps + root package itself.
	type downloadItem struct {
		name    string
		version string
	}

	items := make([]downloadItem, 0, len(deps)+1)

	// Add dependencies first.
	for _, dep := range deps {
		depVersion := extractVersion(dep.Version)
		items = append(items, downloadItem{name: dep.Name, version: depVersion})
	}

	// Add the root package last.
	items = append(items, downloadItem{name: name, version: version})

	// Deduplicate by name@version key.
	seen := make(map[string]bool)
	unique := items[:0]
	for _, item := range items {
		key := item.name + "@" + item.version
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, item)
	}
	items = unique

	// Download all packages in parallel.
	results := make([]DownloadResult, len(items))
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, itemName, itemVersion string) {
			defer wg.Done()

			results[idx] = DownloadResult{
				Name:    itemName,
				Version: itemVersion,
			}

			// Resolve the registry entry to get the URL and checksum.
			// checksum=false for now — we skip verification until checksums
			// are populated for all packages.
			r, err := s.registryRep.Get(itemName, itemVersion, false, runtime.GOOS)
			if err != nil {
				results[idx].Err = fmt.Errorf("registry resolve %s@%s: %w", itemName, itemVersion, err)
				return
			}

			// Download via the silo.
			downloaded, err := s.siloRep.Download(r.URL, r.ChecksumType, r.ChecksumValue)
			if err != nil {
				results[idx].Err = fmt.Errorf("download %s@%s: %w", itemName, itemVersion, err)
				return
			}
			results[idx].Downloaded = downloaded
		}(i, item.name, item.version)
	}

	wg.Wait()

	// Check for errors.
	var hasError bool
	for _, r := range results {
		if r.Err != nil {
			hasError = true
			break
		}
	}

	if hasError {
		return results, fmt.Errorf("one or more downloads failed")
	}

	return results, nil
}

// extractVersion parses a dependency version string in the format
// "exactVersion|constraint" and returns just the exact version part.
// If there is no pipe, the entire string is returned.
// If the string is empty, it returns empty.
func extractVersion(v string) string {
	if v == "" {
		return ""
	}
	if before, _, found := strings.Cut(v, "|"); found {
		return before
	}
	return v
}