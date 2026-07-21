package assembler

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// resolveDepPlaceholders replaces {{dep:NAME}} placeholders in configure flags
// with the install prefix of the named dependency. This is used by the patcher
// to inject dependency paths (e.g., curl's --with-openssl={{dep:openssl}}).
func (s *Service) resolveDepPlaceholders(flags []string, deps []domain.Dependency) []string {
	result := make([]string, len(flags))
	for i, flag := range flags {
		for _, dep := range deps {
			placeholder := "{{dep:" + dep.Name + "}}"
			if !strings.Contains(flag, placeholder) {
				continue
			}
			ver := dep.Version
			if idx := strings.Index(ver, "|"); idx != -1 {
				ver = ver[:idx]
			}
			if ver == "" {
				continue
			}
			prefix := s.silo.PackagePrefix(dep.Name, ver)
			if fi, err := os.Stat(prefix); err == nil && fi.IsDir() {
				flag = strings.ReplaceAll(flag, placeholder, prefix)
			}
		}
		result[i] = flag
	}
	return result
}

// resolveDependencyFlags resolves package-relative configure flags
// (e.g. --with-openssl, --with-zlib, --with-libxml) to absolute paths
// pointing at the locally-built dependency install prefix. Without this
// resolution, PHP's ./configure would use the system OpenSSL/zlib/libxml2
// (which is incompatible with PHP 7.4 for OpenSSL 3.x) instead of the
// version that phpv just built from source.
//
// For each recognized flag, the function looks up the matching dep in the
// build plan, resolves its install prefix via silo.PackagePrefix, and
// replaces the flag with the absolute-path variant. Falls back to
// scanning the dep dir for any installed version if the exact pin is
// not found. Falls back to keeping the original flag if no local build
// is available (letting PHP use the system library).
func (s *Service) resolveDependencyFlags(name, phpVersion string, flags []string, deps []domain.Dependency) []string {
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		switch {
		case flag == "--with-openssl" || flag == "--with-ssl":
			opensslFlag := "--with-openssl"
			if name != "php" {
				opensslFlag = "--with-ssl"
			}
			if path := s.findDepPrefix(deps, "openssl", "include/openssl"); path != "" {
				result = append(result, opensslFlag+"="+path)
			} else {
				result = append(result, flag)
			}

		case flag == "--with-zlib" && (name == "libxml2" || name == "curl"):
			if path := s.findDepPrefix(deps, "zlib", "include/zlib.h"); path != "" {
				result = append(result, "--with-zlib="+path)
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-libxml") || strings.HasPrefix(flag, "--enable-libxml"):
			if path := s.findDepPrefix(deps, "libxml2", "lib/pkgconfig/libxml-2.0.pc"); path != "" {
				if strings.HasPrefix(flag, "--enable-libxml") {
					result = append(result, flag)
					result = append(result, "--with-libxml-dir="+path)
				} else {
					result = append(result, flag+"="+path)
				}
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-curl"):
			if path := s.findDepPrefix(deps, "curl", "lib/libcurl.so"); path != "" {
				result = append(result, "--with-curl="+path)
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-pdo-pgsql"):
			if flag == "--with-pdo-pgsql" || flag == "--with-pdo-pgsql=yes" {
				wrapperPath := filepath.Join(s.silo.PackagePrefix("php", phpVersion), "wrapper")
				pgConfigPath := filepath.Join(wrapperPath, "bin", "pg_config")
				if fi, err := os.Stat(pgConfigPath); err == nil && !fi.IsDir() {
					result = append(result, "--with-pdo-pgsql="+wrapperPath)
				} else {
					result = append(result, flag)
				}
			} else {
				result = append(result, flag)
			}

		case strings.HasPrefix(flag, "--with-pgsql"):
			if flag == "--with-pgsql" || flag == "--with-pgsql=yes" {
				wrapperPath := filepath.Join(s.silo.PackagePrefix("php", phpVersion), "wrapper")
				pgConfigPath := filepath.Join(wrapperPath, "bin", "pg_config")
				if fi, err := os.Stat(pgConfigPath); err == nil && !fi.IsDir() {
					result = append(result, "--with-pgsql="+wrapperPath)
				} else {
					result = append(result, flag)
				}
			} else {
				result = append(result, flag)
			}

		case flag == "--enable-intl":
			result = append(result, flag)
			if path := s.resolveICUPath(deps); path != "" {
				result = append(result, "--with-icu-dir="+path)
			}

		default:
			result = append(result, flag)
		}
	}
	return result
}

// findDepPrefix locates the install prefix for a dependency by looking up
// the exact version in the build plan's dep list, then falling back to
// scanning the dep dir for any installed version. Returns "" if no local
// build is available, signaling the caller to keep the original flag.
func (s *Service) findDepPrefix(deps []domain.Dependency, depName, sentinelPath string) string {
	for _, dep := range deps {
		if dep.Name != depName {
			continue
		}
		ver := dep.Version
		if idx := strings.Index(ver, "|"); idx != -1 {
			ver = ver[:idx]
		}
		if ver == "" {
			continue
		}
		prefix := s.silo.PackagePrefix(depName, ver)
		if fi, err := os.Stat(prefix); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(prefix, sentinelPath)); err == nil {
				return prefix
			}
		}
	}
	depDir := filepath.Join(resolvePHPVRoot(), "packages", depName)
	if entries, err := os.ReadDir(depDir); err == nil {
		var candidates []string
		for _, entry := range entries {
			candidate := filepath.Join(depDir, entry.Name())
			if _, err := os.Stat(filepath.Join(candidate, sentinelPath)); err == nil {
				candidates = append(candidates, entry.Name())
			}
		}
		if len(candidates) > 0 {
			sort.Slice(candidates, func(i, j int) bool {
				return compareVersions(candidates[i], candidates[j]) > 0
			})
			return filepath.Join(depDir, candidates[0])
		}
	}
	return ""
}

// resolveICUPath finds the bundled ICU installation directory. ICU is
// required by the intl extension and must be passed via --with-icu-dir
// to PHP's configure. Searches the dependency directory for any ICU
// version that contains include/unicode/urename.h.
func (s *Service) resolveICUPath(deps []domain.Dependency) string {
	return s.findDepPrefix(deps, "icu", "include/unicode/urename.h")
}
