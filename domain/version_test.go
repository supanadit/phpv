package domain

import (
	"testing"
)

func TestPHPVersion_Compare(t *testing.T) {
	tests := []struct {
		name     string
		v1       PHPVersion
		v2       PHPVersion
		expected int
	}{
		{
			name:     "same version",
			v1:       PHPVersion{Major: 8, Minor: 1, Patch: 0, ReleaseType: "stable"},
			v2:       PHPVersion{Major: 8, Minor: 1, Patch: 0, ReleaseType: "stable"},
			expected: 0,
		},
		{
			name:     "v1 greater major",
			v1:       PHPVersion{Major: 8, Minor: 1, Patch: 0},
			v2:       PHPVersion{Major: 7, Minor: 4, Patch: 0},
			expected: 1,
		},
		{
			name:     "v2 greater major",
			v1:       PHPVersion{Major: 7, Minor: 4, Patch: 0},
			v2:       PHPVersion{Major: 8, Minor: 1, Patch: 0},
			expected: -1,
		},
		{
			name:     "stable > rc",
			v1:       PHPVersion{Major: 8, Minor: 1, Patch: 0, ReleaseType: "stable"},
			v2:       PHPVersion{Major: 8, Minor: 1, Patch: 0, ReleaseType: "rc", ReleaseNumber: 1},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.Compare(tt.v2)
			if result != tt.expected {
				t.Errorf("Compare() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestPHPVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  PHPVersion
		expected string
	}{
		{
			name:     "stable version",
			version:  PHPVersion{Major: 8, Minor: 1, Patch: 0, ReleaseType: "stable"},
			expected: "8.1.0",
		},
		{
			name:     "rc version",
			version:  PHPVersion{Major: 8, Minor: 2, Patch: 0, ReleaseType: "rc", ReleaseNumber: 1},
			expected: "8.2.0-rc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			if result != tt.expected {
				t.Errorf("String() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestPHPVersion_IsStable(t *testing.T) {
	stable := PHPVersion{ReleaseType: "stable"}
	rc := PHPVersion{ReleaseType: "rc"}

	if !stable.IsStable() {
		t.Error("Expected stable version to be stable")
	}
	if rc.IsStable() {
		t.Error("Expected rc version not to be stable")
	}
}

func TestInstallation_Activate(t *testing.T) {
	inst := Installation{IsActive: false}
	inst.Activate()
	if !inst.IsActive {
		t.Error("Expected installation to be active after Activate()")
	}
}

func TestInstallation_Deactivate(t *testing.T) {
	inst := Installation{IsActive: true}
	inst.Deactivate()
	if inst.IsActive {
		t.Error("Expected installation to be inactive after Deactivate()")
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    PHPVersion
		expectError bool
	}{
		{
			name:  "stable version",
			input: "8.1.0",
			expected: PHPVersion{
				Version:     "8.1.0",
				Major:       8,
				Minor:       1,
				Patch:       0,
				ReleaseType: "stable",
			},
			expectError: false,
		},
		{
			name:  "rc version",
			input: "8.2.0-rc1",
			expected: PHPVersion{
				Version:       "8.2.0-rc1",
				Major:         8,
				Minor:         2,
				Patch:         0,
				ReleaseType:   "rc",
				ReleaseNumber: 1,
			},
			expectError: false,
		},
		{
			name:  "alpha version",
			input: "8.3.0-alpha2",
			expected: PHPVersion{
				Version:       "8.3.0-alpha2",
				Major:         8,
				Minor:         3,
				Patch:         0,
				ReleaseType:   "alpha",
				ReleaseNumber: 2,
			},
			expectError: false,
		},
		{
			name:  "beta version",
			input: "8.3.0-beta1",
			expected: PHPVersion{
				Version:       "8.3.0-beta1",
				Major:         8,
				Minor:         3,
				Patch:         0,
				ReleaseType:   "beta",
				ReleaseNumber: 1,
			},
			expectError: false,
		},
		{
			name:        "invalid format",
			input:       "invalid",
			expected:    PHPVersion{},
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    PHPVersion{},
			expectError: true,
		},
		{
			name:        "negative numbers",
			input:       "-1.0.0",
			expected:    PHPVersion{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseVersion(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("ParseVersion() = %+v, expected %+v", result, tt.expected)
				}
			}
		})
	}
}

func TestPHPVersion_Validate(t *testing.T) {
	tests := []struct {
		name        string
		version     PHPVersion
		expectError bool
	}{
		{
			name: "valid stable version",
			version: PHPVersion{
				Major:       8,
				Minor:       1,
				Patch:       0,
				ReleaseType: "stable",
			},
			expectError: false,
		},
		{
			name: "valid rc version",
			version: PHPVersion{
				Major:         8,
				Minor:         2,
				Patch:         0,
				ReleaseType:   "rc",
				ReleaseNumber: 1,
			},
			expectError: false,
		},
		{
			name: "negative major version",
			version: PHPVersion{
				Major:       -1,
				Minor:       1,
				Patch:       0,
				ReleaseType: "stable",
			},
			expectError: true,
		},
		{
			name: "invalid release type",
			version: PHPVersion{
				Major:       8,
				Minor:       1,
				Patch:       0,
				ReleaseType: "invalid",
			},
			expectError: true,
		},
		{
			name: "stable with release number",
			version: PHPVersion{
				Major:         8,
				Minor:         1,
				Patch:         0,
				ReleaseType:   "stable",
				ReleaseNumber: 1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.version.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInstallation_Validate(t *testing.T) {
	tests := []struct {
		name         string
		installation Installation
		expectError  bool
	}{
		{
			name: "valid installation",
			installation: Installation{
				Version: PHPVersion{
					Major:       8,
					Minor:       1,
					Patch:       0,
					ReleaseType: "stable",
				},
				Path: "/usr/local/php/8.1.0",
			},
			expectError: false,
		},
		{
			name: "invalid version",
			installation: Installation{
				Version: PHPVersion{
					Major:       -1,
					Minor:       1,
					Patch:       0,
					ReleaseType: "stable",
				},
				Path: "/usr/local/php/8.1.0",
			},
			expectError: true,
		},
		{
			name: "empty path",
			installation: Installation{
				Version: PHPVersion{
					Major:       8,
					Minor:       1,
					Patch:       0,
					ReleaseType: "stable",
				},
				Path: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.installation.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
