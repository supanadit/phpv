// Package compiler resolves the C/C++ toolchain phpv builds with.
//
// Selection order for a build is: explicit choice (phpv install --compiler),
// then the "compiler" config key, then the platform default. Toolchain.Name is
// the single identifier used by the CLI, the config file and the compiler flag
// rules (internal/repository/memory/compiler.go), so supporting another
// compiler means adding a registry entry here and, when it needs its own
// flags, a flag rule there — no changes to the build pipeline.
package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Toolchain describes one selectable C/C++ compiler.
type Toolchain struct {
	// Name identifies the compiler in the CLI, the config file and the
	// compiler flag rules.
	Name string
	// CC and CXX are exported as $CC and $CXX. They may contain arguments,
	// e.g. zig drives its bundled clang through "zig cc".
	CC  string
	CXX string
	// Hint tells the user how to obtain the compiler when it is missing.
	Hint string
}

var registry = []Toolchain{
	{Name: "gcc", CC: "gcc", CXX: "g++", Hint: "install your distribution's C compiler package (e.g. build-essential, gcc)"},
	{Name: "clang", CC: "clang", CXX: "clang++", Hint: "install clang from your distribution's package manager"},
	{Name: "zig", CC: "zig cc", CXX: "zig c++", Hint: "install zig from https://ziglang.org/download/"},
}

// Names returns the name of every registered compiler.
func Names() []string {
	names := make([]string, 0, len(registry))
	for _, tc := range registry {
		names = append(names, tc.Name)
	}
	return names
}

// Resolve returns the toolchain registered under name. An empty name selects
// the platform default; an unknown name is an error.
func Resolve(name string) (Toolchain, error) {
	if name == "" {
		return Default(), nil
	}
	for _, tc := range registry {
		if tc.Name == name {
			return tc, nil
		}
	}
	return Toolchain{}, fmt.Errorf("unknown compiler %q (available: %s)", name, strings.Join(Names(), ", "))
}

// Default returns the platform default toolchain: clang on macOS, gcc elsewhere.
func Default() Toolchain {
	name := "gcc"
	if runtime.GOOS == "darwin" {
		name = "clang"
	}
	tc, _ := Resolve(name)
	return tc
}

// Env returns the CC/CXX entries for this toolchain.
func (t Toolchain) Env() []string {
	return []string{"CC=" + t.CC, "CXX=" + t.CXX}
}

// Setenv exports CC/CXX into the current process so every child build process
// (./configure, make, phpize) uses this toolchain.
func (t Toolchain) Setenv() {
	os.Setenv("CC", t.CC)
	os.Setenv("CXX", t.CXX)
}

// Check returns an error when the compiler binary is not on PATH.
func (t Toolchain) Check() error {
	bin := t.CC
	if i := strings.IndexByte(bin, ' '); i >= 0 {
		bin = bin[:i]
	}
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("compiler %q is not available: %s", t.Name, t.Hint)
	}
	return nil
}
