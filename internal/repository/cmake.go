package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CMakeCacheCompiler reads a CMAKE_<VAR>:FILEPATH value from a
// CMakeCache.txt (VAR: C_COMPILER or CXX_COMPILER). Returns "" when the
// file or key is absent.
func CMakeCacheCompiler(cachePath, varName string) string {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return ""
	}
	prefix := "CMAKE_" + varName + ":FILEPATH="
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

// InvalidateStaleCMakeCache removes a build dir whose CMake cache is stale:
// cached compiler paths that no longer exist on disk (toolchain removed or
// changed), or a cached install prefix that differs from the requested one.
// A missing cache is fine (fresh configure).
func InvalidateStaleCMakeCache(buildDir, wantPrefix string) error {
	cachePath := filepath.Join(buildDir, "CMakeCache.txt")
	if _, err := os.Stat(cachePath); err != nil {
		return nil // no cache yet: cmake will configure fresh
	}

	stale := false
	reason := ""
	for _, varName := range []string{"C_COMPILER", "CXX_COMPILER"} {
		cached := CMakeCacheCompiler(cachePath, varName)
		if cached == "" {
			continue
		}
		if _, err := os.Stat(cached); err != nil {
			stale = true
			reason = fmt.Sprintf("cached %s %q no longer exists", varName, cached)
			break
		}
	}
	if !stale {
		cachedPrefix := CMakeCacheString(cachePath, "INSTALL_PREFIX")
		if cachedPrefix != "" && cachedPrefix != wantPrefix {
			stale = true
			reason = fmt.Sprintf("cached install prefix %q differs from requested %q", cachedPrefix, wantPrefix)
		}
	}
	if !stale {
		return nil
	}

	if err := os.RemoveAll(buildDir); err != nil {
		return fmt.Errorf("remove stale build dir (%s): %w", reason, err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("recreate build dir: %w", err)
	}
	return nil
}

// CMakeCacheString reads a CMAKE_<VAR>:<TYPE>=value line of any type from
// a CMakeCache.txt. Returns "" when absent.
func CMakeCacheString(cachePath, varName string) string {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return ""
	}
	marker := "CMAKE_" + varName + ":"
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, marker) {
			idx := strings.Index(line, "=")
			if idx < 0 {
				return ""
			}
			return strings.TrimSpace(line[idx+1:])
		}
	}
	return ""
}
