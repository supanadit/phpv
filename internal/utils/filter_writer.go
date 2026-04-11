package utils

import (
	"io"
	"regexp"
	"strings"
)

type ErrorWarningFilter struct {
	writer       io.Writer
	contextLine  string
	pendingError string
	refLines     []string
	patterns     []*regexp.Regexp
}

var contextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\S+:\d+:\d+:\s*`), // file:line:col:
	regexp.MustCompile(`^\S+:\d+:\s*`),     // file:line:
	regexp.MustCompile(`^ld\.lld:\s*`),     // ld.lld:
	regexp.MustCompile(`^ld:\s*`),          // ld:
	regexp.MustCompile(`^gcc:\s*`),         // gcc:
	regexp.MustCompile(`^clang:\s*`),       // clang:
}

var referencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*>>>\s*`), // >>> or >>>
}

var standalonePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^Error[:\s]`),
	regexp.MustCompile(`(?i)^SyntaxError[:\s]`),
	regexp.MustCompile(`(?i)^ModuleNotFoundError[:\s]`),
	regexp.MustCompile(`(?i)^Traceback`),
	regexp.MustCompile(`make\[\d+\]:.*\*\*.*Error`),
	regexp.MustCompile(`(?i)^failed:`),
	regexp.MustCompile(`(?i)^cannot find`),
	regexp.MustCompile(`(?i)^no such file`),
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
		regexp.MustCompile(`(?i)referencing`),
		regexp.MustCompile(`make\[\d+\]:.*\*\*.*Error`),
		regexp.MustCompile(`(?i)\berrored?\b`),
		regexp.MustCompile(`(?i)failed:`),
		regexp.MustCompile(`(?i)Error \d+`),
		regexp.MustCompile(`(?i)Failed to`),
		regexp.MustCompile(`(?i)cannot find`),
		regexp.MustCompile(`(?i)no such file`),
		regexp.MustCompile(`(?i)undefined reference`),
	}
	return &ErrorWarningFilter{
		writer:   w,
		patterns: patterns,
	}
}

func (f *ErrorWarningFilter) isContextLine(line string) bool {
	for _, p := range contextPatterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

func (f *ErrorWarningFilter) isReferenceLine(line string) bool {
	for _, p := range referencePatterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

func (f *ErrorWarningFilter) isStandaloneError(line string) bool {
	for _, p := range standalonePatterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

func (f *ErrorWarningFilter) hasContext(line string) bool {
	return f.isContextLine(line)
}

func (f *ErrorWarningFilter) matches(line string) bool {
	for _, pattern := range f.patterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func (f *ErrorWarningFilter) Write(p []byte) (n int, err error) {
	lines := strings.Split(strings.TrimRight(string(p), "\n"), "\n")
	for _, line := range lines {
		if f.isReferenceLine(line) {
			if f.pendingError != "" || f.contextLine != "" {
				f.refLines = append(f.refLines, line)
			}
			continue
		}

		isStandalone := f.isStandaloneError(line)
		hasCtx := f.hasContext(line)
		matchesPat := f.matches(line)

		if hasCtx && matchesPat {
			f.flushBuffer()
			f.pendingError = line
			f.contextLine = ""
			continue
		}

		if isStandalone {
			f.flushBuffer()
			f.pendingError = line
			f.contextLine = ""
			continue
		}

		if matchesPat && !hasCtx && !isStandalone {
			if f.contextLine != "" && f.pendingError == "" {
				f.pendingError = line
			} else if f.contextLine != "" && f.pendingError != "" {
				f.flushBuffer()
				f.pendingError = line
			}
			continue
		}

		if f.isContextLine(line) && line != "" {
			f.contextLine = line
			continue
		}
	}
	return len(p), nil
}

func (f *ErrorWarningFilter) flushBuffer() {
	if f.pendingError == "" && len(f.refLines) == 0 && f.contextLine == "" {
		return
	}

	var output []string
	if f.contextLine != "" {
		output = append(output, f.contextLine)
	}
	if f.pendingError != "" {
		output = append(output, f.pendingError)
	}
	output = append(output, f.refLines...)

	f.writer.Write([]byte(strings.Join(output, "\n")))
	if len(output) > 0 {
		f.writer.Write([]byte("\n"))
	}

	f.contextLine = ""
	f.pendingError = ""
	f.refLines = f.refLines[:0]
}

func (f *ErrorWarningFilter) Flush() error {
	f.flushBuffer()
	return nil
}

func (f *ErrorWarningFilter) GetOutput() string {
	if f, ok := f.writer.(*strings.Builder); ok {
		return f.String()
	}
	return ""
}
