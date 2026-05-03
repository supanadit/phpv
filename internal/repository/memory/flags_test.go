package memory

import (
	"testing"

	"github.com/supanadit/phpv/flagresolver"
)

func newTestFlagRepo() flagresolver.Repository {
	return NewFlagRepository(NewExtensionRepository())
}

func TestGetCompilerFlags_GCC_PHP5(t *testing.T) {
	repo := newTestFlagRepo()

	flags := repo.GetCompilerFlags("gcc", "5.6.40")
	if len(flags) == 0 {
		t.Fatal("expected flags for gcc PHP 5.6, got none")
	}

	found := false
	for _, f := range flags {
		if f == "-std=gnu11" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected -std=gnu11 in gcc PHP 5.6 flags")
	}

	found = false
	for _, f := range flags {
		if f == "-fno-strict-function-pointer-casts" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected -fno-strict-function-pointer-casts in gcc PHP 5.6 flags (GCC 15+ compat)")
	}

	// PHP 5 should not have -Wno-error as primary flag since it uses -std=gnu11 set
	if flags[0] != "-std=gnu11" {
		t.Errorf("expected first flag to be -std=gnu11, got %s", flags[0])
	}
}

func TestGetCompilerFlags_GCC_PHP7(t *testing.T) {
	repo := newTestFlagRepo()

	flags := repo.GetCompilerFlags("gcc", "7.4.33")
	if len(flags) == 0 {
		t.Fatal("expected flags for gcc PHP 7.4, got none")
	}

	found := false
	for _, f := range flags {
		if f == "-fno-strict-function-pointer-casts" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected -fno-strict-function-pointer-casts in gcc PHP 7.4 flags (GCC 15+ compat)")
	}
}

func TestGetCompilerFlags_GCC_PHP8(t *testing.T) {
	repo := newTestFlagRepo()

	flags := repo.GetCompilerFlags("gcc", "8.2.0")
	if len(flags) == 0 {
		t.Fatal("expected flags for gcc PHP 8.2, got none")
	}

	// PHP 8+ should have simpler flags
	if flags[0] != "-Wno-error" {
		t.Errorf("expected first flag to be -Wno-error, got %s", flags[0])
	}

	found := false
	for _, f := range flags {
		if f == "-fPIC" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected -fPIC in gcc PHP 8.2 flags")
	}

	// PHP 8+ should NOT have -fno-strict-function-pointer-casts
	for _, f := range flags {
		if f == "-fno-strict-function-pointer-casts" {
			t.Error("should not have -fno-strict-function-pointer-casts in gcc PHP 8+ flags")
		}
	}
}

func TestGetCompilerFlags_Zig(t *testing.T) {
	repo := newTestFlagRepo()

	flags := repo.GetCompilerFlags("zig", "8.0.30")
	if len(flags) == 0 {
		t.Fatal("expected flags for zig, got none")
	}

	expectedFlags := []string{"-std=gnu11", "-fPIC", "-Wno-error", "-fno-sanitize=undefined"}
	for _, expected := range expectedFlags {
		found := false
		for _, f := range flags {
			if f == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s in zig flags", expected)
		}
	}

	// Zig flags should be the same for different PHP versions
	flags2 := repo.GetCompilerFlags("zig", "5.4.45")
	if len(flags) != len(flags2) {
		t.Errorf("zig flags should be the same regardless of PHP version, got %d vs %d", len(flags), len(flags2))
	}
}

func TestGetCompilerFlags_UnknownCompiler(t *testing.T) {
	repo := newTestFlagRepo()

	flags := repo.GetCompilerFlags("clang", "8.0.0")
	if len(flags) != 0 {
		t.Errorf("expected empty flags for unknown compiler, got %v", flags)
	}
}

func TestCOnlyWarnings(t *testing.T) {
	// Verify that known C-only warning flags are in the map
	cOnlyFlags := []string{
		"-Wno-deprecated-non-prototype",
		"-Wno-implicit-function-declaration",
		"-Wno-array-parameter",
		"-Wstrict-prototypes",
		"-Wno-incompatible-pointer-types",
	}
	for _, flag := range cOnlyFlags {
		if !flagresolver.COnlyWarnings[flag] {
			t.Errorf("expected %s to be in COnlyWarnings", flag)
		}
	}

	// Verify that non-C-only flags are NOT in the map
	nonCOnlyFlags := []string{"-fPIC", "-Wno-error", "-std=gnu11"}
	for _, flag := range nonCOnlyFlags {
		if flagresolver.COnlyWarnings[flag] {
			t.Errorf("did not expect %s to be in COnlyWarnings", flag)
		}
	}
}

func TestCXXFlagsFromCFlags(t *testing.T) {
	tests := []struct {
		name       string
		cflags     []string
		isPHPBuild bool
		wantCXXStd bool
		wantCXXStdVal string
		wantFiltered []string
	}{
		{
			name:       "converts -std=gnu11 to -std=gnu++17",
			cflags:     []string{"-std=gnu11", "-fPIC"},
			isPHPBuild: false,
			wantCXXStd: true,
			wantCXXStdVal: "-std=gnu++17",
			wantFiltered: []string{"-fPIC"},
		},
		{
			name:       "removes C-only warnings",
			cflags:     []string{"-fPIC", "-Wno-deprecated-non-prototype", "-Wno-implicit-function-declaration", "-Wno-error"},
			isPHPBuild: false,
			wantFiltered: []string{"-fPIC", "-Wno-error"},
		},
		{
			name:       "PHP build ensures C++ standard",
			cflags:     []string{"-fPIC", "-Wno-error"},
			isPHPBuild: true,
			wantCXXStd: true,
			wantCXXStdVal: "-std=gnu++17",
		},
		{
			name:       "non-PHP build without -std does not add one",
			cflags:     []string{"-fPIC", "-Wno-error"},
			isPHPBuild: false,
			wantCXXStd: false,
		},
		{
			name:       "preserves existing C++ standard",
			cflags:     []string{"-std=gnu++14", "-fPIC"},
			isPHPBuild: true,
			wantCXXStd: true,
			wantCXXStdVal: "-std=gnu++14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flagresolver.CXXFlagsFromCFlags(tt.cflags, tt.isPHPBuild)

			if tt.wantCXXStd {
				found := false
				for _, f := range result {
					if f == tt.wantCXXStdVal {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected %s in result, got %v", tt.wantCXXStdVal, result)
				}
			} else {
				for _, f := range result {
					if stringsHasPrefix(f, "-std=c++") || stringsHasPrefix(f, "-std=gnu++") {
						t.Errorf("did not expect C++ standard flag, got %s in %v", f, result)
					}
				}
			}

			// Check filtered items are removed
			cOnlyFlags := []string{"-Wno-deprecated-non-prototype", "-Wno-implicit-function-declaration", "-Wno-array-parameter", "-Wstrict-prototypes", "-Wno-incompatible-pointer-types"}
			for _, f := range result {
				for _, cOnly := range cOnlyFlags {
					if f == cOnly {
						t.Errorf("C-only warning %s should have been removed, got %v", cOnly, result)
					}
				}
			}

			// Check wanted items are present
			for _, want := range tt.wantFiltered {
				found := false
				for _, f := range result {
					if f == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected %s in result, got %v", want, result)
				}
			}
		})
	}
}

func TestCXXFlagsFromCFlagsWithStd(t *testing.T) {
	tests := []struct {
		name       string
		cflags     []string
		isPHPBuild bool
		stdRule    flagresolver.CStdRule
		wantCXXStdVal string
	}{
		{
			name:       "uses stdRule CXXStd for -std=gnu11 replacement",
			cflags:     []string{"-std=gnu11", "-fPIC"},
			isPHPBuild: true,
			stdRule:    flagresolver.CStdRule{CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
			wantCXXStdVal: "-std=gnu++17",
		},
		{
			name:       "uses stdRule CXXStd when no C++ std present in PHP build",
			cflags:     []string{"-fPIC"},
			isPHPBuild: true,
			stdRule:    flagresolver.CStdRule{CStd: "-std=gnu11", CXXStd: "-std=gnu++20"},
			wantCXXStdVal: "-std=gnu++20",
		},
		{
			name:       "falls back to -std=gnu++17 when CXXStd empty",
			cflags:     []string{"-std=gnu11", "-fPIC"},
			isPHPBuild: true,
			stdRule:    flagresolver.CStdRule{CStd: "-std=gnu11", CXXStd: ""},
			wantCXXStdVal: "-std=gnu++17",
		},
		{
			name:       "preserves existing C++ standard from cflags",
			cflags:     []string{"-std=gnu++14", "-fPIC"},
			isPHPBuild: true,
			stdRule:    flagresolver.CStdRule{CStd: "-std=gnu11", CXXStd: "-std=gnu++17"},
			wantCXXStdVal: "-std=gnu++14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flagresolver.CXXFlagsFromCFlagsWithStd(tt.cflags, tt.isPHPBuild, tt.stdRule)

			found := false
			for _, f := range result {
				if f == tt.wantCXXStdVal {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %s in result, got %v", tt.wantCXXStdVal, result)
			}
		})
	}
}

func TestGetCompilerStdRule(t *testing.T) {
	repo := newTestFlagRepo()

	tests := []struct {
		phpVersion string
		wantCStd   string
		wantCXXStd string
	}{
		{"5.6.40", "-std=gnu11", "-std=gnu++17"},
		{"7.4.33", "-std=gnu11", "-std=gnu++17"},
		{"8.0.30", "-std=gnu11", "-std=gnu++17"},
		{"8.2.0", "-std=gnu11", "-std=gnu++17"},
		{"8.3.0", "-std=gnu11", "-std=gnu++17"},
	}

	for _, tt := range tests {
		t.Run(tt.phpVersion, func(t *testing.T) {
			rule := repo.GetCompilerStdRule(tt.phpVersion)
			if rule.CStd != tt.wantCStd {
				t.Errorf("GetCompilerStdRule(%q).CStd = %q, want %q", tt.phpVersion, rule.CStd, tt.wantCStd)
			}
			if rule.CXXStd != tt.wantCXXStd {
				t.Errorf("GetCompilerStdRule(%q).CXXStd = %q, want %q", tt.phpVersion, rule.CXXStd, tt.wantCXXStd)
			}
		})
	}
}

// stringsHasPrefix is a helper to avoid importing strings in tests.
func stringsHasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}