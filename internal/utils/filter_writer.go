package utils

import (
	"io"
	"regexp"
	"strings"
)

type ErrorWarningFilter struct {
	writer   io.Writer
	patterns []*regexp.Regexp
	buf      []string
}

func NewErrorWarningFilter(w io.Writer) *ErrorWarningFilter {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\berror\b`),
		regexp.MustCompile(`(?i)\bwarning\b`),
		regexp.MustCompile(`(?i)Traceback`),
		regexp.MustCompile(`(?i)SyntaxError`),
		regexp.MustCompile(`(?i)ModuleNotFoundError`),
		regexp.MustCompile(`(?i)undefined symbol:`),
		regexp.MustCompile(`(?i)referenced by`),
		regexp.MustCompile(`make\[\d+\]:.*\*\*.*Error`),
		regexp.MustCompile(`(?i)\berrored?\b`),
		regexp.MustCompile(`(?i)failed:`),
		regexp.MustCompile(`(?i)Error \d+`),
		regexp.MustCompile(`(?i)Failed to`),
		regexp.MustCompile(`(?i)cannot find`),
		regexp.MustCompile(`(?i)no such file`),
		regexp.MustCompile(`(?i)undefined reference`),
		regexp.MustCompile(`(?i)ld:.*error`),
		regexp.MustCompile(`(?i)gcc:.*error`),
		regexp.MustCompile(`(?i)clang:.*error`),
	}
	return &ErrorWarningFilter{
		writer:   w,
		patterns: patterns,
		buf:      []string{},
	}
}

func (f *ErrorWarningFilter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if f.matches(line) {
			f.buf = append(f.buf, line)
		}
	}
	return len(p), nil
}

func (f *ErrorWarningFilter) matches(line string) bool {
	for _, pattern := range f.patterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func (f *ErrorWarningFilter) Flush() error {
	if len(f.buf) == 0 {
		return nil
	}
	_, err := io.WriteString(f.writer, strings.Join(f.buf, "\n"))
	if len(f.buf) > 0 {
		_, err = io.WriteString(f.writer, "\n")
	}
	return err
}

func (f *ErrorWarningFilter) WriteString(s string) error {
	io.WriteString(f.writer, s)
	return nil
}
