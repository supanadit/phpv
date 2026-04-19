package memory

import (
	"testing"
)

func TestMemoryAssembler_GetDependencies(t *testing.T) {
	repo := NewMemoryAssemblerRepository()

	tests := []struct {
		name         string
		packageName  string
		version      string
		wantLen      int
		wantContains []string
	}{
		{
			name:         "PHP 8.3 minimal (no default dependencies)",
			packageName:  "php",
			version:      "8.3.0",
			wantLen:      0,
			wantContains: []string{},
		},
		{
			name:         "PHP 8.2 minimal (no default dependencies)",
			packageName:  "php",
			version:      "8.2.0",
			wantLen:      0,
			wantContains: []string{},
		},
		{
			name:         "PHP 8.1 requires OpenSSL 1.1.x, libxml2, zlib, oniguruma, curl, icu",
			packageName:  "php",
			version:      "8.1.0",
			wantLen:      6,
			wantContains: []string{"openssl", "libxml2", "zlib", "oniguruma", "curl", "icu"},
		},
		{
			name:         "PHP 8.0 requires OpenSSL 1.1.x, libxml2, zlib, oniguruma, curl, icu",
			packageName:  "php",
			version:      "8.0.0",
			wantLen:      6,
			wantContains: []string{"openssl", "libxml2", "zlib", "oniguruma", "curl", "icu"},
		},
		{
			name:         "PHP 7.4 requires OpenSSL 1.1.x, libxml2, zlib, oniguruma, curl, icu",
			packageName:  "php",
			version:      "7.4.0",
			wantLen:      6,
			wantContains: []string{"openssl", "libxml2", "zlib", "oniguruma", "curl", "icu"},
		},
		{
			name:         "PHP 5.6 requires OpenSSL 1.1.x, libxml2, zlib, oniguruma, curl",
			packageName:  "php",
			version:      "5.6.0",
			wantLen:      6,
			wantContains: []string{"openssl", "libxml2", "zlib", "oniguruma", "curl", "icu"},
		},
		{
			name:         "PHP 5.4 requires OpenSSL 1.0.x, libxml2, zlib, oniguruma, curl",
			packageName:  "php",
			version:      "5.4.0",
			wantLen:      6,
			wantContains: []string{"openssl", "libxml2", "zlib", "oniguruma", "curl", "icu"},
		},
		{
			name:         "OpenSSL 3.x has build deps",
			packageName:  "openssl",
			version:      "3.3.2",
			wantLen:      4,
			wantContains: []string{"m4", "autoconf", "automake", "libtool"},
		},
		{
			name:         "OpenSSL 1.x has perl and build deps",
			packageName:  "openssl",
			version:      "1.1.1w",
			wantLen:      5,
			wantContains: []string{"perl", "m4", "autoconf", "automake", "libtool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, err := repo.GetDependencies(tt.packageName, tt.version)
			if err != nil {
				t.Fatalf("GetDependencies() error = %v", err)
			}

			if len(deps) != tt.wantLen {
				t.Errorf("GetDependencies() got %d deps, want %d", len(deps), tt.wantLen)
			}

			depNames := make(map[string]bool)
			for _, dep := range deps {
				depNames[dep.Name] = true
			}

			for _, want := range tt.wantContains {
				if !depNames[want] {
					t.Errorf("GetDependencies() missing expected dep: %s", want)
				}
			}
		})
	}
}

func TestMemoryAssembler_GetGraph(t *testing.T) {
	repo := NewMemoryAssemblerRepository()

	t.Run("PHP 8.3 minimal graph (no default dependencies)", func(t *testing.T) {
		graph, err := repo.GetGraph("php", "8.3.0")
		if err != nil {
			t.Fatalf("GetGraph() error = %v", err)
		}

		if _, ok := graph["php"]; !ok {
			t.Error("GetGraph() should contain php as root package")
		}

		if _, ok := graph["openssl"]; ok {
			t.Error("GetGraph() should NOT contain openssl dependency for minimal install")
		}

		if _, ok := graph["libxml2"]; ok {
			t.Error("GetGraph() should NOT contain libxml2 dependency for minimal install")
		}

		if _, ok := graph["zlib"]; ok {
			t.Error("GetGraph() should NOT contain zlib dependency for minimal install")
		}

		if _, ok := graph["oniguruma"]; ok {
			t.Error("GetGraph() should NOT contain oniguruma dependency for minimal install")
		}

		if _, ok := graph["curl"]; ok {
			t.Error("GetGraph() should NOT contain curl dependency for minimal install")
		}

		if _, ok := graph["m4"]; ok {
			t.Error("GetGraph() should NOT contain m4 (build tool) for minimal install")
		}
	})

	t.Run("PHP 5.4 minimal (no default dependencies, no flex/bison)", func(t *testing.T) {
		graph, err := repo.GetGraph("php", "5.4.0")
		if err != nil {
			t.Fatalf("GetGraph() error = %v", err)
		}

		if _, ok := graph["flex"]; ok {
			t.Error("GetGraph() should NOT contain flex dependency for PHP 5.4 minimal")
		}

		if _, ok := graph["bison"]; ok {
			t.Error("GetGraph() should NOT contain bison dependency for PHP 5.4 minimal")
		}
	})

	t.Run("OpenSSL 1.1.1 includes perl", func(t *testing.T) {
		graph, err := repo.GetGraph("openssl", "1.1.1w")
		if err != nil {
			t.Fatalf("GetGraph() error = %v", err)
		}

		if _, ok := graph["perl"]; !ok {
			t.Error("GetGraph() should contain perl dependency for OpenSSL 1.x")
		}
	})

	t.Run("OpenSSL 3.x does not include perl", func(t *testing.T) {
		graph, err := repo.GetGraph("openssl", "3.3.2")
		if err != nil {
			t.Fatalf("GetGraph() error = %v", err)
		}

		if _, ok := graph["perl"]; ok {
			t.Error("GetGraph() should NOT contain perl dependency for OpenSSL 3.x")
		}
	})
}

func TestMemoryAssembler_CircularDependency(t *testing.T) {
	repo := NewMemoryAssemblerRepository()

	_, err := repo.GetGraph("nonexistent", "1.0.0")
	if err == nil {
		t.Error("GetGraph() should return error for nonexistent package")
	}
}

func TestMemoryAssembler_GetOrderedDependencies(t *testing.T) {
	repo := NewMemoryAssemblerRepository()

	t.Run("PHP 8.3 minimal (no ordered dependencies)", func(t *testing.T) {
		deps, err := repo.GetOrderedDependencies("php", "8.3.0")
		if err != nil {
			t.Fatalf("GetOrderedDependencies() error = %v", err)
		}

		if len(deps) > 0 {
			t.Errorf("GetOrderedDependencies() got %d deps, want 0 for minimal PHP install", len(deps))
		}
	})

	t.Run("m4 comes before autoconf (base dependency)", func(t *testing.T) {
		deps, err := repo.GetOrderedDependencies("autoconf", "2.72")
		if err != nil {
			t.Fatalf("GetOrderedDependencies() error = %v", err)
		}

		depMap := make(map[string]int)
		for i, dep := range deps {
			depMap[dep.Name] = i
		}

		if m4Idx, ok := depMap["m4"]; ok {
			if autoconfIdx, ok := depMap["autoconf"]; ok {
				if m4Idx >= autoconfIdx {
					t.Errorf("m4 (index %d) should come before autoconf (index %d)", m4Idx, autoconfIdx)
				}
			}
		}
	})

	t.Run("m4 comes before openssl (transitive)", func(t *testing.T) {
		deps, err := repo.GetOrderedDependencies("openssl", "3.3.2")
		if err != nil {
			t.Fatalf("GetOrderedDependencies() error = %v", err)
		}

		depMap := make(map[string]int)
		for i, dep := range deps {
			depMap[dep.Name] = i
		}

		if m4Idx, ok := depMap["m4"]; ok {
			if opensslIdx, ok := depMap["openssl"]; ok {
				if m4Idx >= opensslIdx {
					t.Errorf("m4 (index %d) should come before openssl (index %d)", m4Idx, opensslIdx)
				}
			}
		}
	})

	t.Run("build tools come before libraries (via openssl)", func(t *testing.T) {
		deps, err := repo.GetOrderedDependencies("openssl", "3.3.2")
		if err != nil {
			t.Fatalf("GetOrderedDependencies() error = %v", err)
		}

		buildTools := map[string]bool{
			"m4": true, "autoconf": true, "automake": true, "libtool": true,
			"perl": true, "bison": true, "flex": true, "re2c": true,
		}
		libraries := map[string]bool{
			"openssl": true,
		}

		var lastBuildToolIdx, firstLibraryIdx int = -1, -1
		for i, dep := range deps {
			if buildTools[dep.Name] {
				lastBuildToolIdx = i
			}
			if libraries[dep.Name] && firstLibraryIdx == -1 {
				firstLibraryIdx = i
			}
		}

		if lastBuildToolIdx >= firstLibraryIdx && firstLibraryIdx != -1 {
			t.Errorf("All build tools should come before libraries. Last build tool (index %d), first library (index %d)", lastBuildToolIdx, firstLibraryIdx)
		}
	})
}
