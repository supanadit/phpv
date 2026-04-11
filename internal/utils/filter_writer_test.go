package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestErrorWarningFilter_Write(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "linker error - undefined symbol",
			input:    "ld.lld: error: undefined symbol: __ubsan_handle_type_mismatch_v1\n>>> referenced by inflate.c:125",
			expected: []string{"ld.lld: error: undefined symbol: __ubsan_handle_type_mismatch_v1", ">>> referenced by inflate.c:125"},
		},
		{
			name:     "make error",
			input:    "make[2]: *** [Makefile:1043: curl] Error 1",
			expected: []string{"make[2]: *** [Makefile:1043: curl] Error 1"},
		},
		{
			name:     "compiler warning - cast align",
			input:    "encoding.c:505:26: warning: cast from 'const unsigned char *' to 'unsigned short *' increases required alignment from 1 to 2 [-Wcast-align]",
			expected: []string{"encoding.c:505:26: warning: cast from 'const unsigned char *' to 'unsigned short *' increases required alignment from 1 to 2 [-Wcast-align]"},
		},
		{
			name:     "python SyntaxError",
			input:    "  File \"/home/supanadit/.phpv/sources/libxml2/2.9.14/src/./gentest.py\", line 11\n    print \"libxml2 python bindings not available, skipping testapi.c generation\"\n    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\nSyntaxError: Missing parentheses in call to 'print'.",
			expected: []string{"SyntaxError: Missing parentheses in call to 'print'."},
		},
		{
			name:     "python ModuleNotFoundError",
			input:    "ModuleNotFoundError: No module named 'distutils'",
			expected: []string{"ModuleNotFoundError: No module named 'distutils'"},
		},
		{
			name:     "normal output should be filtered",
			input:    "libtool: install: ranlib /home/supanadit/.phpv/versions/8.0.30/dependency/oniguruma/6.9.9/lib/libonig.a\nyes\nchecking for sys/time.h... yes",
			expected: []string{},
		},
		{
			name:     "configure status should be filtered",
			input:    "checking if freeaddrinfo is compilable...   CC       HTMLtree.l",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			filter := NewErrorWarningFilter(&buf)

			filter.Write([]byte(tt.input))
			filter.Flush()

			output := strings.TrimSpace(buf.String())
			if len(tt.expected) == 0 {
				if output != "" {
					t.Errorf("expected empty output, got: %q", output)
				}
				return
			}

			lines := strings.Split(output, "\n")
			if len(lines) != len(tt.expected) {
				t.Errorf("expected %d lines, got %d. Output: %q", len(tt.expected), len(lines), output)
				return
			}

			for i, expected := range tt.expected {
				if !strings.Contains(lines[i], expected) {
					t.Errorf("line %d: expected to contain %q, got %q", i, expected, lines[i])
				}
			}
		})
	}
}

func TestErrorWarningFilter_matches(t *testing.T) {
	filter := NewErrorWarningFilter(&bytes.Buffer{})

	tests := []struct {
		line     string
		expected bool
	}{
		{"ld.lld: error: undefined symbol: __ubsan_handle", true},
		{"make[2]: *** [Makefile:1043: curl] Error 1", true},
		{"encoding.c:505:26: warning:", true},
		{"SyntaxError: Missing parentheses", true},
		{"ModuleNotFoundError: No module named", true},
		{"checking for sys/time.h... yes", false},
		{"libtool: install: ranlib", false},
	}

	for _, tt := range tests {
		result := filter.matches(tt.line)
		if result != tt.expected {
			t.Errorf("matches(%q) = %v, expected %v", tt.line, result, tt.expected)
		}
	}
}
