package utils

import (
	"os/exec"
	"strings"
	"sync"
)

var (
	probeCache   = make(map[string]bool)
	probeCacheMu sync.RWMutex
)

// ProbeFlagCacheKey builds a cache key for a specific flag+compiler combination.
func ProbeFlagCacheKey(cc, flag string) string {
	return cc + "\x00" + flag
}

// ProbeCompilerFlag tests whether the given compiler accepts the flag.
// Uses a cache so repeated probes for the same flag are instant.
func ProbeCompilerFlag(cc string, flag string) bool {
	key := ProbeFlagCacheKey(cc, flag)

	probeCacheMu.RLock()
	if ok, exists := probeCache[key]; exists {
		probeCacheMu.RUnlock()
		return ok
	}
	probeCacheMu.RUnlock()

	// Compile /dev/null with the flag; exit 0 means supported
	ccPath := strings.Fields(cc)[0] // handle "zig cc -target ..." by using just "zig"
	cmd := exec.Command(ccPath, flag, "-c", "-x", "c", "/dev/null", "-o", "/dev/null")
	cmd.Stderr = nil
	err := cmd.Run()
	supported := err == nil

	probeCacheMu.Lock()
	probeCache[key] = supported
	probeCacheMu.Unlock()

	return supported
}

// FilterCompilerFlags probes the compiler and returns only supported flags from the candidates.
// Accepts a simple []string of flag names to probe.
func FilterCompilerFlags(cc string, candidates []string) []string {
	var result []string
	for _, flag := range candidates {
		if ProbeCompilerFlag(cc, flag) {
			result = append(result, flag)
		}
	}
	return result
}
