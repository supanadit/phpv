package memory

import (
	"testing"
)

func TestAssemblerRepository_GetOrderedDependencies_PHP7_4(t *testing.T) {
	repo := NewAssemblerRepository()

	deps, err := repo.GetOrderedDependencies("php", "7.4.33")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(php, 7.4.33) returned error: %v", err)
	}

	// PHP 7.4.x matches the ">=7.1.0 <8.1.0" constraint, which has:
	// openssl, libxml2, zlib, oniguruma, curl
	// Each of those has transitive deps (m4, autoconf, automake, libtool, perl)
	// The root "php" itself should NOT appear in the result.

	depNames := make(map[string]bool)
	for _, dep := range deps {
		depNames[dep.Name] = true
	}

	// Direct deps of php 7.4
	expectedDirect := []string{"openssl", "libxml2", "zlib", "oniguruma", "curl"}
	for _, name := range expectedDirect {
		if !depNames[name] {
			t.Errorf("expected dependency %q in result, not found", name)
		}
	}

	// Transitive deps (from openssl, libxml2, curl, etc.)
	expectedTransitive := []string{"m4", "autoconf", "automake", "libtool", "perl"}
	for _, name := range expectedTransitive {
		if !depNames[name] {
			t.Errorf("expected transitive dependency %q in result, not found", name)
		}
	}

	// The root package itself must not appear
	if depNames["php"] {
		t.Error("root package 'php' should not appear in its own dependency list")
	}
}

func TestAssemblerRepository_GetOrderedDependencies_PHP8_1(t *testing.T) {
	repo := NewAssemblerRepository()

	deps, err := repo.GetOrderedDependencies("php", "8.1.0")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(php, 8.1.0) returned error: %v", err)
	}

	// PHP 8.1.0 matches ">=8.1.0 <8.2.0"
	// Deps: openssl@1.1.1w, libxml2@2.9.14, zlib@1.2.13, oniguruma@6.9.9, curl@8.5.0
	depMap := make(map[string]string)
	for _, dep := range deps {
		depMap[dep.Name] = dep.Version
	}

	// Check exact versions for direct deps
	if v, ok := depMap["openssl"]; !ok || v != "1.1.1w" {
		t.Errorf("openssl version = %q, want 1.1.1w", v)
	}
	if v, ok := depMap["curl"]; !ok || v != "8.5.0" {
		t.Errorf("curl version = %q, want 8.5.0", v)
	}
}

func TestAssemblerRepository_GetOrderedDependencies_Zlib_NoDeps(t *testing.T) {
	repo := NewAssemblerRepository()

	deps, err := repo.GetOrderedDependencies("zlib", "1.2.13")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(zlib, 1.2.13) returned error: %v", err)
	}

	if len(deps) != 0 {
		t.Fatalf("zlib should have zero dependencies, got %d: %v", len(deps), deps)
	}
}

func TestAssemblerRepository_GetOrderedDependencies_M4_NoDeps(t *testing.T) {
	repo := NewAssemblerRepository()

	deps, err := repo.GetOrderedDependencies("m4", "1.4.19")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(m4, 1.4.19) returned error: %v", err)
	}

	if len(deps) != 0 {
		t.Fatalf("m4 should have zero dependencies, got %d: %v", len(deps), deps)
	}
}

func TestAssemblerRepository_GetOrderedDependencies_Deduplication(t *testing.T) {
	repo := NewAssemblerRepository()

	deps, err := repo.GetOrderedDependencies("php", "7.4.33")
	if err != nil {
		t.Fatalf("GetOrderedDependencies returned error: %v", err)
	}

	// m4 is a dependency of autoconf, automake, libtool, and others.
	// It should appear only once.
	seen := make(map[string]int)
	for _, dep := range deps {
		seen[dep.Name]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("dependency %q appears %d times, should appear only once", name, count)
		}
	}
}

func TestAssemblerRepository_GetOrderedDependencies_Ordering(t *testing.T) {
	repo := NewAssemblerRepository()

	// autoconf 2.69 matches the ">=2.69" constraint which has empty deps,
	// overriding the Default (which has m4). So 2.69 has zero deps.
	// We test with a version below 2.69 to hit the Default path.
	deps, err := repo.GetOrderedDependencies("autoconf", "2.50")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(autoconf, 2.50) returned error: %v", err)
	}

	// autoconf's Default has m4, so m4 should be the only dep.
	if len(deps) != 1 {
		t.Fatalf("autoconf 2.50 should have 1 dependency (m4), got %d: %v", len(deps), deps)
	}
	if deps[0].Name != "m4" {
		t.Fatalf("expected m4, got %s", deps[0].Name)
	}
}

func TestAssemblerRepository_GetOrderedDependencies_UnknownPackage(t *testing.T) {
	repo := NewAssemblerRepository()

	deps, err := repo.GetOrderedDependencies("unknown", "1.0.0")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(unknown) returned unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Fatalf("unknown package should have zero deps, got %d", len(deps))
	}
}