package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseComposerJSON(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty config",
			content:  `{}`,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "no config",
			content:  `{"name": "test"}`,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "simple version",
			content:  `{"config":{"platform":{"php":"8.1"}}}`,
			expected: "8.1",
			wantErr:  false,
		},
		{
			name:     "caret constraint",
			content:  `{"config":{"platform":{"php":"^8.1"}}}`,
			expected: "8.1",
			wantErr:  false,
		},
		{
			name:     "greater than or equal",
			content:  `{"config":{"platform":{"php":">=8.0"}}}`,
			expected: "8.0",
			wantErr:  false,
		},
		{
			name:     "full version with patch",
			content:  `{"config":{"platform":{"php":"8.1.5"}}}`,
			expected: "8.1.5",
			wantErr:  false,
		},
		{
			name:     "range constraint",
			content:  `{"config":{"platform":{"php":">=8.1 <9.0"}}}`,
			expected: "8.1",
			wantErr:  false,
		},
		{
			name: "complex composer.json",
			content: `{
				"name": "test/package",
				"require": {
					"php": "^8.1"
				},
				"config": {
					"platform": {
						"php": "8.4"
					}
				}
			}`,
			expected: "8.4",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composerPath := filepath.Join(tmpDir, "composer.json")
			if err := os.WriteFile(composerPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got, err := ParseComposerJSON(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseComposerJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseComposerJSON() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestParseComposerJSONNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ParseComposerJSON(tmpDir)
	if err == nil {
		t.Error("expected error when composer.json not found")
	}
}
