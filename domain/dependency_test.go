package domain

import "testing"

func TestSplitVersionSpec(t *testing.T) {
	tests := []struct {
		spec          string
		version, cons string
	}{
		{"1.1.1w|>=1.1.0", "1.1.1w", ">=1.1.0"},
		{"5.38.2|>=5.32.0", "5.38.2", ">=5.32.0"},
		{"2.4.6", "2.4.6", ""},
		{"", "", ""},
		{"8.0|", "8.0", ""},
		{"1.4.19|>=1.4.19", "1.4.19", ">=1.4.19"},
	}
	for _, tt := range tests {
		v, c := SplitVersionSpec(tt.spec)
		if v != tt.version || c != tt.cons {
			t.Errorf("SplitVersionSpec(%q) = (%q, %q), want (%q, %q)", tt.spec, v, c, tt.version, tt.cons)
		}
	}
}
