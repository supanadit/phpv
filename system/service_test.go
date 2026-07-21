package system

import "testing"

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
