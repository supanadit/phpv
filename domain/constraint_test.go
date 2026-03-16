package domain

import (
	"testing"
)

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		input        string
		wantType     ConstraintType
		wantMin      string
		wantMax      string
		wantRec      string
		wantOptional bool
	}{
		{"", ConstraintEmpty, "", "", "", true},
		{"2.12.7", ConstraintExact, "2.12.7", "2.12.7", "2.12.7", false},
		{"2.12.7|2.12.7", ConstraintExact, "2.12.7", "2.12.7", "2.12.7", false},
		{"2.12.7|~2.12.0", ConstraintTilde, "2.12.0", "2.13.0", "2.12.7", false},
		{"~2.12.0", ConstraintTilde, "2.12.0", "2.13.0", "~2.12.0", false},
		{">=2.12.0,<2.13.0", ConstraintRange, "2.12.0", "2.13.0", ">=2.12.0,<2.13.0", false},
		{"2.12.7|>=2.12.0,<2.13.0", ConstraintRange, "2.12.0", "2.13.0", "2.12.7", false},
		{">=8.0.0", ConstraintRange, "8.0.0", "", ">=8.0.0", false},
		{"8.10.1|>=8.0.0", ConstraintRange, "8.0.0", "", "8.10.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c, err := ParseConstraint(tt.input)
			if err != nil {
				t.Fatalf("ParseConstraint(%q) error = %v", tt.input, err)
			}
			if c.Type != tt.wantType {
				t.Errorf("ParseConstraint(%q).Type = %v, want %v", tt.input, c.Type, tt.wantType)
			}
			if c.Min != tt.wantMin {
				t.Errorf("ParseConstraint(%q).Min = %v, want %v", tt.input, c.Min, tt.wantMin)
			}
			if c.Max != tt.wantMax {
				t.Errorf("ParseConstraint(%q).Max = %v, want %v", tt.input, c.Max, tt.wantMax)
			}
			if c.Recommended != tt.wantRec {
				t.Errorf("ParseConstraint(%q).Recommended = %v, want %v", tt.input, c.Recommended, tt.wantRec)
			}
			if c.Optional != tt.wantOptional {
				t.Errorf("ParseConstraint(%q).Optional = %v, want %v", tt.input, c.Optional, tt.wantOptional)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{"2.12.7", "2.12.7", 0},
		{"2.12.10", "2.12.7", 1},
		{"2.12.7", "2.12.10", -1},
		{"1.4.19", "1.4.19", 0},
		{"1.4-p6", "1.4.0", 1},
		{"0.9.8zh", "0.9.8", 1},
		{"2.5.39", "2.5.4", 1},
		{"3.0.14", "3.3.2", -1},
		{"1.1.1w", "1.1.1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := CompareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDependencyConstraintMatches(t *testing.T) {
	tests := []struct {
		constraint string
		version    string
		want       bool
	}{
		{"2.12.7", "2.12.7", true},
		{"2.12.7", "2.12.8", false},
		{"~2.12.0", "2.12.7", true},
		{"~2.12.0", "2.13.0", false},
		{"~2.12.0", "2.11.9", false},
		{">=2.12.0,<2.13.0", "2.12.7", true},
		{">=2.12.0,<2.13.0", "2.13.0", false},
		{">=2.12.0,<2.13.0", "2.11.9", false},
		{">=8.0.0", "8.10.1", true},
		{">=8.0.0", "7.88.1", false},
		{"", "any", true},
	}

	for _, tt := range tests {
		t.Run(tt.constraint+"_"+tt.version, func(t *testing.T) {
			c, err := ParseConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("ParseConstraint(%q) error = %v", tt.constraint, err)
			}
			got := c.Matches(tt.version)
			if got != tt.want {
				t.Errorf("Constraint(%q).Matches(%q) = %v, want %v", tt.constraint, tt.version, got, tt.want)
			}
		})
	}
}

func TestDependencyConstraintWithExclusions(t *testing.T) {
	constraint, err := ParseConstraint("2.12.7|>=2.12.0,!=2.12.5")
	if err != nil {
		t.Fatalf("ParseConstraint error = %v", err)
	}

	tests := []struct {
		version string
		want    bool
	}{
		{"2.12.7", true},
		{"2.12.6", true},
		{"2.12.5", false},
		{"2.13.0", true}, // >= 2.12.0 and not excluded
		{"2.12.0", true},
		{"2.11.9", false}, // below min
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := constraint.Matches(tt.version)
			if got != tt.want {
				t.Errorf("Constraint.Matches(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestDependencyVersionSpec(t *testing.T) {
	spec := DependencyVersionSpec{
		ConstraintStr: "2.12.7|~2.12.0",
		Optional:      false,
	}
	c, err := ParseConstraint(spec.ConstraintStr)
	if err != nil {
		t.Fatalf("ParseConstraint error = %v", err)
	}
	spec.Constraint = c

	if spec.GetRecommended() != "2.12.7" {
		t.Errorf("GetRecommended() = %v, want 2.12.7", spec.GetRecommended())
	}
	if spec.GetMin() != "2.12.0" {
		t.Errorf("GetMin() = %v, want 2.12.0", spec.GetMin())
	}
	if spec.GetMax() != "2.13.0" {
		t.Errorf("GetMax() = %v, want 2.13.0", spec.GetMax())
	}
	if spec.IsOptional() {
		t.Error("IsOptional() = true, want false")
	}

	emptySpec := DependencyVersionSpec{
		ConstraintStr: "",
		Optional:      true,
	}
	ec, _ := ParseConstraint(emptySpec.ConstraintStr)
	emptySpec.Constraint = ec

	if !emptySpec.IsOptional() {
		t.Error("IsOptional() = false, want true for empty spec")
	}
}
