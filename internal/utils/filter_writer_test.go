package utils

import (
	"strings"
	"testing"
)

func TestErrorWarningFilter_Write(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "linker error - undefined symbol",
			input:    "ld.lld: error: undefined symbol: __ubsan_handle_type_mismatch_v1\n>>> referenced by inflate.c:125",
			expected: "ld.lld: error: undefined symbol: __ubsan_handle_type_mismatch_v1\n>>> referenced by inflate.c:125\n",
		},
		{
			name:     "make error",
			input:    "make[2]: *** [Makefile:1043: curl] Error 1",
			expected: "make[2]: *** [Makefile:1043: curl] Error 1\n",
		},
		{
			name:     "compiler warning with context",
			input:    "encoding.c:505:26: warning: cast from 'const unsigned char *' to 'unsigned short *'",
			expected: "encoding.c:505:26: warning: cast from 'const unsigned char *' to 'unsigned short *'\n",
		},
		{
			name:     "python SyntaxError",
			input:    "SyntaxError: Missing parentheses in call to 'print'.",
			expected: "SyntaxError: Missing parentheses in call to 'print'.\n",
		},
		{
			name:     "python ModuleNotFoundError",
			input:    "ModuleNotFoundError: No module named 'distutils'",
			expected: "ModuleNotFoundError: No module named 'distutils'\n",
		},
		{
			name:     "normal output should be filtered",
			input:    "libtool: install: ranlib /home/supanadit/.phpv/versions/8.0.30/dependency/oniguruma/6.9.9/lib/libonig.a\nyes\nchecking for sys/time.h... yes",
			expected: "",
		},
		{
			name:     "standalone warning without context should be filtered",
			input:    "warning:\nwarning:\nwarning:\n",
			expected: "",
		},
		{
			name:     "error without context should be output",
			input:    "Error: some error occurred",
			expected: "Error: some error occurred\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewErrorWarningFilter(&strings.Builder{})

			filter.Write([]byte(tt.input))
			filter.Flush()

			output := filter.GetOutput()
			if output != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, output)
			}
		})
	}
}
