package build

import (
	"context"
	"testing"

	"github.com/supanadit/phpv/domain"
)

func TestFindMatchingVersion(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	versions := []domain.Version{
		{Major: 8, Minor: 4, Patch: 14},
		{Major: 8, Minor: 4, Patch: 13},
		{Major: 8, Minor: 3, Patch: 15},
		{Major: 8, Minor: 3, Patch: 14},
		{Major: 7, Minor: 4, Patch: 33},
	}

	tests := []struct {
		name        string
		major       int
		minor       *int
		patch       *int
		expected    domain.Version
		expectError bool
	}{
		{
			name:     "exact version match",
			major:    8,
			minor:    intPtr(4),
			patch:    intPtr(14),
			expected: domain.Version{Major: 8, Minor: 4, Patch: 14},
		},
		{
			name:     "major.minor match returns latest patch",
			major:    8,
			minor:    intPtr(3),
			expected: domain.Version{Major: 8, Minor: 3, Patch: 15},
		},
		{
			name:     "major only match returns latest",
			major:    8,
			expected: domain.Version{Major: 8, Minor: 4, Patch: 14},
		},
		{
			name:        "no match found",
			major:       9,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.FindMatchingVersion(ctx, versions, tt.major, tt.minor, tt.patch)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
