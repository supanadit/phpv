package repository

import (
	"reflect"
	"testing"
)

func TestBuildMinorRanges(t *testing.T) {
	got := BuildMinorRanges(4, []MinorRange{
		{Minor: 0, PatchEnd: 2},
		{Minor: 1, PatchEnd: 1},
	})
	want := []VersionRange{
		{From: "4.0.0", To: "4.0.2"},
		{From: "4.1.0", To: "4.1.1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildMinorRanges() = %v, want %v", got, want)
	}
}

func TestBuildMinorRanges_PatchStart(t *testing.T) {
	got := BuildMinorRanges(8, []MinorRange{
		{Minor: 0, PatchStart: 2, PatchEnd: 4},
	})
	want := []VersionRange{
		{From: "8.0.2", To: "8.0.4"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildMinorRanges() = %v, want %v", got, want)
	}
}

func TestBuildRanges(t *testing.T) {
	got := BuildRanges(
		BuildMinorRanges(8, []MinorRange{{Minor: 0, PatchEnd: 1}}),
		BuildMinorRanges(7, []MinorRange{{Minor: 4, PatchEnd: 1}}),
	)
	want := []VersionRange{
		{From: "8.0.0", To: "8.0.1"},
		{From: "7.4.0", To: "7.4.1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildRanges() = %v, want %v", got, want)
	}
}

func TestGenerateVersions(t *testing.T) {
	ranges := []VersionRange{
		{From: "1.0.0", To: "1.0.2"},
		{From: "1.1.0", To: "1.1.1"},
	}
	got := GenerateVersions(ranges, nil)
	want := []string{"1.0.0", "1.0.1", "1.0.2", "1.1.0", "1.1.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GenerateVersions() = %v, want %v", got, want)
	}
}

func TestGenerateVersions_Skip(t *testing.T) {
	ranges := []VersionRange{{From: "1.0.0", To: "1.0.3"}}
	got := GenerateVersions(ranges, []string{"1.0.1"})
	want := []string{"1.0.0", "1.0.2", "1.0.3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GenerateVersions(skip) = %v, want %v", got, want)
	}
}

func TestGenerateVersions_Gap(t *testing.T) {
	ranges := []VersionRange{
		{From: "1.0.0", To: "1.0.1"},
		{From: "1.0.3", To: "1.0.3"},
	}
	got := GenerateVersions(ranges, nil)
	want := []string{"1.0.0", "1.0.1", "1.0.3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GenerateVersions(gap) = %v, want %v", got, want)
	}
}

func TestGenerateVersions_Wildcard(t *testing.T) {
	ranges := []VersionRange{
		{From: "1.x.x", To: "1.0.1"},
	}
	got := GenerateVersions(ranges, nil)
	want := []string{"1.0.0", "1.0.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GenerateVersions(wildcard) = %v, want %v", got, want)
	}
}

func TestRenderTemplate(t *testing.T) {
	got := RenderTemplate("https://php.net/distributions/php-{version}.tar.gz", "8.2.1")
	want := "https://php.net/distributions/php-8.2.1.tar.gz"
	if got != want {
		t.Fatalf("RenderTemplate() = %q, want %q", got, want)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.9.0", "1.10.0", -1},
		{"5.9.0", "5.20.0", -1},
		{"5.20.0", "5.9.0", 1},
		{"2.0.0", "1.99.99", 1},
	}
	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  Semver
	}{
		{"8.2.1", Semver{Major: 8, Minor: 2, Patch: 1}},
		{"8.x.x", Semver{Major: 8, Minor: -1, Patch: -1}},
		{"x.X.x", Semver{Major: -1, Minor: -1, Patch: -1}},
		{"not-a-version", Semver{}},
	}
	for _, tt := range tests {
		got := ParseVersion(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
