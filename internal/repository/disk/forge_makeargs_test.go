package disk

import (
	"reflect"
	"testing"
)

// TestMakeArgs_SerialPackages pins the -j1 carve-out for build systems that
// are not parallel-safe. OpenSSL 1.0.x races two `ar r libcrypto.a`
// invocations under -j (parent "lib" vs subdirectory "subdirs" target),
// observed as `ar` dying with SIGBUS; 1.1.0+ rewrote its build system and
// builds fine in parallel. GNU autotools doc regeneration races regardless
// of version.
func TestMakeArgs_SerialPackages(t *testing.T) {
	tests := []struct {
		name    string
		version string
		jobs    int
		want    []string
	}{
		// OpenSSL 1.0.x: serial regardless of requested jobs.
		{"openssl", "1.0.1u", 16, []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}},
		{"openssl", "1.0.2u", 16, []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}},
		{"openssl", "0.9.8zh", 16, []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}},
		// OpenSSL 1.1.0+ is parallel-safe.
		{"openssl", "1.1.0", 16, []string{"-j16"}},
		{"openssl", "1.1.1w", 16, []string{"-j16"}},
		{"openssl", "3.3.2", 16, []string{"-j16"}},
		// Autotools doc tooling races at any version.
		{"automake", "1.15.1", 16, []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}},
		{"autoconf", "2.69", 16, []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}},
		{"libtool", "2.4.6", 16, []string{"-j1", "MAKEINFO=true", "HELP2MAN=true"}},
		// Everything else keeps the requested parallelism.
		{"curl", "7.20.0", 16, []string{"-j16"}},
		{"zlib", "1.2.13", 16, []string{"-j16"}},
		{"php", "5.6.40", 16, []string{"-j16"}},
		// Unset/zero jobs fall back to the default parallelism.
		{"curl", "7.20.0", 0, []string{"-j4"}},
	}
	for _, tt := range tests {
		got := makeArgs(tt.name, tt.version, tt.jobs)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("makeArgs(%s, %s, %d) = %q, want %q", tt.name, tt.version, tt.jobs, got, tt.want)
		}
	}
}
