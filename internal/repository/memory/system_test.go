package memory

import "testing"

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
		pkgs := NewSystemCatalog().PackagesForDistro(c.distro)
		got, ok := pkgs["libpq"]
		if !ok {
			t.Errorf("NewSystemCatalog().PackagesForDistro(%q) has no libpq mapping", c.distro)
			continue
		}
		if got != c.want {
			t.Errorf("NewSystemCatalog().PackagesForDistro(%q)[libpq] = %q, want %q", c.distro, got, c.want)
		}
	}
}

func init() { _ = NewSystemCatalog() }
