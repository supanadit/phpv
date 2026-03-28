package memory

import (
	"fmt"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type SourceRepository struct{}

func NewSourceRepository() *SourceRepository {
	return &SourceRepository{}
}

func (r *SourceRepository) buildSource(name, version string) domain.Source {
	v := domain.ParseVersion(version)
	pattern, _ := domain.MatchPattern(name, v)
	url, _ := domain.BuildURL(pattern, v)
	return domain.Source{Name: name, Version: version, URL: url}
}

func (r *SourceRepository) buildRangeVersions(major, minor, startPatch, endPatch int, name string) []domain.Source {
	versions := make([]domain.Source, 0, endPatch-startPatch+1)
	for patch := startPatch; patch <= endPatch; patch++ {
		version := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		versions = append(versions, r.buildSource(name, version))
	}
	return versions
}

func (r *SourceRepository) GetVersions() ([]domain.Source, error) {
	versions := r.buildRangeVersions(8, 5, 0, 4, "php")
	versions = append(versions, r.buildRangeVersions(8, 4, 0, 19, "php")...)
	versions = append(versions, r.buildRangeVersions(8, 3, 0, 27, "php")...)
	versions = append(versions, r.buildRangeVersions(8, 2, 0, 29, "php")...)
	versions = append(versions, r.buildRangeVersions(8, 1, 0, 33, "php")...)
	versions = append(versions, r.buildRangeVersions(8, 0, 0, 30, "php")...)
	versions = append(versions, r.buildRangeVersions(7, 4, 0, 33, "php")...)
	versions = append(versions, r.buildRangeVersions(7, 3, 0, 33, "php")...)
	versions = append(versions, r.buildRangeVersions(7, 2, 0, 34, "php")...)
	versions = append(versions, r.buildRangeVersions(7, 1, 0, 33, "php")...)
	versions = append(versions, r.buildRangeVersions(7, 0, 0, 33, "php")...)
	versions = append(versions, r.buildRangeVersions(5, 6, 0, 40, "php")...)
	versions = append(versions, r.buildRangeVersions(5, 5, 0, 38, "php")...)
	versions = append(versions, r.buildRangeVersions(5, 4, 0, 45, "php")...)
	versions = append(versions, r.buildRangeVersions(5, 3, 0, 29, "php")...)
	versions = append(versions, r.buildRangeVersions(5, 2, 0, 17, "php")...)
	versions = append(versions, r.buildRangeVersions(5, 1, 0, 6, "php")...)
	versions = append(versions, r.buildRangeVersions(5, 0, 0, 5, "php")...)
	versions = append(versions, r.buildRangeVersions(4, 4, 0, 9, "php")...)
	versions = append(versions, r.buildRangeVersions(4, 3, 0, 11, "php")...)
	versions = append(versions, r.buildRangeVersions(4, 2, 0, 3, "php")...)
	versions = append(versions, r.buildRangeVersions(4, 1, 0, 2, "php")...)
	versions = append(versions, r.buildRangeVersions(4, 0, 0, 6, "php")...)

	versions = append(versions, r.buildRangeVersions(3, 27, 0, 6, "cmake")...)
	versions = append(versions, r.buildRangeVersions(3, 26, 0, 5, "cmake")...)
	versions = append(versions, r.buildRangeVersions(3, 25, 0, 3, "cmake")...)
	versions = append(versions, r.buildRangeVersions(3, 24, 0, 2, "cmake")...)
	versions = append(versions, r.buildRangeVersions(3, 23, 0, 3, "cmake")...)
	versions = append(versions, r.buildRangeVersions(3, 22, 0, 1, "cmake")...)
	versions = append(versions, r.buildRangeVersions(3, 21, 0, 4, "cmake")...)

	versions = append(versions, r.buildSource("zlib", "1.3.1"))
	versions = append(versions, r.buildSource("zlib", "1.2.13"))

	versions = append(versions, r.buildSource("re2c", "3.1"))
	versions = append(versions, r.buildSource("re2c", "2.2"))
	versions = append(versions, r.buildSource("re2c", "1.3"))
	versions = append(versions, r.buildSource("re2c", "0.16"))
	versions = append(versions, r.buildSource("re2c", "0.14"))

	versions = append(versions, r.buildSource("perl", "5.42.1"))
	versions = append(versions, r.buildSource("perl", "5.40.3"))
	versions = append(versions, r.buildSource("perl", "5.38.5"))
	versions = append(versions, r.buildSource("perl", "5.36.3"))
	versions = append(versions, r.buildSource("perl", "5.34.3"))
	versions = append(versions, r.buildSource("perl", "5.32.1"))
	versions = append(versions, r.buildSource("perl", "5.30.3"))
	versions = append(versions, r.buildSource("perl", "5.28.3"))
	versions = append(versions, r.buildSource("perl", "5.26.3"))
	versions = append(versions, r.buildSource("perl", "5.24.4"))
	versions = append(versions, r.buildSource("perl", "5.22.3"))
	versions = append(versions, r.buildSource("perl", "5.20.0"))
	versions = append(versions, r.buildSource("perl", "5.18.4"))
	versions = append(versions, r.buildSource("perl", "5.16.3"))
	versions = append(versions, r.buildSource("perl", "5.14.4"))
	versions = append(versions, r.buildSource("perl", "5.12.5"))
	versions = append(versions, r.buildSource("perl", "5.10.1"))
	versions = append(versions, r.buildSource("perl", "5.8.9"))
	versions = append(versions, r.buildSource("perl", "5.6.2"))
	versions = append(versions, r.buildSource("perl", "5.5.30"))
	versions = append(versions, r.buildSource("perl", "5.4.50"))

	versions = append(versions, r.buildSource("autoconf", "2.72"))
	versions = append(versions, r.buildSource("autoconf", "2.71"))
	versions = append(versions, r.buildSource("autoconf", "2.69"))
	versions = append(versions, r.buildSource("autoconf", "2.59"))
	versions = append(versions, r.buildSource("autoconf", "2.13"))

	versions = append(versions, r.buildSource("automake", "1.17"))
	versions = append(versions, r.buildSource("automake", "1.16.5"))
	versions = append(versions, r.buildSource("automake", "1.15.1"))
	versions = append(versions, r.buildSource("automake", "1.9.6"))
	versions = append(versions, r.buildSource("automake", "1.4-p6"))

	versions = append(versions, r.buildSource("bison", "1.28"))
	versions = append(versions, r.buildSource("bison", "1.35"))

	versions = append(versions, r.buildSource("curl", "8.10.1"))
	versions = append(versions, r.buildSource("curl", "7.88.1"))
	versions = append(versions, r.buildSource("curl", "7.20.0"))
	versions = append(versions, r.buildSource("curl", "7.12.1"))
	versions = append(versions, r.buildSource("curl", "7.12.0"))

	versions = append(versions, r.buildSource("flex", "2.5.39"))

	versions = append(versions, r.buildSource("libtool", "2.5.4"))
	versions = append(versions, r.buildSource("libtool", "2.4.7"))
	versions = append(versions, r.buildSource("libtool", "2.4.6"))
	versions = append(versions, r.buildSource("libtool", "1.5.26"))

	versions = append(versions, r.buildSource("libxml2", "2.12.7"))
	versions = append(versions, r.buildSource("libxml2", "2.11.7"))
	versions = append(versions, r.buildSource("libxml2", "2.9.14"))
	versions = append(versions, r.buildSource("libxml2", "2.6.30"))

	versions = append(versions, r.buildSource("m4", "1.4.19"))

	versions = append(versions, r.buildSource("oniguruma", "6.9.9"))
	versions = append(versions, r.buildSource("oniguruma", "6.9.8"))
	versions = append(versions, r.buildSource("oniguruma", "5.9.6"))

	versions = append(versions, r.buildSource("openssl", "3.3.2"))
	versions = append(versions, r.buildSource("openssl", "3.0.14"))
	versions = append(versions, r.buildSource("openssl", "1.1.1w"))
	versions = append(versions, r.buildSource("openssl", "1.0.1u"))
	versions = append(versions, r.buildSource("openssl", "0.9.8zh"))

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
