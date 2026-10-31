package patches

import "testing"

func TestDirRangeExpr(t *testing.T) {
	tests := []struct {
		dir  string
		want string
	}{
		{"any", ""},
		{"8.0.0-8.1.0", ">=8.0.0,<8.1.0"},
		{"6.0.0+", ">=6.0.0"},
		{"6.9.7", "=6.9.7"},
		{"5.0.0-7.0.0", ">=5.0.0,<7.0.0"},
		{"6.9.9-6.9.10", ">=6.9.9,<6.9.10"},
	}
	for _, tt := range tests {
		if got := DirRangeExpr(tt.dir); got != tt.want {
			t.Errorf("DirRangeExpr(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}

func TestDirRangeExprMatchesVersions(t *testing.T) {
	// End-to-end through the real matcher, pinning the semantics.
	tests := []struct {
		dir     string
		version string
		want    bool
	}{
		{"8.0.0-8.1.0", "8.0.30", true},
		{"8.0.0-8.1.0", "8.1.0", false},
		{"8.0.0-8.1.0", "7.4.33", false},
		{"6.0.0+", "6.9.9", true},
		{"6.0.0+", "6.0.0", true},
		{"6.0.0+", "5.9.6", false},
		{"6.9.7", "6.9.7", true},
		{"6.9.7", "6.9.8", false},
	}
	for _, tt := range tests {
		if got := entryMatches(Entry{RangeDir: tt.dir}, tt.version); got != tt.want {
			t.Errorf("dir %q vs %s = %v, want %v", tt.dir, tt.version, got, tt.want)
		}
	}
}
