package assembler

import "testing"

func TestResolveVersionConstraint_Exact(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.4.2", "8.3.0", "7.4.0"}
	got, err := resolveVersionConstraint(versions, "8.4.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "8.4.1" {
		t.Fatalf("got %q, want 8.4.1", got)
	}
}

func TestResolveVersionConstraint_MajorMinor(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.4.2", "8.3.0", "7.4.0"}
	got, err := resolveVersionConstraint(versions, "8.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "8.4.2" {
		t.Fatalf("got %q, want 8.4.2 (latest patch)", got)
	}
}

func TestResolveVersionConstraint_MajorOnly(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.3.0", "8.2.0", "7.4.0"}
	got, err := resolveVersionConstraint(versions, "8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "8.4.1" {
		t.Fatalf("got %q, want 8.4.1 (latest 8.x)", got)
	}
}

func TestResolveVersionConstraint_NoMatch(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}
	_, err := resolveVersionConstraint(versions, "7")
	if err == nil {
		t.Fatal("expected error for non-matching constraint")
	}
}

func TestResolveVersionConstraint_EmptyVersions(t *testing.T) {
	_, err := resolveVersionConstraint(nil, "8")
	if err == nil {
		t.Fatal("expected error for empty versions")
	}
}

func TestLatestMatching(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.4.2", "8.3.0", "7.4.0"}
	got := latestMatching(versions, "8.4.")
	if got != "8.4.2" {
		t.Fatalf("got %q, want 8.4.2", got)
	}
}

func TestLatestMatching_NoMatch(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}
	got := latestMatching(versions, "7.")
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"8.4.0", "8.3.0", 1},
		{"8.3.0", "8.4.0", -1},
		{"8.4.0", "8.4.0", 0},
		{"8.4.2", "8.4.1", 1},
		{"8.4.0", "7.4.0", 1},
		{"7.4.0", "8.4.0", -1},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
