package disk

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// paths and install prefix.
type fakeCacheSpec struct {
	buildDir, cc, cxx, prefix string
}

func writeFakeCache(t *testing.T, spec fakeCacheSpec) string {
	buildDir, cc, cxx, prefix := spec.buildDir, spec.cc, spec.cxx, spec.prefix
	t.Helper()
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cache := "// CMakeCache.txt (test fixture)\n"
	if cc != "" {
		cache += "CMAKE_C_COMPILER:FILEPATH=" + cc + "\n"
	}
	if cxx != "" {
		cache += "CMAKE_CXX_COMPILER:FILEPATH=" + cxx + "\n"
	}
	cache += "CMAKE_INSTALL_PREFIX:PATH=" + prefix + "\n"
	cache += "CMAKE_BUILD_TYPE:STRING=Release\n"
	path := filepath.Join(buildDir, "CMakeCache.txt")
	if err := os.WriteFile(path, []byte(cache), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInvalidateStaleCMakeCache_MissingCompilerWipes(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	// The cached compiler references a path that does not exist on this
	// machine (the /usr/lib64/ccache/gcc scenario).
	writeFakeCache(t, fakeCacheSpec{buildDir: buildDir, cc: filepath.Join(dir, "ghost", "gcc"), cxx: "", prefix: filepath.Join(dir, "prefix")})

	if err := invalidateStaleCMakeCache(buildDir, filepath.Join(dir, "prefix")); err != nil {
		t.Fatalf("invalidateStaleCMakeCache: %v", err)
	}
	if _, err := os.Stat(filepath.Join(buildDir, "CMakeCache.txt")); !os.IsNotExist(err) {
		t.Error("stale cache should have been wiped")
	}
}

func TestInvalidateStaleCMakeCache_ValidCompilerKept(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	// A real, existing compiler binary.
	realCC := filepath.Join(dir, "bin", "gcc")
	if err := os.MkdirAll(filepath.Dir(realCC), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(realCC, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFakeCache(t, fakeCacheSpec{buildDir: buildDir, cc: realCC, cxx: "", prefix: filepath.Join(dir, "prefix")})

	if err := invalidateStaleCMakeCache(buildDir, filepath.Join(dir, "prefix")); err != nil {
		t.Fatalf("invalidateStaleCMakeCache: %v", err)
	}
	if _, err := os.Stat(filepath.Join(buildDir, "CMakeCache.txt")); err != nil {
		t.Error("valid cache must be kept (incremental rebuilds stay incremental)")
	}
}

func TestInvalidateStaleCMakeCache_PrefixChangeWipes(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	realCC := filepath.Join(dir, "bin", "gcc")
	if err := os.MkdirAll(filepath.Dir(realCC), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(realCC, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFakeCache(t, fakeCacheSpec{buildDir: buildDir, cc: realCC, cxx: "", prefix: filepath.Join(dir, "old-prefix")})

	if err := invalidateStaleCMakeCache(buildDir, filepath.Join(dir, "new-prefix")); err != nil {
		t.Fatalf("invalidateStaleCMakeCache: %v", err)
	}
	if _, err := os.Stat(filepath.Join(buildDir, "CMakeCache.txt")); !os.IsNotExist(err) {
		t.Error("cache with different install prefix should have been wiped")
	}
}

func TestInvalidateStaleCMakeCache_NoCacheIsNoop(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := invalidateStaleCMakeCache(buildDir, filepath.Join(dir, "prefix")); err != nil {
		t.Fatalf("invalidateStaleCMakeCache: %v", err)
	}
	// No cache, nothing to do; build dir must still exist for cmake.
	if info, err := os.Stat(buildDir); err != nil || !info.IsDir() {
		t.Error("build dir should remain for a fresh configure")
	}
}

func TestInvalidateStaleCMakeCache_CXXOnlyStale(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	// C compiler valid, C++ compiler stale: must still wipe.
	realCC := filepath.Join(dir, "bin", "gcc")
	if err := os.MkdirAll(filepath.Dir(realCC), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(realCC, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFakeCache(t, fakeCacheSpec{buildDir: buildDir, cc: realCC, cxx: filepath.Join(dir, "ghost", "g++"), prefix: filepath.Join(dir, "prefix")})

	if err := invalidateStaleCMakeCache(buildDir, filepath.Join(dir, "prefix")); err != nil {
		t.Fatalf("invalidateStaleCMakeCache: %v", err)
	}
	if _, err := os.Stat(filepath.Join(buildDir, "CMakeCache.txt")); !os.IsNotExist(err) {
		t.Error("cache with stale CXX compiler should have been wiped")
	}
}

func TestCmakeCacheCompiler_ParsesRealisticCache(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	cachePath := writeFakeCache(t, fakeCacheSpec{buildDir: buildDir, cc: "/usr/bin/gcc", cxx: "/usr/bin/g++", prefix: "/tmp/prefix"})

	// The real cache also carries AR/RANLIB and comment lines; make sure
	// the parser reads the right keys.
	data, _ := os.ReadFile(cachePath)
	augmented := string(data) + "CMAKE_C_COMPILER_AR:FILEPATH=/usr/bin/gcc-ar\n//ADVANCED property for variable: CMAKE_C_COMPILER\n"
	if err := os.WriteFile(cachePath, []byte(augmented), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := cmakeCacheCompiler(cachePath, "C_COMPILER"); got != "/usr/bin/gcc" {
		t.Errorf("cmakeCacheCompiler C = %q, want /usr/bin/gcc", got)
	}
	if got := cmakeCacheCompiler(cachePath, "CXX_COMPILER"); got != "/usr/bin/g++" {
		t.Errorf("cmakeCacheCompiler CXX = %q, want /usr/bin/g++", got)
	}
	if got := cmakeCacheString(cachePath, "INSTALL_PREFIX"); got != "/tmp/prefix" {
		t.Errorf("cmakeCacheString INSTALL_PREFIX = %q, want /tmp/prefix", got)
	}
	if got := cmakeCacheString(cachePath, "NONEXISTENT"); got != "" {
		t.Errorf("cmakeCacheString missing key = %q, want empty", got)
	}
}

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
