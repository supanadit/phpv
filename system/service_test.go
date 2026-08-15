package system

import (
	"strings"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"3.6.3-1", "3.6.3"},
		{"8.21.0-1", "8.21.0"},
		{"1:1.3.2-3", "1.3.2"}, // Arch epoch prefix
		{"78.3-1", "78.3"},
		{"6.9.10-1", "6.9.10"},
		{"3.53.3-1", "3.53.3"},
		{"1.0.0-beta", "1.0.0"}, // pre-release stripped for system packages
		{"2.15.3-1", "2.15.3"},
		{"3.0.2-0ubuntu1.18", "3.0.2"}, // Ubuntu suffix
		{"3.0.7-1.el9", "3.0.7"},       // RHEL suffix
		{"3.1.4-r0", "3.1.4"},          // Alpine suffix
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeVersion(tt.input)
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeVersion_ExtractsArchEpoch(t *testing.T) {
	// Arch packages sometimes prefix versions with an epoch, e.g. zlib "1:1.3.2-3"
	got := normalizeVersion("1:1.3.2-3")
	want := "1.3.2"
	if got != want {
		t.Errorf("normalizeVersion(1:1.3.2-3) = %q, want %q", got, want)
	}
}

func TestCheckBuildTools_SplitsAvailableAndMissing(t *testing.T) {
	toolNames := []string{"gcc", "g++", "make", "cmake", "autoconf", "automake", "m4", "perl", "bison", "re2c", "flex", "pkg-config", "xz"}

	result, err := (&Service{}).CheckBuildTools(toolNames)
	if err != nil {
		t.Fatalf("CheckBuildTools returned error: %v", err)
	}

	seen := make(map[string]bool)
	for _, p := range result.Available {
		if p.Installed {
			seen[p.Name] = true
		}
	}
	for _, p := range result.Missing {
		if p.Installed {
			t.Errorf("missing package %s has Installed=true", p.Name)
		}
		if seen[p.Name] {
			t.Errorf("package %s appears in both Available and Missing", p.Name)
		}
		seen[p.Name] = true
	}

	var missing []string
	for _, name := range toolNames {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("tools not classified as either available or missing: %s", strings.Join(missing, ", "))
	}
}

func TestPackagesForDistro_LibPQMapping(t *testing.T) {
	cases := []struct {
		distro string
		want   string
	}{
		{"arch", "postgresql-libs"},
		{"ubuntu", "libpq-dev"},
		{"debian", "libpq-dev"},
		{"alpine", "libpq-dev"},
		{"fedora", "postgresql-devel"},
		{"rhel", "postgresql-devel"},
		{"centos", "postgresql-devel"},
	}
	for _, c := range cases {
		pkgs := packagesForDistro(c.distro)
		got, ok := pkgs["libpq"]
		if !ok {
			t.Errorf("packagesForDistro(%q) has no libpq mapping", c.distro)
			continue
		}
		if got != c.want {
			t.Errorf("packagesForDistro(%q)[libpq] = %q, want %q", c.distro, got, c.want)
		}
	}
}
