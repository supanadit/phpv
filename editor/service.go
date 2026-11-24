// Package editor edits php.ini load directives for PHP extensions.
package editor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Service writes php.ini load directives and verifies PHP loads them.
type Service struct{}

// NewService creates an editor service.
func NewService() *Service {
	return &Service{}
}

// Enable writes the load directive for extName into iniPath and verifies the
// module actually loads via php -m. If the initial form (chosen from zendHint)
// does not load, it flips to the other form once. If neither loads, both
// entries are removed and an error carrying PHP's diagnostics is returned.
// Returns the entry that verified.
func (s *Service) Enable(phpBin, iniPath, extName string, zendHint bool) (string, error) {
	zend := zendHint
	entry := iniEntryFor(extName, zend)
	if err := s.Write(iniPath, entry); err != nil {
		return "", err
	}
	if ok, phpErr := moduleLoads(phpBin, extName); !ok {
		s.Remove(iniPath, entry)
		alt := iniEntryFor(extName, !zend)
		if err := s.Write(iniPath, alt); err != nil {
			return "", err
		}
		if ok, phpErr2 := moduleLoads(phpBin, extName); !ok {
			s.Remove(iniPath, alt)
			detail := phpErr2
			if detail == "" {
				detail = phpErr
			}
			detail = strings.TrimSpace(detail)
			if detail == "" {
				// PHP starts cleanly but never saw the directive — name the
				// ini file it actually reads so a path mismatch diagnoses
				// itself.
				detail = iniDiagnostic(phpBin)
			}
			return "", fmt.Errorf("extension %s built and installed but does not load (PHP reports: %s)", extName, detail)
		}
		return alt, nil
	}
	return entry, nil
}

// Resolve returns the php.ini path PHP actually loads: query php-config
// --ini-path (the directory) and append php.ini. Falls back to
// <prefix>/etc/php.ini when php-config fails or answers nothing. This mirrors
// the assembler's getPHPIniDir semantics; phpv-built PHP uses <prefix>/lib,
// NOT <prefix>/etc.
func (s *Service) Resolve(prefix, phpConfigBin string) string {
	out, err := exec.Command(phpConfigBin, "--ini-path").Output()
	if err == nil {
		if dir := strings.TrimSpace(string(out)); dir != "" {
			return filepath.Join(dir, "php.ini")
		}
	}
	return filepath.Join(prefix, "etc", "php.ini")
}

// Write appends the line to php.ini unless it is already present.
func (s *Service) Write(iniPath, line string) error {
	if data, err := os.ReadFile(iniPath); err == nil {
		for _, l := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(l) == line {
				return nil
			}
		}
	}
	f, err := os.OpenFile(iniPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open php.ini: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("write php.ini: %w", err)
	}
	return nil
}

// Remove strips an exact line from php.ini (best effort).
func (s *Service) Remove(iniPath, line string) {
	data, err := os.ReadFile(iniPath)
	if err != nil {
		return
	}
	var kept []string
	for _, l := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(l) != line {
			kept = append(kept, l)
		}
	}
	os.WriteFile(iniPath, []byte(strings.Join(kept, "\n")), 0644)
}

// iniEntryFor returns the php.ini load directive for an extension.
func iniEntryFor(extName string, zend bool) string {
	if zend {
		return "zend_extension=" + extName + ".so"
	}
	return "extension=" + extName + ".so"
}

// moduleLoads runs php -m and reports whether extName is listed and PHP
// started cleanly. stderr is returned for error reporting.
func moduleLoads(phpBin, extName string) (bool, string) {
	cmd := exec.Command(phpBin, "-m")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return false, stderr.String()
	}
	for _, mod := range strings.Split(string(out), "\n") {
		if strings.EqualFold(strings.TrimSpace(mod), extName) {
			return true, ""
		}
	}
	return false, stderr.String()
}

// iniDiagnostic captures php --ini output so a "module does not load"
// failure can name the file PHP actually reads.
func iniDiagnostic(phpBin string) string {
	out, err := exec.Command(phpBin, "--ini").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
