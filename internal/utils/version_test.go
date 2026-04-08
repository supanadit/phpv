package utils

import (
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestParseVersion_Full(t *testing.T) {
	tests := []struct {
		input    string
		expected *domain.Version
	}{
		{"8.4.0", &domain.Version{Major: 8, Minor: 4, Patch: 0, Raw: "8.4.0"}},
		{"8.4.1", &domain.Version{Major: 8, Minor: 4, Patch: 1, Raw: "8.4.1"}},
		{"7.4.33", &domain.Version{Major: 7, Minor: 4, Patch: 33, Raw: "7.4.33"}},
		{"5.6.40", &domain.Version{Major: 5, Minor: 6, Patch: 40, Raw: "5.6.40"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseVersion(tc.input)
			if result.Major != tc.expected.Major {
				t.Errorf("Expected Major %d, got %d", tc.expected.Major, result.Major)
			}
			if result.Minor != tc.expected.Minor {
				t.Errorf("Expected Minor %d, got %d", tc.expected.Minor, result.Minor)
			}
			if result.Patch != tc.expected.Patch {
				t.Errorf("Expected Patch %d, got %d", tc.expected.Patch, result.Patch)
			}
		})
	}
}

func TestParseVersion_WithSuffix(t *testing.T) {
	result := ParseVersion("8.4.0alpha")
	if result.Suffix != "alpha" {
		t.Errorf("Expected suffix 'alpha', got '%s'", result.Suffix)
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	result := ParseVersion("invalid")
	if result.Major != 0 || result.Minor != 0 || result.Patch != 0 {
		t.Error("Invalid version should return zeros")
	}
	if result.Raw != "invalid" {
		t.Errorf("Expected Raw 'invalid', got '%s'", result.Raw)
	}
}

func TestCompareVersions_Greater(t *testing.T) {
	a := &domain.Version{Major: 8, Minor: 4, Patch: 1}
	b := &domain.Version{Major: 8, Minor: 4, Patch: 0}

	result := CompareVersions(a, b)
	if result <= 0 {
		t.Error("8.4.1 should be greater than 8.4.0")
	}
}

func TestCompareVersions_Less(t *testing.T) {
	a := &domain.Version{Major: 8, Minor: 3, Patch: 33}
	b := &domain.Version{Major: 8, Minor: 4, Patch: 0}

	result := CompareVersions(a, b)
	if result >= 0 {
		t.Error("8.3.33 should be less than 8.4.0")
	}
}

func TestCompareVersions_Equal(t *testing.T) {
	a := &domain.Version{Major: 8, Minor: 4, Patch: 0}
	b := &domain.Version{Major: 8, Minor: 4, Patch: 0}

	result := CompareVersions(a, b)
	if result != 0 {
		t.Error("8.4.0 should be equal to 8.4.0")
	}
}

func TestCompareVersions_MajorDiff(t *testing.T) {
	a := &domain.Version{Major: 9, Minor: 0, Patch: 0}
	b := &domain.Version{Major: 8, Minor: 4, Patch: 0}

	result := CompareVersions(a, b)
	if result <= 0 {
		t.Error("9.0.0 should be greater than 8.4.0")
	}
}

func TestSortVersions_Sorted(t *testing.T) {
	versions := []string{"8.4.0", "7.4.33", "8.3.0", "7.3.0"}

	SortVersions(versions)

	if versions[0] != "8.4.0" {
		t.Errorf("Expected first version 8.4.0, got %s", versions[0])
	}
}

func TestSortVersions_Empty(t *testing.T) {
	versions := []string{}
	SortVersions(versions)

	if len(versions) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(versions))
	}
}

func TestSortVersions_Single(t *testing.T) {
	versions := []string{"8.4.0"}
	SortVersions(versions)

	if len(versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(versions))
	}
	if versions[0] != "8.4.0" {
		t.Errorf("Expected version 8.4.0, got %s", versions[0])
	}
}

func TestFilterVersionsByConstraint_MajorMinor(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.3.0", "7.4.33"}

	result := FilterVersionsByConstraint(versions, "8.4")

	if len(result) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(result))
	}
}

func TestFilterVersionsByConstraint_Major(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0", "7.4.33"}

	result := FilterVersionsByConstraint(versions, "8.0")

	if len(result) != 0 {
		t.Errorf("Expected 0 versions (minor must match), got %d", len(result))
	}
}

func TestFilterVersionsByConstraint_Exact(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.4.0"}

	result := FilterVersionsByConstraint(versions, "8.4.0")

	if len(result) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(result))
	}
}

func TestFilterVersionsByConstraint_NoMatch(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}

	result := FilterVersionsByConstraint(versions, "9.0")

	if len(result) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(result))
	}
}

func TestFilterVersionsByConstraint_Empty(t *testing.T) {
	versions := []string{}

	result := FilterVersionsByConstraint(versions, "8.4")

	if len(result) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(result))
	}
}

func TestResolveVersionConstraint_Found(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.3.0"}

	result, err := ResolveVersionConstraint(versions, "8.4")
	if err != nil {
		t.Errorf("ResolveVersionConstraint failed: %v", err)
	}
	if result != "8.4.1" {
		t.Errorf("Expected 8.4.1, got %s", result)
	}
}

func TestResolveVersionConstraint_NotFound(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}

	_, err := ResolveVersionConstraint(versions, "9.0")
	if err == nil {
		t.Error("Should have returned error for non-existent version")
	}
}

func TestResolveInstalledVersion_Found(t *testing.T) {
	versions := []string{"8.4.0", "8.4.1", "8.3.0"}

	result, err := ResolveInstalledVersion(versions, "8.4")
	if err != nil {
		t.Errorf("ResolveInstalledVersion failed: %v", err)
	}
	if result != "8.4.1" {
		t.Errorf("Expected 8.4.1, got %s", result)
	}
}

func TestResolveInstalledVersion_NotFound(t *testing.T) {
	versions := []string{"8.4.0", "8.3.0"}

	_, err := ResolveInstalledVersion(versions, "9.0")
	if err == nil {
		t.Error("Should have returned error for non-installed version")
	}
}

func TestMatchVersionRange_Equal(t *testing.T) {
	if !MatchVersionRange("=8.4.0", "8.4.0") {
		t.Error("8.4.0 should match =8.4.0")
	}
}

func TestMatchVersionRange_GreaterThanOrEqual(t *testing.T) {
	if !MatchVersionRange(">=8.4.0", "8.4.1") {
		t.Error("8.4.1 should match >=8.4.0")
	}
}

func TestMatchVersionRange_LessThan(t *testing.T) {
	if !MatchVersionRange("<8.5.0", "8.4.0") {
		t.Error("8.4.0 should match <8.5.0")
	}
}

func TestMatchVersionRange_Tilde(t *testing.T) {
	if !MatchVersionRange("~8.4.0", "8.4.5") {
		t.Error("8.4.5 should match ~8.4.0")
	}
	if MatchVersionRange("~8.4.0", "8.5.0") {
		t.Error("8.5.0 should not match ~8.4.0")
	}
}

func TestMatchVersionRange_Caret(t *testing.T) {
	if !MatchVersionRange("^8.4.0", "8.5.0") {
		t.Error("8.5.0 should match ^8.4.0")
	}
	if MatchVersionRange("^8.4.0", "9.0.0") {
		t.Error("9.0.0 should not match ^8.4.0")
	}
}

func TestSplitConstraint(t *testing.T) {
	result := splitConstraint("8.4")
	if len(result) != 2 {
		t.Errorf("Expected 2 parts, got %d", len(result))
	}

	result = splitConstraint("8")
	if len(result) != 1 {
		t.Errorf("Expected 1 part, got %d", len(result))
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"8", 8},
		{"84", 84},
		{"123", 123},
		{"0", 0},
	}

	for _, tc := range tests {
		result := parseInt(tc.input)
		if result != tc.expected {
			t.Errorf("parseInt(%s): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}
