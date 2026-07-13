package memory

import (
	"testing"
)

func TestGraphRepository_GetOrderedDependencies_PHP7_4(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("php", "7.4.33")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(php, 7.4.33) returned error: %v", err)
	}

	depNames := make(map[string]bool)
	for _, dep := range deps {
		depNames[dep.Name] = true
	}

	expectedDirect := []string{"openssl", "libxml2", "zlib", "oniguruma", "curl"}
	for _, name := range expectedDirect {
		if !depNames[name] {
			t.Errorf("expected dependency %q in result, not found", name)
		}
	}

	expectedTransitive := []string{"m4", "autoconf", "automake", "libtool", "perl"}
	for _, name := range expectedTransitive {
		if !depNames[name] {
			t.Errorf("expected transitive dependency %q in result, not found", name)
		}
	}

	if depNames["php"] {
		t.Error("root package 'php' should not appear in its own dependency list")
	}
}

func TestGraphRepository_GetOrderedDependencies_PHP8_1(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("php", "8.1.0")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(php, 8.1.0) returned error: %v", err)
	}

	depMap := make(map[string]string)
	for _, dep := range deps {
		depMap[dep.Name] = dep.Version
	}

	if v, ok := depMap["openssl"]; !ok || v != "1.1.1w" {
		t.Errorf("openssl version = %q, want 1.1.1w", v)
	}
	if v, ok := depMap["curl"]; !ok || v != "8.5.0" {
		t.Errorf("curl version = %q, want 8.5.0", v)
	}
}

func TestGraphRepository_GetOrderedDependencies_Zlib_NoDeps(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("zlib", "1.2.13")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(zlib, 1.2.13) returned error: %v", err)
	}

	if len(deps) != 0 {
		t.Fatalf("zlib should have zero dependencies, got %d: %v", len(deps), deps)
	}
}

func TestGraphRepository_GetOrderedDependencies_M4_NoDeps(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("m4", "1.4.19")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(m4, 1.4.19) returned error: %v", err)
	}

	if len(deps) != 0 {
		t.Fatalf("m4 should have zero dependencies, got %d: %v", len(deps), deps)
	}
}

func TestGraphRepository_GetOrderedDependencies_Deduplication(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("php", "7.4.33")
	if err != nil {
		t.Fatalf("GetOrderedDependencies returned error: %v", err)
	}

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

func TestGraphRepository_GetOrderedDependencies_Ordering(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("autoconf", "2.50")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(autoconf, 2.50) returned error: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("autoconf 2.50 should have 1 dependency (m4), got %d: %v", len(deps), deps)
	}
	if deps[0].Name != "m4" {
		t.Fatalf("expected m4, got %s", deps[0].Name)
	}
}

func TestGraphRepository_GetOrderedDependencies_UnknownPackage(t *testing.T) {
	repo := NewGraphRepository()

	deps, err := repo.GetOrderedDependencies("unknown", "1.0.0")
	if err != nil {
		t.Fatalf("GetOrderedDependencies(unknown) returned unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Fatalf("unknown package should have zero deps, got %d", len(deps))
	}
}
