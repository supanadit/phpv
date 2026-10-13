package graph

import (
	"reflect"
	"testing"

	"github.com/supanadit/phpv/domain"
)

// TestCXXFlagsFromCFlagsWithStd_TranslatesCStd makes sure no C-only standard
// flag (-std=gnu89 for PHP 4, -std=gnu11 otherwise) leaks into CXXFLAGS, where
// g++/clang++ would reject or ignore it.
func TestCXXFlagsFromCFlagsWithStd_TranslatesCStd(t *testing.T) {
	rule := domain.CompilerRule{CStd: "-std=gnu89", CXXStd: "-std=gnu++17"}
	cflags := []string{
		"-fPIC",
		"-std=gnu11", // from the per-compiler flag rules
		"-std=gnu89", // CStd appended last by the assembler
		"-Wno-implicit-function-declaration",
		"-fvisibility=hidden",
	}

	got := CXXFlagsFromCFlagsWithStd(cflags, true, rule)
	want := []string{"-fPIC", "-std=gnu++17", "-fvisibility=hidden"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CXXFlagsFromCFlagsWithStd() = %v, want %v", got, want)
	}
}

// TestCXXFlagsFromCFlagsWithStd_KeepsExplicitCXXStd verifies an already-C++
// standard passes through untouched.
func TestCXXFlagsFromCFlagsWithStd_KeepsExplicitCXXStd(t *testing.T) {
	rule := domain.CompilerRule{CStd: "-std=gnu11", CXXStd: "-std=gnu++17"}
	got := CXXFlagsFromCFlagsWithStd([]string{"-std=gnu++20", "-O2"}, false, rule)
	want := []string{"-std=gnu++20", "-O2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CXXFlagsFromCFlagsWithStd() = %v, want %v", got, want)
	}
}
