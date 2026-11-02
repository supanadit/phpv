package disk

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFakeCache writes a minimal CMakeCache.txt with the given compiler
// paths and install prefix.
func writeFakeCache(t *testing.T, buildDir, cc, cxx, prefix string) string {
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
	writeFakeCache(t, buildDir, filepath.Join(dir, "ghost", "gcc"), "", filepath.Join(dir, "prefix"))

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
	writeFakeCache(t, buildDir, realCC, "", filepath.Join(dir, "prefix"))

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
	writeFakeCache(t, buildDir, realCC, "", filepath.Join(dir, "old-prefix"))

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
	writeFakeCache(t, buildDir, realCC, filepath.Join(dir, "ghost", "g++"), filepath.Join(dir, "prefix"))

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
	cachePath := writeFakeCache(t, buildDir, "/usr/bin/gcc", "/usr/bin/g++", "/tmp/prefix")

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
