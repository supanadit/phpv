package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/pattern"
)

type SourceRepository struct {
	patternRegistry *pattern.PatternRegistry
}

func NewSourceRepository() *SourceRepository {
	registry := pattern.NewPatternRegistry()
	registry.RegisterPatterns(pattern.DefaultURLPatterns)
	return &SourceRepository{
		patternRegistry: registry,
	}
}

func (r *SourceRepository) buildSource(name, version, sourceType string) domain.Source {
	v := pattern.ParseVersion(version)
	urlPattern, _ := r.patternRegistry.MatchPattern(name, v)
	url, _ := pattern.BuildURL(urlPattern, v)
	return domain.Source{Name: name, Version: version, URL: url, Type: sourceType}
}

func (r *SourceRepository) GetSources(name, version string) ([]domain.Source, error) {
	v := pattern.ParseVersion(version)
	patterns, err := r.patternRegistry.MatchPatterns(name, v)
	if err != nil {
		return nil, err
	}
	var sources []domain.Source
	for _, p := range patterns {
		url, err := pattern.BuildURL(p, v)
		if err != nil {
			continue
		}
		sources = append(sources, domain.Source{
			Name:    name,
			Version: version,
			URL:     url,
			Type:    p.Type,
			OS:      p.OS,
			Arch:    p.Arch,
		})
	}
	return sources, nil
}

func (r *SourceRepository) buildRangeVersions(major, minor, startPatch, endPatch int, name string, sourceType string) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		version := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		versions = append(versions, r.buildSource(name, version, sourceType))
	}
	return versions
}

func (r *SourceRepository) GetVersions() ([]domain.Source, error) {
	sourceType := domain.SourceTypeSource
	binaryType := domain.SourceTypeBinary

	versions := r.buildRangeVersions(8, 5, 0, 4, "php", sourceType)
	versions = append(versions, r.buildRangeVersions(8, 4, 0, 19, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(8, 3, 0, 27, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(8, 2, 0, 29, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(8, 1, 0, 33, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(8, 0, 0, 30, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(7, 4, 0, 33, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(7, 3, 0, 33, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(7, 2, 0, 34, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(7, 1, 0, 33, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(7, 0, 0, 33, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(5, 6, 0, 40, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(5, 5, 0, 38, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(5, 4, 0, 45, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(5, 3, 0, 29, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(5, 2, 0, 17, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(5, 1, 0, 6, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(5, 0, 0, 5, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(4, 4, 0, 9, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(4, 3, 0, 11, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(4, 2, 0, 3, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(4, 1, 0, 2, "php", sourceType)...)
	versions = append(versions, r.buildRangeVersions(4, 0, 0, 6, "php", sourceType)...)

	versions = append(versions, r.buildRangeVersions(3, 27, 0, 6, "cmake", binaryType)...)
	versions = append(versions, r.buildRangeVersions(3, 26, 0, 5, "cmake", binaryType)...)
	versions = append(versions, r.buildRangeVersions(3, 25, 0, 3, "cmake", binaryType)...)
	versions = append(versions, r.buildRangeVersions(3, 24, 0, 2, "cmake", binaryType)...)
	versions = append(versions, r.buildRangeVersions(3, 23, 0, 3, "cmake", binaryType)...)
	versions = append(versions, r.buildRangeVersions(3, 22, 0, 1, "cmake", binaryType)...)
	versions = append(versions, r.buildRangeVersions(3, 21, 0, 4, "cmake", binaryType)...)

	versions = append(versions, r.buildSource("zlib", "1.3.1", sourceType))
	versions = append(versions, r.buildSource("zlib", "1.2.13", sourceType))

	versions = append(versions, r.buildSource("re2c", "3.1", sourceType))
	versions = append(versions, r.buildSource("re2c", "2.2", sourceType))
	versions = append(versions, r.buildSource("re2c", "1.3", sourceType))
	versions = append(versions, r.buildSource("re2c", "0.16", sourceType))
	versions = append(versions, r.buildSource("re2c", "0.14", sourceType))

	versions = append(versions, r.buildSource("perl", "5.42.1", binaryType))
	versions = append(versions, r.buildSource("perl", "5.40.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.38.5", binaryType))
	versions = append(versions, r.buildSource("perl", "5.36.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.34.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.32.1", binaryType))
	versions = append(versions, r.buildSource("perl", "5.30.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.28.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.26.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.24.4", binaryType))
	versions = append(versions, r.buildSource("perl", "5.22.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.20.0", binaryType))
	versions = append(versions, r.buildSource("perl", "5.18.4", binaryType))
	versions = append(versions, r.buildSource("perl", "5.16.3", binaryType))
	versions = append(versions, r.buildSource("perl", "5.14.4", binaryType))
	versions = append(versions, r.buildSource("perl", "5.12.5", binaryType))
	versions = append(versions, r.buildSource("perl", "5.10.1", binaryType))
	versions = append(versions, r.buildSource("perl", "5.8.9", binaryType))
	versions = append(versions, r.buildSource("perl", "5.6.2", binaryType))
	versions = append(versions, r.buildSource("perl", "5.5.30", binaryType))
	versions = append(versions, r.buildSource("perl", "5.4.50", binaryType))

	versions = append(versions, r.buildSource("autoconf", "2.72", sourceType))
	versions = append(versions, r.buildSource("autoconf", "2.71", sourceType))
	versions = append(versions, r.buildSource("autoconf", "2.69", sourceType))
	versions = append(versions, r.buildSource("autoconf", "2.59", sourceType))
	versions = append(versions, r.buildSource("autoconf", "2.13", sourceType))

	versions = append(versions, r.buildSource("automake", "1.17", sourceType))
	versions = append(versions, r.buildSource("automake", "1.16.5", sourceType))
	versions = append(versions, r.buildSource("automake", "1.15.1", sourceType))
	versions = append(versions, r.buildSource("automake", "1.9.6", sourceType))
	versions = append(versions, r.buildSource("automake", "1.4-p6", sourceType))

	versions = append(versions, r.buildSource("bison", "1.28", sourceType))
	versions = append(versions, r.buildSource("bison", "1.35", sourceType))

	versions = append(versions, r.buildSource("curl", "8.10.1", sourceType))
	versions = append(versions, r.buildSource("curl", "7.88.1", sourceType))
	versions = append(versions, r.buildSource("curl", "7.20.0", sourceType))
	versions = append(versions, r.buildSource("curl", "7.12.1", sourceType))
	versions = append(versions, r.buildSource("curl", "7.12.0", sourceType))

	versions = append(versions, r.buildSource("flex", "2.5.39", sourceType))

	versions = append(versions, r.buildSource("libtool", "2.5.4", sourceType))
	versions = append(versions, r.buildSource("libtool", "2.4.7", sourceType))
	versions = append(versions, r.buildSource("libtool", "2.4.6", sourceType))
	versions = append(versions, r.buildSource("libtool", "1.5.26", sourceType))

	versions = append(versions, r.buildSource("libxml2", "2.12.7", sourceType))
	versions = append(versions, r.buildSource("libxml2", "2.11.7", sourceType))
	versions = append(versions, r.buildSource("libxml2", "2.9.14", sourceType))
	versions = append(versions, r.buildSource("libxml2", "2.6.30", sourceType))

	versions = append(versions, r.buildSource("m4", "1.4.19", sourceType))

	versions = append(versions, r.buildSource("oniguruma", "6.9.9", sourceType))
	versions = append(versions, r.buildSource("oniguruma", "6.9.8", sourceType))
	versions = append(versions, r.buildSource("oniguruma", "5.9.6", sourceType))

	versions = append(versions, r.buildSource("openssl", "3.3.2", sourceType))
	versions = append(versions, r.buildSource("openssl", "3.0.14", sourceType))
	versions = append(versions, r.buildSource("openssl", "1.1.1w", sourceType))
	versions = append(versions, r.buildSource("openssl", "1.0.1u", sourceType))
	versions = append(versions, r.buildSource("openssl", "0.9.8zh", sourceType))

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
