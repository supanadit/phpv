package compiler

import (
	"strings"
	"testing"
)

func TestResolveEmptyUsesPlatformDefault(t *testing.T) {
	tc, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve(\"\") returned error: %v", err)
	}
	if tc.Name != Default().Name {
		t.Fatalf("Resolve(\"\") = %q, want platform default %q", tc.Name, Default().Name)
	}
	if tc.CC == "" || tc.CXX == "" {
		t.Fatalf("default toolchain must define CC and CXX, got %+v", tc)
	}
}

func TestResolveEveryRegisteredCompiler(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("expected at least one registered compiler")
	}
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			t.Errorf("duplicate compiler name %q", name)
		}
		seen[name] = true

		tc, err := Resolve(name)
		if err != nil {
			t.Fatalf("Resolve(%q) returned error: %v", name, err)
		}
		if tc.Name != name {
			t.Errorf("Resolve(%q).Name = %q", name, tc.Name)
		}
		env := tc.Env()
		if len(env) != 2 || env[0] != "CC="+tc.CC || env[1] != "CXX="+tc.CXX {
			t.Errorf("Env() = %v, want [CC=%s CXX=%s]", env, tc.CC, tc.CXX)
		}
		if err := tc.Check(); err != nil && name != "zig" {
			// gcc/clang are the platform defaults and must exist on a build
			// host; zig is optional so a missing install is not a test failure.
			t.Errorf("Check() for %q: %v", name, err)
		}
	}
}

func TestResolveUnknownCompiler(t *testing.T) {
	if _, err := Resolve("tcc"); err == nil {
		t.Fatal("Resolve(\"tcc\") expected error, got nil")
	} else if !strings.Contains(err.Error(), "tcc") || !strings.Contains(err.Error(), "zig") {
		t.Errorf("error should name the bad value and the available compilers, got %v", err)
	}
}

func TestCheckMissingCompiler(t *testing.T) {
	tc := Toolchain{Name: "nope", CC: "phpv-no-such-compiler cc", CXX: "phpv-no-such-compiler c++", Hint: "install it"}
	err := tc.Check()
	if err == nil {
		t.Fatal("Check() expected error for a compiler that is not on PATH")
	}
	if !strings.Contains(err.Error(), "nope") || !strings.Contains(err.Error(), "install it") {
		t.Errorf("Check() error should name the toolchain and its install hint, got %v", err)
	}
}

func TestCheckUsesOnlyTheBinaryOfAMultiWordCC(t *testing.T) {
	tc := Toolchain{Name: "sh", CC: "sh cc", CXX: "sh c++"}
	if err := tc.Check(); err != nil {
		t.Fatalf("Check() should look up the first word of CC, got %v", err)
	}
}
