package repository

import (
	"runtime"
	"strconv"

	vsn "github.com/supanadit/phpv/pkg/version"
)

// serialMakeOnly names packages whose build systems are not parallel-safe:
// GNU autotools regenerating docs race under -j, and OpenSSL 1.0.x's
// crypto/Makefile builds the parent "lib" and subdirectory "subdirs"
// targets independently, so two concurrent `ar r libcrypto.a` invocations
// corrupt the archive (observed as `ar` dying with SIGBUS under -j16).
// OpenSSL 1.1.0+ rewrote its build system and is parallel-safe.
var serialMakeOnly = map[string]string{
	"automake": "",
	"autoconf": "",
	"libtool":  "",
	"openssl":  "1.1.0", // serial only for versions < 1.1.0
}

// MakeArgs returns the make arguments for (name, version). Versions before
// the package's serialMakeOnly threshold build with -j1. jobs < 1 falls back
// to the historical default of 4.
func MakeArgs(name, version string, jobs int) []string {
	if jobs < 1 {
		jobs = 4
	}
	if threshold, ok := serialMakeOnly[name]; ok {
		if threshold == "" || vsn.CompareVersions(version, threshold) < 0 {
			return []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}
		}
	}
	return []string{"-j" + strconv.Itoa(jobs)}
}

// TODO: Hardcoded OS/arch OpenSSL target
// Description: We should instead make this dynamic like "List" in "internal/repository/memory/registry.go"
func OpensslTarget() string {
	if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			return "darwin64-arm64-cc"
		}
		return "darwin64-x86_64-cc"
	}
	return "linux-x86_64"
}
